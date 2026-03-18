package slack

import (
	"fmt"
	"os"
	"strings"

	"github.com/magifd2/scat/internal/provider"
)

// ResolveChannelID ensures a channel ID is returned for a given name.
// It first checks the local cache. If the name is not found, it refreshes
// the cache from the API and checks again.
func (p *Provider) ResolveChannelID(name string) (string, error) {
	// First, try to get the ID from the existing cache.
	id, err := p.getCachedChannelID(name)
	if err == nil {
		return id, nil // Found in cache
	}

	// If not found, refresh the cache from the API.
	if p.Context.Debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Channel \"%s\" not in cache. Refreshing...\n", name)
	}
	if refreshErr := p.populateChannelCache(); refreshErr != nil {
		return "", fmt.Errorf("failed to refresh channel list: %w", refreshErr)
	}

	// Try checking the cache again after refreshing.
	id, err = p.getCachedChannelID(name)
	if err == nil {
		return id, nil // Found after refresh
	}

	// If it's still not found, the channel likely doesn't exist.
	return "", fmt.Errorf("channel \"%s\" not found after refreshing cache", name)
}

// getCachedChannelID is a helper that only checks the local cache.
func (p *Provider) getCachedChannelID(name string) (string, error) {
	name = strings.TrimPrefix(name, "#")

	if p.channelIDCache != nil {
		if id, ok := p.channelIDCache[name]; ok {
			return id, nil
		}
	}

	// Also consider the case where the name is already a valid ID.
	if strings.HasPrefix(name, "C") || strings.HasPrefix(name, "G") || strings.HasPrefix(name, "D") {
		return name, nil
	}

	return "", fmt.Errorf("not found in cache")
}

func (p *Provider) ListChannels() ([]provider.Channel, error) {
	// Ensure the cache is populated before listing.
	if p.channelIDCache == nil {
		if err := p.populateChannelCache(); err != nil {
			return nil, err
		}
	}
	var channels []provider.Channel
	for name, id := range p.channelIDCache {
		channels = append(channels, provider.Channel{ID: id, Name: name})
	}
	return channels, nil
}

// InviteToChannel invites users or user groups to an existing channel.
func (p *Provider) InviteToChannel(opts provider.InviteToChannelOptions) error {
	channelID, err := p.ResolveChannelID(opts.Channel)
	if err != nil {
		return fmt.Errorf("failed to resolve channel: %w", err)
	}

	userIDsToInvite := make(map[string]struct{})
	for _, invitee := range opts.Invitees {
		userID, err := p.ResolveUserID(invitee)
		if err == nil {
			userIDsToInvite[userID] = struct{}{}
			continue
		}

		userGroupID, err := p.ResolveUserGroupID(invitee)
		if err == nil {
			userGroupUserIDs, err := p.getUserGroupUsers(userGroupID)
			if err != nil {
				return fmt.Errorf("failed to get users for user group '%s': %w", invitee, err)
			}
			for _, ugUserID := range userGroupUserIDs {
				userIDsToInvite[ugUserID] = struct{}{}
			}
			continue
		}

		return fmt.Errorf("could not resolve '%s' as a user or user group", invitee)
	}

	var finalUserIDs []string
	for id := range userIDsToInvite {
		finalUserIDs = append(finalUserIDs, id)
	}

	if len(finalUserIDs) == 0 {
		return nil
	}

	return p.inviteUsersToChannel(channelID, finalUserIDs)
}

// CreateChannel creates a new channel.
func (p *Provider) CreateChannel(opts provider.CreateChannelOptions) (string, error) {
	channelID, err := p.createConversation(opts)
	if err != nil {
		return "", err
	}

	// Invite users and members of user groups if any are specified.
	if len(opts.Invitees) > 0 {
		userIDsToInvite := make(map[string]struct{})

		for _, invitee := range opts.Invitees {
			// Try to resolve as a user first.
			userID, err := p.ResolveUserID(invitee)
			if err == nil {
				userIDsToInvite[userID] = struct{}{}
				continue
			}

			// If not a user, try to resolve as a user group.
			userGroupID, err := p.ResolveUserGroupID(invitee)
			if err == nil {
				userGroupUserIDs, err := p.getUserGroupUsers(userGroupID)
				if err != nil {
					return "", fmt.Errorf("failed to get users for user group '%s': %w", invitee, err)
				}
				for _, ugUserID := range userGroupUserIDs {
					userIDsToInvite[ugUserID] = struct{}{}
				}
				continue
			}

			return "", fmt.Errorf("could not resolve '%s' as a user or user group", invitee)
		}

		// Convert map to slice for inviting.
		var finalUserIDs []string
		for id := range userIDsToInvite {
			finalUserIDs = append(finalUserIDs, id)
		}

		if len(finalUserIDs) > 0 {
			if err := p.inviteUsersToChannel(channelID, finalUserIDs); err != nil {
				return "", fmt.Errorf("failed to invite users: %w", err)
			}
		}
	}

	// Repopulate the channel cache since we've made a change.
	if err := p.populateChannelCache(); err != nil {
			if p.Context.Debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] Failed to repopulate channel cache after creation: %v\n", err)
			}
		// Don't fail the whole operation if the cache refresh fails, but log it.
	}

	return channelID, nil
}
