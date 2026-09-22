package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/nlink-jp/scat/internal/input"
	"github.com/nlink-jp/scat/internal/slack"
	"github.com/spf13/cobra"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

func readBounded(r io.Reader, limit int64) ([]byte, error) {
	if limit == 0 {
		return io.ReadAll(r)
	}
	b, e := io.ReadAll(io.LimitReader(r, limit))
	if e != nil {
		return nil, e
	}
	var extra [1]byte
	n, e := io.ReadFull(r, extra[:])
	if n > 0 {
		return nil, errors.New("input exceeds configured size limit")
	}
	if e != nil && e != io.EOF {
		return nil, e
	}
	return b, nil
}
func (a *app) postCommand() *cobra.Command {
	var channel, user, fromFile, format, username, icon, thread string
	var stream, tee, dry, unfurlLinks, unfurlMedia, mrkdwn bool
	c := &cobra.Command{Use: "post [text]", Short: "Post an argument, file or stdin", RunE: func(c *cobra.Command, args []string) error {
		ch, err := a.destination(channel, user)
		if err != nil {
			return err
		}
		if err = slack.ValidateThread(thread); err != nil {
			return err
		}
		if format != "text" && format != "blocks" && format != "payload" {
			return errors.New("--format must be text, blocks or payload")
		}
		if tee && (a.json || len(args) > 0 || fromFile != "" || format != "text") {
			return errors.New("--tee requires text stdin and cannot be combined with --json")
		}
		if stream && (dry || format != "text" || len(args) > 0 || fromFile != "") {
			return errors.New("--stream requires text stdin and cannot use --dry-run")
		}
		name := username
		if name == "" {
			name = a.selected.Username
		}
		send := func(text string) error {
			if err := c.Context().Err(); err != nil {
				return err
			}
			payload, e := slack.ParseContent(text, format)
			if e != nil {
				return e
			}
			for _, f := range []struct {
				key   string
				value bool
			}{{"unfurl-links", unfurlLinks}, {"unfurl-media", unfurlMedia}, {"mrkdwn", mrkdwn}} {
				if c.Flags().Changed(f.key) {
					payload[strings.ReplaceAll(f.key, "-", "_")] = f.value
				}
			}
			if dry {
				fmt.Fprintln(c.ErrOrStderr(), "Dry run: post input and destination validated locally; bot identity unverified")
				return nil
			}
			r, e := a.client.Post(c.Context(), slack.PostOptions{Channel: ch, User: user, Username: name, IconEmoji: icon, Thread: thread, Payload: payload})
			if e != nil {
				return e
			}
			if tee {
				return nil
			}
			return a.result(c, r, r.TS)
		}
		if stream {
			return streamText(c.Context(), c.InOrStdin(), c.OutOrStdout(), tee, a.selected.Limits.MaxStdinSizeBytes, a.deps.Ticks, send)
		}
		var data []byte
		switch {
		case len(args) > 0:
			data = []byte(strings.Join(args, " "))
		case fromFile != "":
			// #nosec G304 -- Explicit operator-selected input path; regular-file and size checks follow.
			f, e := os.Open(fromFile)
			if e != nil {
				return e
			}
			defer f.Close()
			info, e := f.Stat()
			if e != nil {
				return e
			}
			if !info.Mode().IsRegular() {
				return errors.New("--from-file must be a regular file")
			}
			data, err = readBounded(input.Reader{Context: c.Context(), Source: f}, a.selected.Limits.MaxFileSizeBytes)
		default:
			data, err = readBounded(input.Reader{Context: c.Context(), Source: c.InOrStdin()}, a.selected.Limits.MaxStdinSizeBytes)
		}
		if err != nil {
			return err
		}
		if !utf8.Valid(data) {
			return errors.New("post input must be UTF-8")
		}
		if tee {
			if _, err = c.OutOrStdout().Write(data); err != nil {
				return err
			}
		}
		return send(string(data))
	}}
	f := c.Flags()
	f.StringVarP(&channel, "channel", "c", "", "Channel name or ID")
	f.StringVar(&user, "user", "", "User name or ID for a DM")
	f.StringVar(&fromFile, "from-file", "", "Read a file")
	f.StringVar(&format, "format", "text", "text, blocks or payload")
	f.StringVar(&username, "username", "", "Posting name")
	f.StringVar(&icon, "icon-emoji", "", "Posting emoji")
	f.StringVar(&thread, "thread", "", "Parent timestamp")
	f.BoolVar(&stream, "stream", false, "Batch stdin every 3 seconds (max 4000 characters)")
	f.BoolVar(&tee, "tee", false, "Copy stdin to stdout, suppress result IDs")
	f.BoolVar(&dry, "dry-run", false, "Validate locally without calling Slack")
	f.BoolVar(&unfurlLinks, "unfurl-links", false, "Expand links")
	f.BoolVar(&unfurlMedia, "unfurl-media", false, "Expand media")
	f.BoolVar(&mrkdwn, "mrkdwn", true, "Interpret Slack formatting")
	return c
}

type runeRead struct {
	r    rune
	err  error
	size int
}

// streamText bounds queued input and pending text independently of source length.
func streamText(ctx context.Context, in io.Reader, out io.Writer, tee bool, limit int64, ticks <-chan time.Time, send func(string) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	events := make(chan runeRead, 1)
	// A blocking arbitrary Reader cannot be canceled by Go. The CLI exits on signal;
	// injected finite readers and all subsequent sends observe this context.
	go func() {
		r := bufio.NewReader(in)
		for {
			v, n, e := r.ReadRune()
			select {
			case events <- runeRead{v, e, n}:
			case <-ctx.Done():
				return
			}
			if e != nil {
				return
			}
		}
	}()
	if ticks == nil {
		t := time.NewTicker(3 * time.Second)
		defer t.Stop()
		ticks = t.C
	}
	pending := make([]rune, 0, 4000)
	var bytesRead int64
	flush := func() error {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("stream canceled with %d buffered characters unsent: %w", len(pending), err)
		}
		if len(pending) == 0 {
			return nil
		}
		text := string(pending)
		if strings.TrimSpace(text) == "" {
			pending = pending[:0]
			return nil
		}
		if err := send(text); err != nil {
			return err
		}
		pending = pending[:0]
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("stream canceled with %d buffered characters unsent: %w", len(pending), ctx.Err())
		case _, open := <-ticks:
			if !open {
				ticks = nil
				continue
			}
			if err := flush(); err != nil {
				return err
			}
		case e := <-events:
			if e.err == io.EOF {
				return flush()
			}
			if e.err != nil {
				return fmt.Errorf("stream input failed with %d buffered characters unsent: %w", len(pending), e.err)
			}
			if e.r == utf8.RuneError && e.size == 1 {
				return errors.New("stream input must be UTF-8")
			}
			if limit > 0 && int64(e.size) > limit-bytesRead {
				return errors.New("stdin exceeds configured size limit")
			}
			bytesRead += int64(e.size)
			if tee {
				if _, err := io.WriteString(out, string(e.r)); err != nil {
					return err
				}
			}
			pending = append(pending, e.r)
			if len(pending) == 4000 {
				if err := flush(); err != nil {
					return err
				}
			}
		}
	}
}
