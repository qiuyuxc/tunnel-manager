package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"tunnel-manager/models"
)

type AIDelta struct{ Type, Text string }

func AIFormat(connection models.AIConnection) string {
	if connection.APIFormat != "" {
		return connection.APIFormat
	}
	if strings.HasSuffix(strings.TrimRight(connection.Endpoint, "/"), "/responses") {
		return "responses"
	}
	return "chat_completions"
}

func (client *AIClient) Complete(ctx context.Context, connection models.AIConnection, key string, messages []models.AIMessage, tools []map[string]any) (AIReply, error) {
	return client.complete(ctx, connection, key, messages, tools, nil)
}

func (client *AIClient) CompleteStream(ctx context.Context, connection models.AIConnection, key string, messages []models.AIMessage, tools []map[string]any, emit func(AIDelta) error) (AIReply, error) {
	return client.complete(ctx, connection, key, messages, tools, emit)
}

func (client *AIClient) complete(ctx context.Context, connection models.AIConnection, key string, messages []models.AIMessage, tools []map[string]any, emit func(AIDelta) error) (AIReply, error) {
	endpoint, err := ValidateAIEndpoint(connection.Endpoint)
	if err != nil {
		return AIReply{}, err
	}
	format := AIFormat(connection)
	if format != "responses" && format != "chat_completions" {
		return AIReply{}, errors.New("不支持的 AI 接口格式")
	}
	endpoint = strings.TrimSuffix(strings.TrimSuffix(endpoint, "/chat/completions"), "/responses")
	request := map[string]any{"model": connection.Model, "stream": emit != nil}
	// Provider reasoning summaries belong to the UI, not subsequent prompts.
	input := []map[string]string{}
	for _, message := range messages {
		input = append(input, map[string]string{"role": message.Role, "content": message.Content})
	}
	if format == "responses" {
		endpoint += "/responses"
		request["input"], request["store"] = input, false
		if connection.ReasoningSummary {
			request["reasoning"] = map[string]string{"summary": "auto"}
		}
		if len(tools) > 0 {
			flat := []map[string]any{}
			for _, tool := range tools {
				function, ok := tool["function"].(map[string]any)
				if !ok {
					return AIReply{}, errors.New("工具格式无效")
				}
				definition := map[string]any{"type": "function", "strict": false}
				for key, value := range function {
					definition[key] = value
				}
				flat = append(flat, definition)
			}
			request["tools"], request["tool_choice"] = flat, "auto"
		}
	} else {
		endpoint += "/chat/completions"
		request["messages"] = input
		if len(tools) > 0 {
			request["tools"], request["tool_choice"] = tools, "auto"
		}
	}
	body, err := json.Marshal(request)
	if err != nil {
		return AIReply{}, errors.New("构造 AI 请求失败")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return AIReply{}, errors.New("构造 AI 请求失败")
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	if emit != nil {
		req.Header.Set("Accept", "text/event-stream")
	}
	response, err := client.Client.Do(req)
	if err != nil {
		return AIReply{}, errors.New("AI 连接失败或超时，请检查端点与网络后重试")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return AIReply{}, fmt.Errorf("AI 服务返回 HTTP %d，请检查密钥、模型、接口格式和额度", response.StatusCode)
	}
	var reply AIReply
	if emit != nil && strings.Contains(response.Header.Get("Content-Type"), "text/event-stream") {
		reply, err = readAIStream(response.Body, format, emit)
	} else {
		data, readErr := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
		if readErr != nil || len(data) > 1<<20 {
			return AIReply{}, errors.New("AI 响应过大或读取失败")
		}
		reply, err = parseAIReply(data, format)
		// Some compatible providers ignore stream=true; consume that one
		// response without retrying or charging for a second generation.
		if err == nil && emit != nil {
			if reply.Reasoning != "" {
				err = emit(AIDelta{"reasoning", reply.Reasoning})
			}
			if err == nil && reply.Content != "" {
				err = emit(AIDelta{"text", reply.Content})
			}
		}
	}
	if err != nil {
		return AIReply{}, err
	}
	return validateAIReply(reply)
}

func validateAIReply(reply AIReply) (AIReply, error) {
	if len(reply.ToolCalls) > 8 || len(reply.Content) > 32000 || len(reply.Reasoning) > 32000 || (strings.TrimSpace(reply.Content) == "" && len(reply.ToolCalls) == 0) {
		return AIReply{}, errors.New("AI 输出为空、过长或任务过多，请缩小本次请求范围")
	}
	for _, call := range reply.ToolCalls {
		if call.Type != "function" || call.Function.Name == "" || len(call.Function.Name) > 200 || len(call.Function.Arguments) > 12000 || !json.Valid([]byte(call.Function.Arguments)) {
			return AIReply{}, errors.New("AI 工具参数不完整，未生成任务")
		}
	}
	return reply, nil
}

func parseAIReply(data []byte, format string) (AIReply, error) {
	if format == "responses" {
		var result struct {
			Status string `json:"status"`
			Output []struct {
				Type      string `json:"type"`
				ID        string `json:"id"`
				CallID    string `json:"call_id"`
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
				Content   []struct {
					Type    string `json:"type"`
					Text    string `json:"text"`
					Refusal string `json:"refusal"`
				} `json:"content"`
				Summary []struct {
					Text string `json:"text"`
				} `json:"summary"`
			} `json:"output"`
		}
		if json.Unmarshal(data, &result) != nil || result.Status != "completed" {
			return AIReply{}, errors.New("Responses 响应未完成，未生成任务")
		}
		reply := AIReply{}
		for _, item := range result.Output {
			switch item.Type {
			case "message":
				for _, part := range item.Content {
					if part.Type == "output_text" {
						reply.Content += part.Text
					}
					if part.Type == "refusal" {
						reply.Content += part.Refusal
					}
				}
			case "reasoning":
				for _, part := range item.Summary {
					reply.Reasoning += part.Text
				}
			case "function_call":
				call := AIToolCall{ID: item.CallID, Type: "function"}
				call.Function.Name, call.Function.Arguments = item.Name, item.Arguments
				reply.ToolCalls = append(reply.ToolCalls, call)
			}
		}
		return validateAIReply(reply)
	}
	var result struct {
		Choices []struct {
			Message      AIReply `json:"message"`
			FinishReason string  `json:"finish_reason"`
		} `json:"choices"`
	}
	if json.Unmarshal(data, &result) != nil || len(result.Choices) == 0 {
		return AIReply{}, errors.New("AI 响应格式无效，需兼容 Chat Completions")
	}
	if result.Choices[0].FinishReason == "length" || result.Choices[0].FinishReason == "content_filter" {
		return AIReply{}, errors.New("AI 输出未完整结束，未生成任务")
	}
	return validateAIReply(result.Choices[0].Message)
}

func readAIStream(reader io.Reader, format string, emit func(AIDelta) error) (AIReply, error) {
	scanner := bufio.NewScanner(io.LimitReader(reader, (4<<20)+1))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	reply := AIReply{}
	calls := map[int]*AIToolCall{}
	complete := false
	textBytes, reasoningBytes, total := 0, 0, 0
	push := func(kind, text string) error {
		if kind == "text" {
			textBytes += len(text)
		} else {
			reasoningBytes += len(text)
		}
		if textBytes > 32000 || reasoningBytes > 32000 {
			return errors.New("AI 输出过长，请缩小请求范围")
		}
		if text == "" {
			return nil
		}
		return emit(AIDelta{kind, text})
	}
	process := func(raw string) error {
		if raw == "[DONE]" {
			if format == "chat_completions" {
				complete = true
			}
			return nil
		}
		if format == "responses" {
			var event struct {
				Type     string          `json:"type"`
				Delta    string          `json:"delta"`
				Response json.RawMessage `json:"response"`
			}
			if json.Unmarshal([]byte(raw), &event) != nil {
				return errors.New("AI 流式数据格式无效")
			}
			switch event.Type {
			case "response.output_text.delta", "response.refusal.delta":
				return push("text", event.Delta)
			case "response.reasoning_summary_text.delta":
				return push("reasoning", event.Delta)
			case "response.completed":
				var err error
				reply, err = parseAIReply(event.Response, "responses")
				if err != nil {
					return err
				}
				complete = true
			case "response.failed", "response.incomplete", "error":
				return errors.New("AI 输出中断或未完成，未生成任务")
			}
			return nil
		}
		var event struct {
			Error   json.RawMessage `json:"error"`
			Choices []struct {
				Index        int    `json:"index"`
				FinishReason string `json:"finish_reason"`
				Delta        struct {
					Content   string `json:"content"`
					Reasoning string `json:"reasoning_content"`
					ToolCalls []struct {
						Index int `json:"index"`
						AIToolCall
					} `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if json.Unmarshal([]byte(raw), &event) != nil {
			return errors.New("AI 流式数据格式无效")
		}
		if len(event.Error) > 0 && string(event.Error) != "null" {
			return errors.New("AI 服务流式输出失败")
		}
		for _, choice := range event.Choices {
			if choice.Index != 0 {
				continue
			}
			if choice.FinishReason == "length" || choice.FinishReason == "content_filter" {
				return errors.New("AI 输出未完整结束，未生成任务")
			}
			reply.Content += choice.Delta.Content
			reply.Reasoning += choice.Delta.Reasoning
			if err := push("text", choice.Delta.Content); err != nil {
				return err
			}
			if err := push("reasoning", choice.Delta.Reasoning); err != nil {
				return err
			}
			for _, delta := range choice.Delta.ToolCalls {
				if delta.Index < 0 || delta.Index >= 8 {
					return errors.New("AI 任务过多")
				}
				call := calls[delta.Index]
				if call == nil {
					call = &AIToolCall{}
					calls[delta.Index] = call
				}
				if delta.ID != "" {
					call.ID = delta.ID
				}
				if delta.Type != "" {
					call.Type = delta.Type
				}
				call.Function.Name += delta.Function.Name
				call.Function.Arguments += delta.Function.Arguments
				if len(call.Function.Name) > 200 || len(call.Function.Arguments) > 12000 {
					return errors.New("AI 工具参数过长")
				}
			}
		}
		return nil
	}
	data := ""
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		total += len(line) + 1
		if total > 4<<20 {
			return AIReply{}, errors.New("AI 流式响应过大")
		}
		if line == "" {
			if data != "" {
				if err := process(strings.TrimSuffix(data, "\n")); err != nil {
					return AIReply{}, err
				}
				data = ""
			}
			if complete {
				break
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			value := strings.TrimPrefix(line, "data:")
			value = strings.TrimPrefix(value, " ")
			data += value + "\n"
			if len(data) > 1<<20 {
				return AIReply{}, errors.New("AI 流式事件过大")
			}
		}
	}
	if scanner.Err() != nil {
		return AIReply{}, errors.New("AI 流式读取失败，未生成任务")
	}
	if !complete {
		return AIReply{}, errors.New("AI 连接中断，回复未完整接收，未生成任务")
	}
	if format == "chat_completions" {
		for index := 0; index < len(calls); index++ {
			if calls[index] == nil {
				return AIReply{}, errors.New("AI 工具序号不完整")
			}
			reply.ToolCalls = append(reply.ToolCalls, *calls[index])
		}
	}
	return validateAIReply(reply)
}
