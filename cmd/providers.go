package cmd

import (
	"fmt"

	"github.com/nlink-jp/scat/internal/appcontext"
	"github.com/nlink-jp/scat/internal/config"
	"github.com/nlink-jp/scat/internal/provider"
	"github.com/nlink-jp/scat/internal/provider/mock"
	"github.com/nlink-jp/scat/internal/provider/slack"
	"github.com/nlink-jp/scat/internal/provider/testprovider"
)

// providerFactory defines the function signature for creating a new provider.Interface.
type providerFactory func(p config.Profile, ctx appcontext.Context) (provider.Interface, error)

// providerRegistry holds the mapping from a provider name to its factory function.
var providerRegistry = map[string]providerFactory{
	"slack": slack.NewProvider,
	"mock":  mock.NewProvider,
	"test":  testprovider.NewProvider,
}

// GetProvider retrieves a provider instance based on the provider name in the profile.
func GetProvider(ctx appcontext.Context, p config.Profile) (provider.Interface, error) {
	factory, ok := providerRegistry[p.Provider]
	if !ok {
		return nil, fmt.Errorf("unknown provider: '%s'", p.Provider)
	}
	return factory(p, ctx)
}

// requireCLIMode returns an error if the app is running in server mode.
// Use this to guard commands that manage the config file (profile subcommands, config init).
func requireCLIMode(appCtx appcontext.Context) error {
	if appCtx.ServerMode {
		return fmt.Errorf("this command is not available in server mode (SCAT_MODE=server)")
	}
	return nil
}
