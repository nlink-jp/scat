package slack

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/nlink-jp/scat/internal/export"
	"github.com/nlink-jp/scat/internal/util"
)

// ExportLog performs the entire export operation for Slack.
func (p *Provider) ExportLog(opts export.Options) (*export.ExportedLog, error) {
	var exportedMessages []export.ExportedMessage
	userCache := make(map[string]string)
	var userCacheMux sync.Mutex

	channelID, err := p.ResolveChannelID(opts.ChannelName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve channel ID for \"%s\": %w", opts.ChannelName, err)
	}

	// Fetch main channel messages and process threads
	historyCursor := ""
	for {
		historyResp, err := p.getConversationHistory(channelID, opts, historyCursor)
		if err != nil {
			return nil, err
		}

		for _, msg := range historyResp.Messages {
			// If the message has replies, fetch the entire thread.
			// We process threads first to avoid adding the parent message twice.
			if msg.ReplyCount > 0 {
				threadMessages, err := p.fetchAllReplies(channelID, msg.Timestamp, userCache, &userCacheMux, opts)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not fetch replies for thread %s: %v\n", msg.Timestamp, err)
					continue // Skip this thread on error
				}
				exportedMessages = append(exportedMessages, threadMessages...)
			} else if msg.ThreadTimestamp == "" {
				// This is a regular message (not a reply, not a thread parent).
				exportedMsg, err := p.buildExportedMessage(msg, userCache, &userCacheMux, opts)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not process message %s: %v\n", msg.Timestamp, err)
					continue
				}
				exportedMessages = append(exportedMessages, *exportedMsg)
			}
			// Note: Replies that are also broadcast to the channel are handled
			// when their parent thread is processed. They are not processed here.
		}

		if !historyResp.HasMore {
			break
		}
		historyCursor = historyResp.ResponseMetadata.NextCursor
	}

	// Sort messages by timestamp ascending
	sort.Slice(exportedMessages, func(i, j int) bool {
		return exportedMessages[i].TimestampUnix < exportedMessages[j].TimestampUnix
	})

	return &export.ExportedLog{
		ExportTimestamp: time.Now().UTC().Format(time.RFC3339),
		ChannelName:     opts.ChannelName,
		Messages:        exportedMessages,
	},
	nil
}

// fetchAllReplies fetches all messages in a specific thread using pagination.
func (p *Provider) fetchAllReplies(channelID, threadTS string, userCache map[string]string, userCacheMux *sync.Mutex, opts export.Options) ([]export.ExportedMessage, error) {
	var allReplies []export.ExportedMessage
	repliesCursor := ""
	for {
		repliesResp, err := p.getConversationReplies(channelID, threadTS, repliesCursor)
		if err != nil {
			return nil, err
		}

		for _, msg := range repliesResp.Messages {
			exportedMsg, err := p.buildExportedMessage(msg, userCache, userCacheMux, opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not process reply message %s: %v\n", msg.Timestamp, err)
				continue
			}
			allReplies = append(allReplies, *exportedMsg)
		}

		if !repliesResp.HasMore {
			break
		}
		repliesCursor = repliesResp.ResponseMetadata.NextCursor
	}
	return allReplies, nil
}

// buildExportedMessage constructs an ExportedMessage from a Slack message.
func (p *Provider) buildExportedMessage(msg message, userCache map[string]string, userCacheMux *sync.Mutex, opts export.Options) (*export.ExportedMessage, error) {
	var userID, postType, userName string

	if msg.SubType == "bot_message" {
		userName = msg.Username
		if userName == "" && msg.BotID != "" {
			userName = fmt.Sprintf("bot:%s", msg.BotID)
		}
		userID = msg.BotID
		postType = "bot"
	} else {
		var err error
		userName, err = p.resolveUserName(msg.UserID, userCache, userCacheMux)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not resolve user %s: %v\n", msg.UserID, err)
		}
		userID = msg.UserID
		postType = "user"
	}

	files, err := p.handleAttachedFiles(msg.Files, opts.OutputDir, opts.IncludeFiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not process files for message %s: %v\n", msg.Timestamp, err)
	}

	resolvedText, err := p.resolveMentions(msg.Text, userCache, userCacheMux)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not resolve mentions in message %s: %v\n", msg.Timestamp, err)
		resolvedText = msg.Text
	}

	rfc3339Time, err := util.ToRFC3339(msg.Timestamp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not parse timestamp %s: %v\n", msg.Timestamp, err)
		rfc3339Time = ""
	}

	isReply := msg.ThreadTimestamp != "" && msg.ThreadTimestamp != msg.Timestamp
	exportedMsg := &export.ExportedMessage{
		UserID:              userID,
		UserName:            userName,
		PostType:            postType,
		Timestamp:           rfc3339Time,
		TimestampUnix:       msg.Timestamp,
		Text:                resolvedText,
		Files:               files,
		ThreadTimestampUnix: msg.ThreadTimestamp,
		IsReply:             isReply,
	}
	return exportedMsg, nil
}

func (p *Provider) handleAttachedFiles(files []file, outputDir string, download bool) ([]export.ExportedFile, error) {
	var exportedFiles []export.ExportedFile
	for _, f := range files {
		exportedFile := export.ExportedFile{
			ID:       f.ID,
			Name:     f.Name,
			Mimetype: f.Mimetype,
		}
		if download && f.URLPrivateDownload != "" {
			safeFilename := filepath.Base(f.Name)
			localPath := filepath.Join(outputDir, f.ID+"_"+safeFilename)
			fileData, err := p.sendRequest("GET", f.URLPrivateDownload, nil, "")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not download file %s: %v\n", f.Name, err)
				continue
			}
			if err := os.WriteFile(localPath, fileData, 0600); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not save file %s: %v\n", f.Name, err)
				continue
			}
			exportedFile.LocalPath = localPath
		}
		exportedFiles = append(exportedFiles, exportedFile)
	}
	return exportedFiles, nil
}
