package cmd

import (
	"errors"
	"fmt"
	"github.com/nlink-jp/scat/internal/config"
	"github.com/spf13/cobra"
	"sort"
	"strconv"
	"strings"
)

func localCommand(use string, args cobra.PositionalArgs, run func(*cobra.Command, []string) error) *cobra.Command {
	return &cobra.Command{Use: use, Args: args, Annotations: map[string]string{"local": "true"}, RunE: run}
}
func (a *app) needConfig() error {
	if a.cfg == nil {
		return errors.New("configuration file not found; run scat config init")
	}
	return nil
}
func (a *app) save(c *cobra.Command) error {
	if err := a.cfg.Save(a.configPath); err != nil {
		return err
	}
	a.info(c, "Configuration saved")
	return nil
}
func (a *app) configCommand() *cobra.Command {
	c := &cobra.Command{Use: "config", Short: "Manage local bot configuration"}
	c.AddCommand(localCommand("init", cobra.NoArgs, func(c *cobra.Command, _ []string) error {
		if a.cfg != nil {
			a.info(c, "Configuration already exists")
			return nil
		}
		a.cfg = config.NewDefaultConfig()
		return a.save(c)
	}))
	return c
}
func (a *app) profileCommand() *cobra.Command {
	c := &cobra.Command{Use: "profile", Short: "Manage named bot profiles"}
	var channel, username string
	limits := config.NewDefaultLimits()
	add := localCommand("add <name>", cobra.ExactArgs(1), func(c *cobra.Command, args []string) error {
		if err := a.needConfig(); err != nil {
			return err
		}
		name := args[0]
		if !config.ValidName(name) {
			return errors.New("invalid profile name")
		}
		if _, ok := a.cfg.Profiles[name]; ok {
			return errors.New("profile already exists")
		}
		if err := limits.Validate(); err != nil {
			return err
		}
		token, e := a.deps.Prompt(c.ErrOrStderr())
		if e != nil {
			return e
		}
		if e = validateTokenInput(token); e != nil {
			return e
		}
		a.cfg.Profiles[name] = config.Profile{Token: token, Channel: channel, Username: username, Limits: limits}
		return a.save(c)
	})
	f := add.Flags()
	f.StringVar(&channel, "channel", "", "Default channel")
	f.StringVar(&username, "username", "", "Posting name")
	f.Int64Var(&limits.MaxFileSizeBytes, "limits-max-file-size-bytes", limits.MaxFileSizeBytes, "File limit, 0 means unlimited")
	f.Int64Var(&limits.MaxStdinSizeBytes, "limits-max-stdin-size-bytes", limits.MaxStdinSizeBytes, "Stdin limit, 0 means unlimited")
	list := localCommand("list", cobra.NoArgs, func(c *cobra.Command, _ []string) error {
		if e := a.needConfig(); e != nil {
			return e
		}
		names := []string{}
		for name := range a.cfg.Profiles {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, n := range names {
			mark := " "
			if n == a.cfg.CurrentProfile {
				mark = "*"
			}
			p := a.cfg.Profiles[n]
			if _, e := fmt.Fprintf(c.OutOrStdout(), "%s %s\t%s\n", mark, n, p.Channel); e != nil {
				return e
			}
		}
		return nil
	})
	use := localCommand("use <name>", cobra.ExactArgs(1), func(c *cobra.Command, args []string) error {
		if e := a.needConfig(); e != nil {
			return e
		}
		if _, ok := a.cfg.Profiles[args[0]]; !ok {
			return errors.New("profile not found")
		}
		a.cfg.CurrentProfile = args[0]
		return a.save(c)
	})
	remove := localCommand("remove <name>", cobra.ExactArgs(1), func(c *cobra.Command, args []string) error {
		if e := a.needConfig(); e != nil {
			return e
		}
		name := args[0]
		if name == "default" || name == a.cfg.CurrentProfile {
			return errors.New("cannot remove default or currently active profile")
		}
		if _, ok := a.cfg.Profiles[name]; !ok {
			return errors.New("profile not found")
		}
		delete(a.cfg.Profiles, name)
		return a.save(c)
	})
	set := localCommand("set <key> [value]", cobra.RangeArgs(1, 2), func(c *cobra.Command, args []string) error {
		if e := a.needConfig(); e != nil {
			return e
		}
		name := a.profile
		if name == "" {
			name = a.cfg.CurrentProfile
		}
		p, ok := a.cfg.Profiles[name]
		if !ok {
			return errors.New("profile not found")
		}
		key := args[0]
		if key == "provider" || key == "endpoint" {
			return errors.New("provider and endpoint were removed; scat accepts Slack bot profiles only")
		}
		if key == "token" {
			if len(args) != 1 {
				return errors.New("tokens must not be passed as command arguments")
			}
			token, e := a.deps.Prompt(c.ErrOrStderr())
			if e != nil {
				return e
			}
			if e = validateTokenInput(token); e != nil {
				return e
			}
			p.Token = token
		} else {
			if len(args) != 2 {
				return errors.New("value is required")
			}
			value := args[1]
			switch key {
			case "channel":
				p.Channel = value
			case "username":
				p.Username = value
			case "limits.max_file_size_bytes", "limits.max_stdin_size_bytes":
				n, e := strconv.ParseInt(value, 10, 64)
				if e != nil || n < 0 {
					return errors.New("limit must be a nonnegative integer")
				}
				if key == "limits.max_file_size_bytes" {
					p.Limits.MaxFileSizeBytes = n
				} else {
					p.Limits.MaxStdinSizeBytes = n
				}
			default:
				return errors.New("unknown configuration key")
			}
		}
		a.cfg.Profiles[name] = p
		return a.save(c)
	})
	c.AddCommand(add, list, use, remove, set)
	return c
}
func validateTokenInput(s string) error {
	if strings.TrimSpace(s) == "" || strings.ContainsAny(s, "\r\n") {
		return errors.New("a nonempty bot token is required")
	}
	if strings.HasPrefix(s, "xoxp-") || strings.HasPrefix(s, "xapp-") {
		return errors.New("user and app-level tokens are not supported")
	}
	return nil
}
func (a *app) cacheCommand() *cobra.Command {
	c := &cobra.Command{Use: "cache", Short: "Manage local cache"}
	c.AddCommand(localCommand("clear", cobra.NoArgs, func(c *cobra.Command, _ []string) error {
		a.info(c, "No persistent cache: scat caches lookups within an invocation only")
		return nil
	}))
	return c
}
