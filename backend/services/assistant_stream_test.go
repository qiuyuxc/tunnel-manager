package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"tunnel-manager/models"
)

func aiSSE(value any) string { raw, _ := json.Marshal(value); return "data: " + string(raw) + "\n\n" }

func TestAIResponsesNativeFormat(t *testing.T) {
	client := NewAIClient()
	client.Client.Transport = aiTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://example.com/v1/responses" {
			t.Fatal(r.URL.String())
		}
		var request map[string]any
		if json.NewDecoder(r.Body).Decode(&request) != nil {
			t.Fatal("bad request")
		}
		if request["stream"] != false || request["store"] != false || request["messages"] != nil {
			t.Fatal("wrong Responses request")
		}
		tool := request["tools"].([]any)[0].(map[string]any)
		if tool["type"] != "function" || tool["name"] != "create_tunnel" || tool["function"] != nil || tool["strict"] != false {
			t.Fatal("tools were not flattened, or optional parameters became mandatory")
		}
		if request["reasoning"].(map[string]any)["summary"] != "auto" {
			t.Fatal("summary not enabled")
		}
		raw, _ := json.Marshal(request)
		if strings.Contains(string(raw), "private reasoning") {
			t.Fatal("reasoning resent to provider")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"status":"completed","output":[{"type":"reasoning","summary":[{"type":"summary_text","text":"简要计划"}]},{"type":"message","content":[{"type":"output_text","text":"准备好了"}]},{"type":"function_call","call_id":"call1","name":"create_tunnel","arguments":"{\"name\":\"site\"}"}]}`))}, nil
	})
	reply, err := client.Complete(context.Background(), models.AIConnection{Endpoint: "https://example.com/v1/chat/completions", APIFormat: "responses", Model: "mock", ReasoningSummary: true}, "key", []models.AIMessage{{Role: "user", Content: "hello", Reasoning: "private reasoning"}}, []map[string]any{{"type": "function", "function": map[string]any{"name": "create_tunnel", "parameters": map[string]any{"type": "object"}}}})
	if err != nil || reply.Content != "准备好了" || reply.Reasoning != "简要计划" || len(reply.ToolCalls) != 1 || reply.ToolCalls[0].Function.Name != "create_tunnel" {
		t.Fatalf("reply=%+v err=%v", reply, err)
	}
	if AIFormat(models.AIConnection{Endpoint: "https://example.com/v1/responses"}) != "responses" {
		t.Fatal("full native endpoint not detected")
	}
}

func TestAIStreamingChatAssemblesToolsAndEmitsDeltas(t *testing.T) {
	wire := aiSSE(map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"reasoning_content": "先核对", "tool_calls": []any{map[string]any{"index": 0, "id": "call", "type": "function", "function": map[string]string{"name": "create_tunnel", "arguments": "{\"na"}}}}}}})
	wire += aiSSE(map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"content": "准备", "tool_calls": []any{map[string]any{"index": 0, "function": map[string]string{"arguments": "me\":\"site\"}"}}}}}}})
	wire += aiSSE(map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"content": "好了"}, "finish_reason": "tool_calls"}}}) + "data: [DONE]\n\n"
	deltas := []AIDelta{}
	reply, err := readAIStream(strings.NewReader(wire), "chat_completions", func(delta AIDelta) error { deltas = append(deltas, delta); return nil })
	if err != nil || reply.Content != "准备好了" || reply.Reasoning != "先核对" || len(reply.ToolCalls) != 1 || reply.ToolCalls[0].Function.Arguments != `{"name":"site"}` {
		t.Fatalf("reply=%+v err=%v", reply, err)
	}
	if !reflect.DeepEqual(deltas, []AIDelta{{"reasoning", "先核对"}, {"text", "准备"}, {"text", "好了"}}) {
		t.Fatal(deltas)
	}
}

func TestAIStreamingResponsesAndFailures(t *testing.T) {
	good := aiSSE(map[string]any{"type": "response.reasoning_summary_text.delta", "delta": "计划"}) + aiSSE(map[string]any{"type": "response.output_text.delta", "delta": "答案"}) + aiSSE(map[string]any{"type": "response.completed", "response": map[string]any{"status": "completed", "output": []any{map[string]any{"type": "message", "content": []any{map[string]string{"type": "output_text", "text": "答案"}}}, map[string]any{"type": "reasoning", "summary": []any{map[string]string{"type": "summary_text", "text": "计划"}}}, map[string]any{"type": "function_call", "call_id": "call", "name": "create_tunnel", "arguments": `{"name":"site"}`}}}})
	for _, tc := range []struct {
		name, format, wire string
		valid              bool
	}{
		{"native", "responses", good, true},
		{"native truncated", "responses", aiSSE(map[string]string{"type": "response.output_text.delta", "delta": "partial"}), false},
		{"native incomplete", "responses", aiSSE(map[string]string{"type": "response.incomplete"}), false},
		{"native error", "responses", aiSSE(map[string]string{"type": "error", "message": "upstream-secret"}), false},
		{"chat truncated", "chat_completions", `data: {"choices":[{"delta":{"content":"partial"}}]}` + "\n\n", false},
		{"chat unfinished tool", "chat_completions", `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"type":"function","function":{"name":"create_tunnel","arguments":"{"}}]}}]}` + "\n\ndata: [DONE]\n\n", false},
		{"chat length", "chat_completions", `data: {"choices":[{"delta":{"content":"partial"},"finish_reason":"length"}]}` + "\n\ndata: [DONE]\n\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deltas := []AIDelta{}
			reply, err := readAIStream(strings.NewReader(tc.wire), tc.format, func(delta AIDelta) error { deltas = append(deltas, delta); return nil })
			if (err == nil) != tc.valid {
				t.Fatalf("reply=%+v err=%v", reply, err)
			}
			if err != nil && strings.Contains(err.Error(), "upstream-secret") {
				t.Fatal("raw upstream error leaked")
			}
			if tc.valid && (reply.Content != "答案" || reply.Reasoning != "计划" || len(reply.ToolCalls) != 1 || len(deltas) != 2) {
				t.Fatal(reply, deltas)
			}
		})
	}
}

func TestAIStreamStartsBeforeUpstreamCompletes(t *testing.T) {
	client := NewAIClient()
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	client.Client.Transport = aiTransport(func(r *http.Request) (*http.Response, error) {
		var request map[string]any
		json.NewDecoder(r.Body).Decode(&request)
		if request["stream"] != true {
			t.Error("upstream streaming not requested")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: reader}, nil
	})
	first := make(chan struct{})
	go func() {
		io.WriteString(writer, "data: {\"choices\":[{\"delta\":{\"content\":\"first\"}}]}\n\n")
		<-first
		io.WriteString(writer, "data: {\"choices\":[{\"delta\":{\"content\":\" last\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
		writer.Close()
	}()
	reply, err := client.CompleteStream(context.Background(), models.AIConnection{Endpoint: "https://example.com/v1", Model: "mock"}, "key", nil, nil, func(delta AIDelta) error {
		if delta.Text == "first" {
			close(first)
		}
		return nil
	})
	if err != nil || reply.Content != "first last" {
		t.Fatal(reply, err)
	}
}
