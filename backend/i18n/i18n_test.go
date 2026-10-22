package i18n

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLanguagePreference(t *testing.T) {
	for _, item := range []struct{ header, expected string }{
		{"", "zh-CN"}, {"en-US,en;q=0.9", "en"}, {"zh-TW,en;q=0.8", "zh-CN"},
		{"en;q=0.2,zh-CN;q=0.8", "zh-CN"}, {"fr,ja;q=0.8", "en"}, {"en;q=0,zh;q=1", "zh-CN"},
	} {
		if result := Language(item.header); result != item.expected {
			t.Errorf("%q: %s", item.header, result)
		}
	}
}

func TestLocalizedErrorsPreserveResourceValuesAndStreaming(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/test", nil)
	request.Header.Set("Accept-Language", "en")
	response := httptest.NewRecorder()
	Middleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(`{"error":"请求失败","name":"请求失败","content":"原始记录"}`))
	})).ServeHTTP(response, request)
	if response.Code != 400 || response.Header().Get("Content-Language") != "en" {
		t.Fatal(response.Result())
	}
	if response.Body.String() == `{"error":"请求失败","name":"请求失败","content":"原始记录"}` {
		t.Fatal("error not translated")
	}
	if !strings.Contains(response.Body.String(), `"name":"请求失败"`) || !strings.Contains(response.Body.String(), `"content":"原始记录"`) {
		t.Fatal(response.Body.String())
	}
	stream := httptest.NewRecorder()
	Middleware(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = writer.Write([]byte("event: text\ndata: 用户内容\n\n"))
		if err := http.NewResponseController(writer).Flush(); err != nil {
			t.Error(err)
		}
	})).ServeHTTP(stream, request)
	if stream.Body.String() != "event: text\ndata: 用户内容\n\n" || !stream.Flushed {
		t.Fatal(stream.Body.String())
	}
}

func TestStreamApplicationMessagesPreserveModelContent(t *testing.T) {
	raw := []byte(`{"message":"已收到消息，正在准备…","delta":"用户内容","tasks":[{"message":"请求失败"}]}`)
	result := string(TranslateJSON(raw, "en"))
	if strings.Contains(result, `"message":"已收到消息，正在准备…"`) || !strings.Contains(result, `"delta":"用户内容"`) || !strings.Contains(result, `"message":"请求失败"`) {
		t.Fatal(result)
	}
}
