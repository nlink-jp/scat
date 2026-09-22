package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/nlink-jp/scat/internal/config"
	"github.com/nlink-jp/scat/internal/slack"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var version = "dev"

// Dependencies are invocation-scoped. No global configuration, HTTP client or test provider.
type Dependencies struct {
	HTTP   *http.Client
	Now    func() time.Time
	Wait   func(context.Context, time.Duration) error
	Getenv func(string) string
	Prompt func(io.Writer) (string, error)
	Ticks  <-chan time.Time
}
type app struct {
	deps                       Dependencies
	cfg                        *config.Config
	selected                   config.Profile
	configPath, profile        string
	quiet, debug, json, server bool
	client                     *slack.Client
}

func NewCommand(d Dependencies) *cobra.Command {
	if d.Getenv == nil {
		d.Getenv = os.Getenv
	}
	if d.Prompt == nil {
		d.Prompt = func(w io.Writer) (string, error) {
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				return "", errors.New("token setup requires a terminal; services use SCAT_TOKEN")
			}
			fmt.Fprint(w, "Bot token (hidden): ")
			b, e := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(w)
			return string(b), e
		}
	}
	a := &app{deps: d}
	root := &cobra.Command{Use: "scat", Short: "Slack CLI for services using bot credentials", Version: version, SilenceUsage: true, SilenceErrors: true}
	f := root.PersistentFlags()
	f.StringVar(&a.configPath, "config", "", "Configuration path")
	f.StringVarP(&a.profile, "profile", "p", "", "Bot profile")
	f.BoolVarP(&a.quiet, "quiet", "q", false, "Suppress informational stderr")
	f.BoolVar(&a.debug, "debug", false, "Enable safe diagnostic output")
	f.BoolVar(&a.json, "json", false, "Machine-readable results")
	f.BoolP("version", "V", false, "Print version")
	root.SetFlagErrorFunc(func(c *cobra.Command, e error) error {
		for old, newFlag := range map[string]string{"--provider": "remove --provider (Slack only)", "--noop": "use --dry-run", "--silent": "use --quiet", "--iconemoji": "use --icon-emoji", "--filetype": "remove --filetype (unused by Slack)", "--output-files": "use --save-dir", "--start-time": "use --start", "--end-time": "use --end", "--output-format": "use --format"} {
			if strings.Contains(e.Error(), old) {
				return errors.New("scat v2 migration: " + newFlag)
			}
		}
		return e
	})
	root.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		if d.Getenv("SCAT_PROVIDER") != "" {
			return errors.New("SCAT_PROVIDER was removed; unset it for Slack-only scat v2")
		}
		var err error
		a.server, err = config.ServerMode(d.Getenv)
		if err != nil {
			return err
		}
		local := c.Annotations["local"] == "true"
		if a.server {
			if c.Flags().Changed("config") || c.Flags().Changed("profile") || local {
				return errors.New("server mode rejects config/profile flags and local management commands")
			}
			a.cfg, err = config.FromEnv(d.Getenv)
		} else {
			a.configPath, err = config.GetConfigPath(a.configPath)
			if err == nil {
				a.cfg, err = config.Load(a.configPath)
				if os.IsNotExist(err) {
					a.cfg = nil
					err = nil
				}
			}
			if info, e := os.Stat(a.configPath); e == nil && info.Mode().Perm()&0077 != 0 {
				a.warn(c, "credential file permissions are too broad; use chmod 600")
			}
		}
		if err != nil {
			return err
		}
		if local {
			if a.json {
				return errors.New("--json is not supported for local management")
			}
			return nil
		}
		if a.cfg == nil {
			return errors.New("configuration file not found; run scat config init or set SCAT_MODE=server")
		}
		name := a.profile
		if name == "" {
			name = a.cfg.CurrentProfile
		}
		p, ok := a.cfg.Profiles[name]
		if !ok {
			return errors.New("profile not found")
		}
		a.selected = p
		a.client = slack.New(p.Token, slack.Options{HTTP: d.HTTP, Now: d.Now, Wait: d.Wait, Warn: func(s string) { a.warn(c, s) }, MaxFileSize: p.Limits.MaxFileSizeBytes})
		if a.debug {
			a.info(c, "using selected bot configuration")
		}
		return nil
	}
	root.AddCommand(a.postCommand(), a.uploadCommand(), a.channelCommand(), a.userCommand(), a.configCommand(), a.profileCommand(), a.cacheCommand())
	root.AddCommand(&cobra.Command{Use: "export", Hidden: true, DisableFlagParsing: true, RunE: func(*cobra.Command, []string) error {
		return errors.New("scat v2: use channel export <channel> --start/--end --save-dir --format")
	}})
	return root
}
func (a *app) warn(c *cobra.Command, s string) { fmt.Fprintln(c.ErrOrStderr(), "Warning:", s) }
func (a *app) info(c *cobra.Command, s string) {
	if !a.quiet {
		fmt.Fprintln(c.ErrOrStderr(), s)
	}
}
func (a *app) result(c *cobra.Command, v any, text string) error {
	if a.json {
		return json.NewEncoder(c.OutOrStdout()).Encode(v)
	}
	_, err := fmt.Fprintln(c.OutOrStdout(), text)
	return err
}
func (a *app) destination(channel, user string) (string, error) {
	if user != "" {
		if channel != "" {
			return "", errors.New("--user and --channel are mutually exclusive")
		}
		return "", nil
	}
	if channel == "" {
		channel = a.selected.Channel
	}
	if channel == "" {
		return "", errors.New("channel or --user is required")
	}
	return channel, nil
}
func Execute() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	return NewCommand(Dependencies{}).ExecuteContext(ctx)
}
