package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"tunnel-manager/models"
	"tunnel-manager/services"
)

// Editing validates and persists a new pending revision. It never executes a
// business operation. Execution must refer to the revision the user reviewed.
func (handler *AssistantHandler) editTask(response http.ResponseWriter, request *http.Request, user *models.SessionUser, conversationID, taskID string, arguments json.RawMessage, revision *int) {
	_, _, state := handler.store.AIStateFor(user.ID)
	conversation, err := aiConversation(&state, conversationID)
	if err != nil {
		writeJSON(response, http.StatusNotFound, map[string]string{"error": "任务不存在"})
		return
	}
	var original *models.AITask
	for index := range conversation.Tasks {
		if conversation.Tasks[index].ID == taskID {
			original = &conversation.Tasks[index]
			break
		}
	}
	if original == nil || original.Status != "pending" || revision == nil || *revision != original.Revision {
		writeJSON(response, http.StatusConflict, map[string]string{"error": "只能编辑当前版本的待确认任务，请刷新后重试"})
		return
	}
	account, found := handler.store.GetUserByID(user.ID)
	if !found || account.Status != models.UserActive || !aiAllowed(user, original.Tool) || (aiPermission(original.Tool) == "assistant_admin" && account.Role != models.RoleAdmin) {
		writeJSON(response, http.StatusForbidden, map[string]string{"error": "当前账号无权修改此任务"})
		return
	}
	if strings.HasPrefix(original.Tool, "delete_") {
		writeJSON(response, http.StatusBadRequest, map[string]string{"error": "删除任务没有可修改的配置；更换删除对象请取消后重新生成"})
		return
	}
	if aiCloudflareTool(original.Tool) {
		if handler.connectionMarker(user.ID) != original.ConnectionID {
			writeJSON(response, http.StatusConflict, map[string]string{"error": "Cloudflare 连接已变化，请重新生成任务"})
			return
		}
		if client := UserCF(request); client != nil {
			pinned, err := client.SnapshotForAssistant()
			if err != nil {
				writeJSON(response, http.StatusBadRequest, map[string]string{"error": "Cloudflare 连接不可用"})
				return
			}
			request = withCF(request, pinned)
		}
	}
	if err := handler.checkChange(request, user, *original); err != nil {
		writeJSON(response, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	raw := string(arguments)
	if original.Tool == "update_dns_record" {
		fields, err := aiObject(raw, "zone_id", "record_id", "name", "type", "content", "ttl", "proxied", "priority", "data")
		var oldArgs, before map[string]json.RawMessage
		_ = json.Unmarshal(original.Arguments, &oldArgs)
		_ = json.Unmarshal(original.Before, &before)
		if err != nil || aiString(fields, "zone_id") != aiString(oldArgs, "zone_id") || aiString(fields, "record_id") != aiString(oldArgs, "record_id") {
			writeJSON(response, http.StatusBadRequest, map[string]string{"error": "请保留原区域和记录 ID；更换对象需要重新生成任务"})
			return
		}
		delete(fields, "zone_id")
		delete(fields, "record_id")
		intent := map[string]any{"zone_id": aiString(oldArgs, "zone_id"), "record_id": aiString(oldArgs, "record_id"), "record_name": aiString(before, "name"), "record_type": aiString(before, "type"), "changes": fields}
		normalized, _ := json.Marshal(intent)
		raw = string(normalized)
	}
	call := services.AIToolCall{Type: "function"}
	call.Function.Name, call.Function.Arguments = original.Tool, raw
	edited, err := aiBuildTask(user, call)
	if err == nil {
		err = handler.prepareChange(request, user, &edited)
	}
	if err != nil {
		writeJSON(response, http.StatusBadRequest, map[string]string{"error": "修改未保存：" + err.Error()})
		return
	}
	if aiCloudflareTool(original.Tool) && handler.connectionMarker(user.ID) != original.ConnectionID {
		writeJSON(response, http.StatusConflict, map[string]string{"error": "Cloudflare 连接已变化，请重新生成任务"})
		return
	}
	edited.ID, edited.ConnectionID, edited.Revision = original.ID, original.ConnectionID, original.Revision+1
	err = handler.store.UpdateAI(user.ID, func(_ *models.AIState, state *models.AIUserState) error {
		current, err := aiConversation(state, conversationID)
		if err != nil {
			return err
		}
		for index, task := range current.Tasks {
			if task.ID == taskID && task.Status == "pending" && task.Revision == *revision {
				current.Tasks[index] = edited
				return nil
			}
		}
		return errors.New("任务状态已变化，请刷新后重试")
	})
	if err != nil {
		writeJSON(response, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(response, http.StatusOK, edited)
}
