package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/nlink-jp/scat/internal/provider"
)

func (p *Provider) PostMessage(opts provider.PostMessageOptions) error {
	if p.Context.Debug {
		fmt.Fprintln(os.Stderr, "[DEBUG] PostMessage called with Debug mode ON.")
	}

	var channelID, destinationName string
	var err error

	switch {
	case opts.TargetUserID != "":
		destinationName = opts.TargetUserID
		var userID string
		// Check if the provided user string is a user ID or a mention name
		if strings.HasPrefix(opts.TargetUserID, "U") || strings.HasPrefix(opts.TargetUserID, "W") {
			userID = opts.TargetUserID
		} else {
			userID, err = p.ResolveUserID(opts.TargetUserID)
			if err != nil {
				return err
			}
		}
		channelID, err = p.openDMChannel(userID)
		if err != nil {
			return fmt.Errorf("failed to open DM channel with user %s: %w", opts.TargetUserID, err)
		}

	case opts.TargetChannel != "":
		destinationName = opts.TargetChannel
		channelID, err = p.ResolveChannelID(opts.TargetChannel)
		if err != nil {
			return err
		}

	default:
		destinationName = p.Profile.Channel
		if destinationName == "" {
			return fmt.Errorf("no channel or user specified; please set a default channel in the profile or use the --channel or --user flag")
		}
		channelID, err = p.ResolveChannelID(p.Profile.Channel)
		if err != nil {
			return err
		}
	}

	username := p.Profile.Username
	if opts.OverrideUsername != "" {
		username = opts.OverrideUsername
	}

	payload := messagePayload{
		Channel:   channelID,
		Text:      opts.Text,
		Username:  username,
		IconEmoji: opts.IconEmoji,
		Blocks:    opts.Blocks,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	// Attempt to post message
	_, err = p.sendRequest("POST", postMessageURL, bytes.NewBuffer(jsonPayload), "application/json; charset=utf-8")
	if err != nil {
		// Check if the error is 'not_in_channel' (only applicable to channels, not DMs)
		if opts.TargetUserID == "" && strings.Contains(err.Error(), "not_in_channel") {
			if !p.Context.Silent {
				fmt.Fprintf(os.Stderr, "Bot not in channel \"%s\". Attempting to join...\n", destinationName)
			}
			if joinErr := p.joinChannel(channelID); joinErr != nil {
				return fmt.Errorf("failed to join channel \"%s\": %w", destinationName, joinErr)
			}
			if !p.Context.Silent {
				fmt.Fprintf(os.Stderr, "Successfully joined channel \"%s\". Retrying post...\n", destinationName)
			}
			// Retry post after joining
			_, retryErr := p.sendRequest("POST", postMessageURL, bytes.NewBuffer(jsonPayload), "application/json; charset=utf-8")
			return retryErr
		}
		return err // Return original error if not applicable for retry
	}

	return nil
}