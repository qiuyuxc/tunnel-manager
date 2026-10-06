package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"tunnel-manager/models"
)

func TestAIChatStreamsBeforeCompletionAndSavesOnlyCompleteTasks(t *testing.T) {
	for _, complete := range []bool{true, false} {
		name := "truncated"
		if complete {
			name = "complete"
		}
		t.Run(name, func(t *testing.T) {
			handler, state, admin, _ := newAITestHandler(t)
			settings := `{"scope":"personal","endpoint":"https://model.example.com/v1","model":"test-model","api_key":"secret","api_format":"responses","reasoning_summary":true}`
			if result := aiTestRequest(handler.SaveSettings, admin, settings, "", ""); result.Code != 200 {
				t.Fatal(result.Body.String())
			}
			var conversation models.AIConversation
			_ = json.Unmarshal(aiTestRequest(handler.NewConversation, admin, "", "", "").Body.Bytes(), &conversation)
			upstream, writer := io.Pipe()
			release := make(chan struct{})
			var once sync.Once
			finish := func() { once.Do(func() { close(release) }) }
			defer upstream.Close()
			handler.client.Client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
				go func() {
					defer writer.Close()
					_, _ = io.WriteString(writer, "data: {\"type\":\"response.reasoning_summary_text.delta\",\"delta\":\"核对监控参数\"}\n\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"请核对\"}\n\n")
					<-release
					if complete {
						_, _ = io.WriteString(writer, `data: {"type":"response.completed","response":{"status":"completed","output":[{"type":"reasoning","summary":[{"text":"核对监控参数"}]},{"type":"message","content":[{"type":"output_text","text":"请核对"}]},{"type":"function_call","call_id":"call","name":"create_monitor","arguments":"{\"name\":\"site\"}"}]}}`+"\n\n")
					}
				}()
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: upstream}, nil
			})
			handler.actions = map[string]http.HandlerFunc{"create_monitor": func(http.ResponseWriter, *http.Request) { t.Error("chat executed a task") }}
			router := chi.NewRouter()
			router.Use(chimw.Compress(5))
			router.Post("/{conversationID}", func(w http.ResponseWriter, r *http.Request) { handler.Chat(w, withUser(r, admin)) })
			server := httptest.NewServer(router)
			defer server.Close()
			defer finish()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			request, _ := http.NewRequestWithContext(ctx, "POST", server.URL+"/"+conversation.ID, strings.NewReader(`{"message":"创建监控","stream":true}`))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Accept-Encoding", "gzip")
			response, err := server.Client().Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if !strings.Contains(response.Header.Get("Content-Type"), "text/event-stream") || response.Header.Get("Content-Encoding") != "" {
				t.Fatal("SSE was buffered/compressed")
			}
			reader := bufio.NewReader(response.Body)
			partial := ""
			for !strings.Contains(partial, `"delta":"请核对"`+"}\n\n") {
				line, err := reader.ReadString('\n')
				if err != nil {
					t.Fatalf("first delta was not flushed before completion: %v", err)
				}
				partial += line
			}
			if !strings.Contains(partial, "event: status") || !strings.Contains(partial, "event: reasoning") {
				t.Fatal(partial)
			}
			_, _, before := state.AIStateFor(admin.ID)
			if len(before.Conversations[0].Tasks) != 0 || len(before.Conversations[0].Messages) != 0 {
				t.Fatal("partial reply persisted")
			}
			finish()
			tail, err := io.ReadAll(reader)
			if err != nil {
				t.Fatal(err)
			}
			_, _, after := state.AIStateFor(admin.ID)
			if complete {
				if !strings.Contains(string(tail), "event: done") || strings.Contains(string(tail), "event: error") {
					t.Fatal(string(tail))
				}
				stored := after.Conversations[0]
				if len(stored.Tasks) != 1 || stored.Tasks[0].Status != "pending" || len(stored.Messages) != 2 || stored.Messages[1].Reasoning != "核对监控参数" {
					t.Fatalf("invalid completed conversation: %+v", stored)
				}
			} else {
				if !strings.Contains(string(tail), "event: error") || strings.Contains(string(tail), "event: done") || len(after.Conversations[0].Tasks) != 0 || len(after.Conversations[0].Messages) != 0 {
					t.Fatalf("truncated reply became a task: %s", tail)
				}
			}
		})
	}
}

func TestAISettingsProtocolSelectionAndLegacyUpdates(t *testing.T) {
	handler, state, admin, _ := newAITestHandler(t)
	settings := `{"scope":"personal","endpoint":"https://model.example.com/v1","model":"test-model","api_key":"secret","api_format":"responses","reasoning_summary":true}`
	if response := aiTestRequest(handler.SaveSettings, admin, settings, "", ""); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	legacy := `{"scope":"personal","endpoint":"https://model.example.com/v1","model":"next-model"}`
	if response := aiTestRequest(handler.SaveSettings, admin, legacy, "", ""); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	_, _, saved := state.AIStateFor(admin.ID)
	if saved.Connection.APIFormat != "responses" || !saved.Connection.ReasoningSummary {
		t.Fatal("legacy update reset protocol options")
	}
	if response := aiTestRequest(handler.SaveSettings, admin, strings.Replace(settings, "responses", "unsupported", 1), "", ""); response.Code != 400 {
		t.Fatal("accepted unknown API format")
	}
}
