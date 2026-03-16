package appcontext

import "github.com/magifd2/scat/internal/config"

// CtxKey is the key for the application context in a context.Context.
type CtxKeyType struct{}

var CtxKey = CtxKeyType{}

// Context holds application-wide execution settings.
type Context struct {
	Debug      bool
	NoOp       bool
	Silent     bool
	ConfigPath string        // Path to the config file (CLI mode only; empty in server mode)
	ServerMode bool          // true when SCAT_MODE=server; uses env vars only, ignores config file
	Config     *config.Config // Resolved at startup; nil if config file was not found in CLI mode
}

// NewContext creates a new application context.
func NewContext(debug, noOp, silent bool, configPath string, serverMode bool, cfg *config.Config) Context {
	return Context{
		Debug:      debug,
		NoOp:       noOp,
		Silent:     silent,
		ConfigPath: configPath,
		ServerMode: serverMode,
		Config:     cfg,
	}
}
