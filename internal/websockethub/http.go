package websockethub

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/coder/websocket"
)

type HTTPConfig struct {
	ResolvePrincipal func(*http.Request) (Principal, error)
	AcceptOptions    *websocket.AcceptOptions
}

func (h *Hub) HTTPHandler(config HTTPConfig) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if config.ResolvePrincipal == nil {
			http.Error(writer, "websocket principal resolver is not configured", http.StatusInternalServerError)
			return
		}
		principal, err := config.ResolvePrincipal(request)
		if err != nil {
			http.Error(writer, "unauthorized", http.StatusUnauthorized)
			return
		}
		workspaceID := request.URL.Query().Get("workspace_id")
		if workspaceID == "" {
			http.Error(writer, "workspace_id is required", http.StatusBadRequest)
			return
		}
		conn, err := websocket.Accept(writer, request, config.AcceptOptions)
		if err != nil {
			return
		}
		transport := &websocketTransport{conn: conn}
		session, err := h.Connect(request.Context(), principal, workspaceID, transport)
		if err != nil {
			_ = conn.Close(websocket.StatusPolicyViolation, err.Error())
			return
		}
		defer session.Close()
		for {
			messageType, payload, err := conn.Read(request.Context())
			if err != nil {
				return
			}
			if messageType != websocket.MessageText {
				continue
			}
			var message clientMessage
			if err := json.Unmarshal(payload, &message); err != nil {
				_ = enqueueSessionError(session, "", "invalid_json")
				continue
			}
			if message.Type != "subscription.set" {
				_ = enqueueSessionError(session, message.RequestID, "unsupported_message")
				continue
			}
			var subscription SubscriptionRequest
			if err := json.Unmarshal(message.Data, &subscription); err != nil {
				_ = enqueueSessionError(session, message.RequestID, "invalid_subscription")
				continue
			}
			if err := session.Subscribe(request.Context(), subscription); err != nil {
				_ = enqueueSessionError(session, message.RequestID, err.Error())
			}
		}
	})
}

type clientMessage struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Data      json.RawMessage `json:"data"`
}

func enqueueSessionError(session *Session, requestID, reason string) error {
	data, _ := json.Marshal(map[string]string{"reason": reason})
	return session.enqueue(Event{
		EventID: "session-error-" + requestID, Type: "session.error",
		SchemaVersion: "1.0", WorkspaceID: session.workspaceID,
		RequestID: requestID, Data: data,
	})
}

type websocketTransport struct {
	conn *websocket.Conn
}

func (t *websocketTransport) Write(ctx context.Context, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return t.conn.Write(ctx, websocket.MessageText, payload)
}

func (t *websocketTransport) Close() error {
	return t.conn.Close(websocket.StatusNormalClosure, "")
}
