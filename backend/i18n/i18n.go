package i18n

import (
	"embed"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

//go:embed zh-CN.json en.json
var files embed.FS
var chinese, english map[string]string
var messages []string
var bySource map[string]string

func init() {
	chinese = read("zh-CN.json")
	english = read("en.json")
	bySource = make(map[string]string)
	for id, source := range chinese {
		if english[id] != "" && source != "" {
			bySource[source] = english[id]
			messages = append(messages, source)
		}
	}
	sort.Slice(messages, func(a, b int) bool { return len(messages[a]) > len(messages[b]) })
}

func read(name string) map[string]string {
	data, err := files.ReadFile(name)
	if err != nil {
		panic(err)
	}
	var catalog map[string]string
	if err := json.Unmarshal(data, &catalog); err != nil {
		panic(err)
	}
	return catalog
}

func Language(header string) string {
	type preference struct {
		language string
		weight   float64
		index    int
	}
	var preferences []preference
	for index, value := range strings.Split(header, ",") {
		parts := strings.Split(strings.TrimSpace(value), ";")
		language := strings.ToLower(strings.TrimSpace(parts[0]))
		weight := 1.0
		if len(parts) > 1 {
			var q float64
			if parsed, err := strconv.ParseFloat(strings.TrimPrefix(strings.TrimSpace(parts[1]), "q="), 64); err == nil {
				q = parsed
				weight = q
			}
		}
		if weight <= 0 {
			continue
		}
		if language == "zh" || strings.HasPrefix(language, "zh-") {
			preferences = append(preferences, preference{"zh-CN", weight, index})
		}
		if language == "en" || strings.HasPrefix(language, "en-") {
			preferences = append(preferences, preference{"en", weight, index})
		}
	}
	sort.SliceStable(preferences, func(a, b int) bool { return preferences[a].weight > preferences[b].weight })
	if len(preferences) > 0 {
		return preferences[0].language
	}
	if header == "" {
		return "zh-CN"
	}
	return "en"
}

func Translate(message, language string) string {
	if language != "en" {
		return message
	}
	if translated, ok := bySource[message]; ok {
		return translated
	}
	// Errors can contain a fixed application message followed by an external error.
	for _, source := range messages {
		if len(source) >= 6 && strings.HasPrefix(message, source) {
			return bySource[source] + strings.TrimPrefix(message, source)
		}
	}
	return message
}

// Localize only application messages at the top level. Resource values and
// conversation output, including nested fields, must remain untouched.
func TranslateJSON(data []byte, language string) []byte {
	var value map[string]json.RawMessage
	if json.Unmarshal(data, &value) != nil {
		return data
	}
	changed := false
	for _, key := range []string{"error", "message"} {
		var message string
		if json.Unmarshal(value[key], &message) != nil {
			continue
		}
		translated := Translate(message, language)
		if translated != message {
			value[key], _ = json.Marshal(translated)
			changed = true
		}
	}
	if !changed {
		return data
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return data
	}
	return encoded
}

type responseWriter struct {
	http.ResponseWriter
	language string
}

func (writer *responseWriter) Write(data []byte) (int, error) {
	if !strings.Contains(writer.Header().Get("Content-Type"), "application/json") {
		return writer.ResponseWriter.Write(data)
	}
	_, err := writer.ResponseWriter.Write(TranslateJSON(data, writer.language))
	return len(data), err
}
func (writer *responseWriter) Unwrap() http.ResponseWriter { return writer.ResponseWriter }
func (writer *responseWriter) Flush()                      { _ = http.NewResponseController(writer.ResponseWriter).Flush() }
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		language := Language(request.Header.Get("Accept-Language"))
		writer.Header().Set("Content-Language", language)
		writer.Header().Add("Vary", "Accept-Language")
		next.ServeHTTP(&responseWriter{writer, language}, request)
	})
}
