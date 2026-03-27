package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/nlink-jp/scat/internal/provider"
)

func (p *Provider) PostFile(opts provider.PostFileOptions) error {
	if p.Context.NoOp {
		fmt.Printf("---\n")
		fmt.Printf("Provider: slack\n")
		fmt.Printf("Action: Upload file %s\n", opts.FilePath)
		fmt.Printf("---------------------\n")
		return nil
	}

	// Step 1: Get Upload URL
	fi, err := os.Stat(opts.FilePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	getURLParams := url.Values{}
	getURLParams.Add("filename", opts.Filename)
	getURLParams.Add("length", fmt.Sprintf("%d", fi.Size()))

	respBody, err := p.sendRequest("GET", getUploadURLExternalURL+"?"+getURLParams.Encode(), nil, "")
	if err != nil {
		return fmt.Errorf("step 1 (getUploadURLExternal) failed: %w", err)
	}

	var getURLResp getUploadURLExternalResponse
	if err := json.Unmarshal(respBody, &getURLResp); err != nil {
		return fmt.Errorf("failed to unmarshal getUploadURLExternal response: %w", err)
	}
	if !getURLResp.Ok {
		return fmt.Errorf("slack API error on getUploadURLExternal: %s", getURLResp.Error)
	}

	// Step 2: Upload file to the provided URL
	file, err := os.Open(opts.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open file for upload: %w", err)
	}
	defer file.Close()

	uploadReq, err := http.NewRequest("POST", getURLResp.UploadURL, file)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}
	uploadReq.Header.Set("Content-Type", "application/octet-stream")

	httpClient := &http.Client{}
	uploadResp, err := httpClient.Do(uploadReq)
	if err != nil {
		return fmt.Errorf("step 2 (upload to url) failed: %w", err)
	}
	defer uploadResp.Body.Close()
	if uploadResp.StatusCode != 200 {
		body, _ := io.ReadAll(uploadResp.Body)
		return fmt.Errorf("upload to url failed with status %d: %s", uploadResp.StatusCode, string(body))
	}

	// Step 3: Complete the upload
	var channelID, destinationName string

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

	completePayload := completeUploadExternalPayload{
		Files:          []fileInfo{{ID: getURLResp.FileID}},
		ChannelID:      channelID,
		InitialComment: opts.Comment,
	}
	completePayloadBytes, err := json.Marshal(completePayload)
	if err != nil {
		return fmt.Errorf("failed to marshal completeUploadExternal payload: %w", err)
	}

	_, err = p.sendRequest("POST", completeUploadExternalURL, bytes.NewBuffer(completePayloadBytes), "application/json; charset=utf-8")
	if err != nil {
		// Check if the error is 'not_in_channel' and retry if so.
		if opts.TargetUserID == "" && strings.Contains(err.Error(), "not_in_channel") {
			if !p.Context.Silent {
				fmt.Fprintf(os.Stderr, "Bot not in channel '%s'. Attempting to join...\n", destinationName)
			}
			if joinErr := p.joinChannel(channelID); joinErr != nil {
				return fmt.Errorf("failed to join channel '%s': %w", destinationName, joinErr)
			}
			if !p.Context.Silent {
				fmt.Fprintf(os.Stderr, "Successfully joined channel '%s'. Retrying file upload completion...\n", destinationName)
			}
			// Retry completing the upload after joining.
			_, retryErr := p.sendRequest("POST", completeUploadExternalURL, bytes.NewBuffer(completePayloadBytes), "application/json; charset=utf-8")
			if retryErr != nil {
				return fmt.Errorf("step 3 (completeUploadExternal) failed on retry: %w", retryErr)
			}
			return nil // Success on retry
		}
		return fmt.Errorf("step 3 (completeUploadExternal) failed: %w", err)
	}

	return nil
}
