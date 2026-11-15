package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"tunnel-manager/i18n"
)

// Only completed, validated conversation data is published as a done event.
// Ordinary Chat errors keep using writeJSON and become sanitized SSE errors.
type aiChatStream struct {
	writer http.ResponseWriter
	header http.Header
	status int
	mu     sync.Mutex
	stop   chan struct{}
	closed chan struct{}
}

func newAIChatStream(writer http.ResponseWriter, ctx context.Context) *aiChatStream {
	writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-cache, no-store, no-transform")
	writer.Header().Set("X-Accel-Buffering", "no")
	writer.WriteHeader(http.StatusOK)
	stream := &aiChatStream{writer: writer, header: http.Header{}, status: 200, stop: make(chan struct{}), closed: make(chan struct{})}
	_ = stream.Send("status", map[string]string{"message": "已收到消息，正在准备…"})
	go func() {
		defer close(stream.closed)
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if stream.Send("ping", map[string]bool{"ok": true}) != nil {
					return
				}
			case <-stream.stop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	return stream
}
func (stream *aiChatStream) Close()                 { close(stream.stop); <-stream.closed }
func (stream *aiChatStream) Header() http.Header    { return stream.header }
func (stream *aiChatStream) WriteHeader(status int) { stream.status = status }
func (stream *aiChatStream) Write(data []byte) (int, error) {
	kind := "done"
	if stream.status >= 400 {
		kind = "error"
	}
	return len(data), stream.Send(kind, json.RawMessage(data))
}
func (stream *aiChatStream) Send(kind string, data any) error {
	stream.mu.Lock()
	defer stream.mu.Unlock()
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if kind == "status" || kind == "error" {
		raw = i18n.TranslateJSON(raw, stream.writer.Header().Get("Content-Language"))
	}
	if _, err = fmt.Fprintf(stream.writer, "event: %s\ndata: %s\n\n", kind, raw); err != nil {
		return err
	}
	return http.NewResponseController(stream.writer).Flush()
}
