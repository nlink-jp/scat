package slack

import (
	"fmt"
	"net/http"
	"os"

	"github.com/magifd2/scat/internal/appcontext"
	"github.com/magifd2/scat/internal/config"
	"github.com/magifd2/scat/internal/provider"
)

// Provider implements the provider.Interface for Slack.
type Provider struct {
	Profile          config.Profile
	Context          appcontext.Context
	httpClient       *http.Client
	channelIDCache   map[string]string
	userIDCache      map[string]string
	userGroupIDCache map[string]string
}

// NewProvider creates a new Slack Provider.
func NewProvider(p config.Profile, ctx appcontext.Context) (provider.Interface, error) {
	prov := &Provider{
		Profile:          p,
		Context:          ctx,
		httpClient:       &http.Client{},
		channelIDCache:   make(map[string]string),
		userIDCache:      make(map[string]string),
		userGroupIDCache: make(map[string]string),
	}

	// Best-effort attempt to populate caches on initialization.
	if err := prov.populateChannelCache(); err != nil {
		if ctx.Debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Failed to populate channel cache on init: %v\n", err)
		}
	}
	if err := prov.populateUserCache(); err != nil {
		if ctx.Debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Failed to populate user cache on init: %v\n", err)
		}
	}
	if err := prov.populateUserGroupCache(); err != nil {
		if ctx.Debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Failed to populate user group cache on init: %v\n", err)
		}
	}

	return prov, nil
}

// Capabilities returns the features supported by the Slack provider.
func (p *Provider) Capabilities() provider.Capabilities {
	return provider.Capabilities{
		CanListChannels:    true,
		CanListUsers:       true,
		CanPostFile:        true,
		CanUseIconEmoji:    true,
		CanExportLogs:      true,
		CanPostBlocks:      true,
		CanCreateChannel:   true,
		CanInviteToChannel: true,
	}
}
