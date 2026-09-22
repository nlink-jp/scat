package cmd

import (
	"errors"
	"fmt"
	"github.com/nlink-jp/scat/internal/export"
	"github.com/nlink-jp/scat/internal/slack"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func (a *app) uploadCommand() *cobra.Command {
	var file, filename, channel, user, comment, thread string
	var dry bool
	c := &cobra.Command{Use: "upload --file <path|->", Short: "Upload a file to Slack", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		ch, err := a.destination(channel, user)
		if err != nil {
			return err
		}
		if err = slack.ValidateThread(thread); err != nil {
			return err
		}
		if file == "" {
			return errors.New("--file is required (use - for stdin)")
		}
		var reader io.Reader = c.InOrStdin()
		limit := a.selected.Limits.MaxFileSizeBytes
		name := filename
		if file != "-" {
			// #nosec G304 -- Explicit operator-selected upload path; regular-file and size checks follow.
			f, e := os.Open(file)
			if e != nil {
				return e
			}
			defer f.Close()
			info, e := f.Stat()
			if e != nil {
				return e
			}
			if !info.Mode().IsRegular() {
				return errors.New("upload source must be a regular file")
			}
			reader = f
			if name == "" {
				name = filepath.Base(file)
			}
		} else {
			if name == "" {
				return errors.New("stdin upload requires --filename")
			}
			if n := a.selected.Limits.MaxStdinSizeBytes; n > 0 && (limit == 0 || n < limit) {
				limit = n
			}
		}
		if strings.ContainsAny(name, "/\\\x00\r\n") || name == "." || name == ".." {
			return errors.New("--filename must be a basename")
		}
		snapshot, err := slack.Stage(c.Context(), reader, limit)
		if err != nil {
			return err
		}
		defer snapshot.Close()
		if err = c.Context().Err(); err != nil {
			return err
		}
		if dry {
			fmt.Fprintf(c.ErrOrStderr(), "Dry run: upload %d bytes; destination checked locally, bot identity unverified\n", snapshot.Size)
			return nil
		}
		r, err := a.client.Upload(c.Context(), slack.UploadOptions{Channel: ch, User: user, Filename: name, Comment: comment, Thread: thread, Source: snapshot})
		if err != nil {
			return err
		}
		ids := []string{}
		for _, f := range r.Files {
			ids = append(ids, f.ID)
		}
		return a.result(c, r, strings.Join(ids, "\n"))
	}}
	f := c.Flags()
	f.StringVarP(&file, "file", "f", "", "Input path or -")
	f.StringVar(&filename, "filename", "", "Uploaded basename (required for stdin)")
	f.StringVarP(&channel, "channel", "c", "", "Channel name or ID")
	f.StringVar(&user, "user", "", "User name or ID for DM")
	f.StringVarP(&comment, "comment", "m", "", "Initial comment")
	f.StringVar(&thread, "thread", "", "Parent timestamp")
	f.BoolVar(&dry, "dry-run", false, "Validate and stage input without calling Slack")
	return c
}
func (a *app) channelCommand() *cobra.Command {
	c := &cobra.Command{Use: "channel", Short: "List, export and manage channels"}
	list := &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		rows, e := a.client.Channels(c.Context())
		if e != nil {
			return e
		}
		if a.json {
			return a.result(c, rows, "")
		}
		for _, r := range rows {
			if _, e = fmt.Fprintf(c.OutOrStdout(), "%s\t#%s\n", r.ID, r.Name); e != nil {
				return e
			}
		}
		return nil
	}}
	var output, start, end, dir, format string
	exp := &cobra.Command{Use: "export <channel>", Short: "Export selected parents and complete threads", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, args []string) error {
		if format != "json" && format != "text" {
			return errors.New("export --format must be json or text")
		}
		if a.json && format != "json" {
			return errors.New("--json cannot be combined with --format text")
		}
		e := export.Exporter{Source: a.client, Now: a.deps.Now, Warn: func(s string) { a.warn(c, s) }}
		data, err := e.Run(c.Context(), export.Request{Channel: args[0], Start: start, End: end, SaveDir: dir})
		if err != nil {
			return err
		}
		return export.Write(c.Context(), c.OutOrStdout(), output, format, data)
	}}
	f := exp.Flags()
	f.StringVar(&output, "output", "-", "Output file or - (atomic replacement)")
	f.StringVar(&start, "start", "", "Exclusive parent-selection start (RFC3339)")
	f.StringVar(&end, "end", "", "Exclusive parent-selection end (RFC3339)")
	f.StringVar(&dir, "save-dir", "", "Download attachments into this directory")
	f.StringVar(&format, "format", "json", "json or text")
	var topic, description string
	var private, dry bool
	var invite []string
	create := &cobra.Command{Use: "create <name>", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, args []string) error {
		if !regexp.MustCompile(`^[a-z0-9_-]{1,80}$`).MatchString(args[0]) {
			return errors.New("channel name must be 1-80 lowercase letters, digits, hyphens or underscores")
		}
		if dry {
			fmt.Fprintln(c.ErrOrStderr(), "Dry run: channel creation validated locally; invitee names unresolved")
			return nil
		}
		r, e := a.client.Create(c.Context(), slack.CreateOptions{Name: args[0], Topic: topic, Description: description, Private: private, Invite: invite})
		if e != nil {
			return e
		}
		return a.result(c, map[string]string{"id": r.ID, "name": r.Name}, r.ID)
	}}
	f = create.Flags()
	f.StringVar(&topic, "topic", "", "Channel topic")
	f.StringVar(&description, "description", "", "Channel purpose")
	f.BoolVar(&private, "private", false, "Private channel")
	f.StringSliceVar(&invite, "invite", nil, "Users or groups to invite")
	f.BoolVar(&dry, "dry-run", false, "Validate locally without calling Slack")
	var inviteDry bool
	inv := &cobra.Command{Use: "invite <channel> <user-or-group>...", Args: cobra.MinimumNArgs(2), RunE: func(c *cobra.Command, args []string) error {
		for _, s := range args {
			if strings.TrimSpace(s) == "" {
				return errors.New("channel and invitees must be nonempty")
			}
		}
		if inviteDry {
			fmt.Fprintln(c.ErrOrStderr(), "Dry run: invitation validated locally; names unresolved")
			return nil
		}
		channel, users, e := a.client.Invite(c.Context(), args[0], args[1:])
		if e != nil {
			return e
		}
		if a.json {
			return a.result(c, map[string]any{"channel": channel, "users": users}, "")
		}
		a.info(c, "Invitation completed")
		return nil
	}}
	inv.Flags().BoolVar(&inviteDry, "dry-run", false, "Validate locally without calling Slack")
	c.AddCommand(list, exp, create, inv)
	return c
}
func (a *app) userCommand() *cobra.Command {
	c := &cobra.Command{Use: "user", Short: "List workspace users"}
	c.AddCommand(&cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		rows, e := a.client.Users(c.Context())
		if e != nil {
			return e
		}
		if a.json {
			return a.result(c, rows, "")
		}
		for _, r := range rows {
			if _, e = fmt.Fprintf(c.OutOrStdout(), "%s\t%s\t%s\n", r.ID, r.Name, r.DisplayName()); e != nil {
				return e
			}
		}
		return nil
	}})
	return c
}
