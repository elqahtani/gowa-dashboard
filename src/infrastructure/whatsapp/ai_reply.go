package whatsapp

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
)

// AIReplyHandler is the contract the AI Reply usecase implements. We keep it
// here as a tiny interface so the whatsapp package does not import the
// usecase/aireply package directly (avoids import cycles and lets cmd/ wire
// the dependency at startup).
//
// presence lets the service show a "typing..." indicator while the LLM is
// generating. state == "composing" starts/refreshes typing; "paused" stops.
type AIReplyHandler interface {
	HandleIncoming(
		ctx context.Context,
		deviceID, chatJID, senderJID, text string,
		send func(ctx context.Context, recipientJID, text string) (msgID string, ts time.Time, err error),
		presence func(state string),
	) bool
}

var (
	aiHandlerMu sync.RWMutex
	aiHandler   AIReplyHandler
)

// RegisterAIReplyHandler installs the orchestrator. Safe to call from cmd/
// during startup. Passing nil disables AI auto-reply at runtime.
func RegisterAIReplyHandler(h AIReplyHandler) {
	aiHandlerMu.Lock()
	defer aiHandlerMu.Unlock()
	aiHandler = h
}

func currentAIHandler() AIReplyHandler {
	aiHandlerMu.RLock()
	defer aiHandlerMu.RUnlock()
	return aiHandler
}

// handleAIAutoReply runs before the static auto-reply. Returns true if the AI
// took ownership of this message (a reply was sent OR the message was
// deliberately swallowed). The static auto-reply must be skipped in that
// case.
//
// Skip rules mirror auto_reply.go to keep behaviour aligned.
func handleAIAutoReply(ctx context.Context, evt *events.Message, client *whatsmeow.Client) bool {
	h := currentAIHandler()
	if h == nil || client == nil {
		return false
	}

	// Skip groups, broadcasts, self-messages.
	if utils.IsGroupJID(evt.Info.Chat.String()) || evt.Info.IsIncomingBroadcast() || evt.Info.IsFromMe {
		return false
	}
	if evt.Info.Chat.Server != types.DefaultUserServer && evt.Info.Chat.Server != types.HiddenUserServer {
		return false
	}
	source := evt.Info.SourceString()
	if strings.Contains(source, "broadcast") ||
		strings.HasSuffix(evt.Info.Chat.String(), "@broadcast") ||
		strings.HasPrefix(evt.Info.Chat.String(), "status@") {
		return false
	}

	text := extractIncomingText(evt)
	if text == "" {
		return false
	}

	if client.Store == nil || client.Store.ID == nil {
		return false
	}
	deviceID := client.Store.ID.ToNonAD().String()

	// WhatsApp delivers many events with @lid JIDs. The per-chat AI toggle
	// stores chat IDs in their resolved @s.whatsapp.net form (that's what
	// users see and what other parts of the system use), so normalise both
	// chat and sender before handing off — otherwise lookup misses and the
	// handler silently returns false.
	chatJID := NormalizeJIDFromLID(ctx, evt.Info.Chat, client).String()
	senderJID := NormalizeJIDFromLID(ctx, evt.Info.Sender, client).String()

	send := func(sctx context.Context, recipient, body string) (string, time.Time, error) {
		jid := utils.FormatJID(recipient)
		resp, err := client.SendMessage(sctx, jid, &waE2E.Message{
			Conversation: proto.String(body),
		})
		if err != nil {
			return "", time.Time{}, err
		}
		return resp.ID, resp.Timestamp, nil
	}

	// Detach from the whatsmeow event-handler context. That ctx has a very
	// short lifetime and would cancel mid-flight while we wait on embeddings
	// + LLM (each up to AIRequestTimeoutSec). Use a fresh context with our
	// own budget covering both round-trips plus DB work.
	timeout := time.Duration(config.AIRequestTimeoutSec*2+10) * time.Second
	if timeout < 30*time.Second {
		timeout = 30 * time.Second
	}
	aiCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Typing indicator. WhatsApp expires "composing" after ~25s, so the
	// service refreshes periodically while waiting for the LLM. Uses aiCtx
	// (not the event ctx) so presence calls keep working throughout the
	// processing window.
	presence := func(state string) {
		var pState types.ChatPresence
		switch state {
		case "composing":
			pState = types.ChatPresenceComposing
		case "paused":
			pState = types.ChatPresencePaused
		default:
			return
		}
		_ = client.SendChatPresence(aiCtx, evt.Info.Chat, pState, types.ChatPresenceMediaText)
	}

	return h.HandleIncoming(aiCtx, deviceID, chatJID, senderJID, text, send, presence)
}

// extractIncomingText pulls user-typed text from any of the supported wrappers.
// Returns "" for non-text messages so the AI handler can skip them.
func extractIncomingText(evt *events.Message) string {
	inner := utils.UnwrapMessage(evt.Message)
	if conv := inner.GetConversation(); conv != "" {
		return conv
	}
	if ext := inner.GetExtendedTextMessage(); ext != nil && ext.GetText() != "" {
		return ext.GetText()
	}
	if protoMsg := inner.GetProtocolMessage(); protoMsg != nil {
		if edited := protoMsg.GetEditedMessage(); edited != nil {
			if ext := edited.GetExtendedTextMessage(); ext != nil && ext.GetText() != "" {
				return ext.GetText()
			}
			if conv := edited.GetConversation(); conv != "" {
				return conv
			}
		}
	}
	return ""
}
