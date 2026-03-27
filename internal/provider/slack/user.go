package slack

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/nlink-jp/scat/internal/provider"
)

// populateUserCache fetches all users and populates the userIDCache.
func (p *Provider) populateUserCache() error {
	users, err := p.getUsers()
	if err != nil {
		return fmt.Errorf("failed to get users for cache: %w", err)
	}

	p.userIDCache = make(map[string]string)
	for _, user := range users {
		p.userIDCache[user.Name] = user.ID
		if p.Context.Debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Caching user: Name=%s, ID=%s\n", user.Name, user.ID)
		}
	}

	if p.Context.Debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] User cache populated with %d users.\n", len(p.userIDCache))
	}
	return nil
}

// ResolveUserID finds a user ID for a given user name.
// It checks the cache first, and repopulates it if the user is not found.
func (p *Provider) ResolveUserID(userName string) (string, error) {
	cleanUserName := strings.TrimPrefix(userName, "@")

	id, ok := p.userIDCache[cleanUserName]
	if ok {
		return id, nil
	}

	// If not found, refresh the cache and try again.
	if p.Context.Debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] User '%s' not found in cache, repopulating...\n", cleanUserName)
	}
	if err := p.populateUserCache(); err != nil {
		return "", fmt.Errorf("failed to repopulate user cache: %w", err)
	}

	id, ok = p.userIDCache[cleanUserName]
	if !ok {
		return "", fmt.Errorf("user '%s' not found", userName)
	}

	return id, nil
}

// ListUsers returns all non-bot, non-deleted users in the workspace.
func (p *Provider) ListUsers() ([]provider.UserInfo, error) {
	users, err := p.getUsers()
	if err != nil {
		return nil, err
	}
	result := make([]provider.UserInfo, 0, len(users))
	for _, u := range users {
		result = append(result, provider.UserInfo{ID: u.ID, Name: u.Name})
	}
	return result, nil
}

// getUsers fetches all non-bot, non-deleted users from the workspace.
func (p *Provider) getUsers() ([]struct{ ID, Name string }, error) {
	if p.Context.Debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Fetching users by calling users.list...\n")
	}
	var allUsers []struct{ ID, Name string }
	cursor := ""

	for {
		url := fmt.Sprintf("%s?cursor=%s&limit=200", usersListURL, cursor)
		body, err := p.sendRequest("GET", url, nil, "")
		if err != nil {
			return nil, err
		}

		var listResp usersListResponse
		if err := json.Unmarshal(body, &listResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal users.list response: %w", err)
		}

		if !listResp.Ok {
			return nil, fmt.Errorf("slack API error on users.list: %s", listResp.Error)
		}

		for _, user := range listResp.Members {
			if !user.IsBot && !user.Deleted {
				name := user.Profile.DisplayName
				if name == "" {
					name = user.Name
				}
				allUsers = append(allUsers, struct{ ID, Name string }{ID: user.ID, Name: name})
			}
		}

		cursor = listResp.ResponseMetadata.NextCursor
		if cursor == "" {
			break
		}
	}
	return allUsers, nil
}