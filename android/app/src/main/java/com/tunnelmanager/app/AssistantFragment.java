package com.tunnelmanager.app;

import android.graphics.Typeface;
import android.os.Handler;
import android.os.Looper;
import android.text.Editable;
import android.text.InputType;
import android.text.InputFilter;
import android.text.TextWatcher;
import android.view.LayoutInflater;
import android.view.Gravity;
import android.view.View;
import android.view.ViewGroup;
import android.view.inputmethod.EditorInfo;
import android.view.inputmethod.InputMethodManager;
import android.widget.CheckBox;
import android.widget.EditText;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import org.json.JSONArray;
import org.json.JSONObject;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.Set;

public class AssistantFragment extends PageFragment {
    static final class State {
        final String token = Session.token();
        final String server = Session.server();
        final Map<String, String> drafts = new HashMap<>();
        final Set<String> busy = new HashSet<>();
        JSONArray conversations = new JSONArray();
        JSONObject settings = new JSONObject();
        String currentID = "";
        String error = "";
        boolean sending;
        boolean includeResources;
        boolean loaded;
    }

    private State state;
    private LinearLayout messages;
    private ScrollView scroll;
    private EditText input;
    private TextView send;
    private TextView status;
    private TextView historyButton;
    private TextView taskButton;
    private TextView executeButton;
    private View firstPendingCard;
    private View taskSection;
    private String renderedConversationID = "";
    private int renderedTaskCount;
    private final AssistantTaskSelection selection = new AssistantTaskSelection();
    private final Handler poller = new Handler(Looper.getMainLooper());

    @Override String route() { return "/assistant"; }

    @Override protected View build(@NonNull LayoutInflater inflater, @Nullable ViewGroup container) {
        state = console().assistantState;
        LinearLayout root = UI.column(requireContext());
        root.setPadding(UI.dp(12), UI.dp(8), UI.dp(12), UI.dp(8));
        root.setFocusableInTouchMode(true);
        LinearLayout header = UI.row(requireContext());
        header.addView(UI.text(requireContext(), "AI 助手", 22, Theme.p().ink, Typeface.BOLD), new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1));
        historyButton = compactButton("会话", UI.BTN_GHOST);
        historyButton.setOnClickListener(view -> showHistory());
        header.addView(historyButton);
        TextView settings = compactButton("连接", UI.BTN_GHOST);
        settings.setContentDescription("AI 连接配置");
        settings.setOnClickListener(view -> showSettings());
        header.addView(settings);
        root.addView(header);
        LinearLayout summary = UI.row(requireContext());
        status = UI.muted(requireContext(), "正在读取连接与会话…");
        status.setTextSize(11);
        summary.addView(status, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1));
        taskButton = compactButton("任务 0", UI.BTN_GHOST);
        taskButton.setOnClickListener(view -> revealTasks());
        summary.addView(taskButton);
        root.addView(summary);
        scroll = new ScrollView(requireContext());
        messages = UI.column(requireContext());
        scroll.addView(messages);
        root.addView(scroll, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, 0, 1));
        LinearLayout options = UI.row(requireContext());
        CheckBox resources = new CheckBox(requireContext());
        resources.setText("允许发送资源名称与 ID");
        resources.setMinHeight(UI.dp(44));
        resources.setTextColor(Theme.p().mute);
        resources.setTextSize(11);
        resources.setChecked(state.includeResources);
        resources.setOnCheckedChangeListener((button, checked) -> state.includeResources = checked);
        options.addView(resources, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1));
        TextView privacy = compactButton("隐私说明", UI.BTN_GHOST);
        privacy.setTextSize(11);
        privacy.setOnClickListener(view -> Sheet.of(requireContext(), "隐私与资源授权")
                .content(UI.muted(requireContext(), "会话仅对你可见。消息会发送给所配置的 AI 服务，请勿填写密码或连接令牌。勾选后还会附带你有权限访问的资源名称与 ID，不包含 Cloudflare 凭据。"))
                .show());
        options.addView(privacy);
        root.addView(options);
        LinearLayout composer = UI.row(requireContext());
        composer.setGravity(Gravity.BOTTOM);
        input = UI.textArea(requireContext(), "描述你想完成的配置…");
        input.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_FLAG_MULTI_LINE | InputType.TYPE_TEXT_FLAG_CAP_SENTENCES);
        input.setMinLines(1);
        input.setMaxLines(4);
        input.setTextSize(16);
        input.setTypeface(Typeface.DEFAULT);
        input.setPadding(UI.dp(12), UI.dp(10), UI.dp(12), UI.dp(10));
        input.setImeOptions(EditorInfo.IME_FLAG_NO_ENTER_ACTION | EditorInfo.IME_FLAG_NO_EXTRACT_UI);
        input.setFilters(new InputFilter[]{new InputFilter.LengthFilter(8000)});
        input.setSaveEnabled(false);
        input.setImportantForAutofill(View.IMPORTANT_FOR_AUTOFILL_NO);
        input.setContentDescription("发送给 AI 助手的消息");
        input.setText(state.drafts.getOrDefault(draftKey(), ""));
        input.addTextChangedListener(new TextWatcher() {
            public void beforeTextChanged(CharSequence value, int start, int count, int after) { }
            public void onTextChanged(CharSequence value, int start, int before, int count) { state.drafts.put(draftKey(), value.toString()); }
            public void afterTextChanged(Editable value) { updateSend(); }
        });
        composer.addView(input, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1));
        send = compactButton("发送", UI.BTN_PRIMARY);
        send.setMinHeight(UI.dp(48));
        send.setOnClickListener(view -> sendMessage());
        composer.addView(send);
        UI.margin(send, UI.SM, 0, 0, 0);
        root.addView(composer);
        render();
        load();
        return root;
    }

    private boolean sameSession() { return state != null && state.token.equals(Session.token()) && state.server.equals(Session.server()); }
    private String draftKey() { return state.currentID.isEmpty() ? "new" : state.currentID; }
    private JSONObject current() {
        for (int index = 0; index < state.conversations.length(); index++) {
            JSONObject conversation = state.conversations.optJSONObject(index);
            if (conversation != null && state.currentID.equals(conversation.optString("id"))) return conversation;
        }
        return new JSONObject();
    }
    private JSONArray tasks() { JSONArray tasks = current().optJSONArray("tasks"); return tasks == null ? new JSONArray() : tasks; }
    private void merge(JSONObject conversation) {
        for (int index = 0; index < state.conversations.length(); index++) {
            if (state.conversations.optJSONObject(index).optString("id").equals(conversation.optString("id"))) {
                try { state.conversations.put(index, conversation); } catch (Exception ignored) { }
                return;
            }
        }
        state.conversations.put(conversation);
    }
    private void load() {
        if (!sameSession()) return;
        Api.async(() -> {
            if (!sameSession()) throw new IllegalStateException("会话已变更");
            return new JSONObject().put("settings", Api.get("/api/assistant/settings")).put("history", Api.get("/api/assistant/conversations"));
        }, result -> {
            if (!sameSession()) return;
            state.settings = result.optJSONObject("settings");
            state.conversations = result.optJSONObject("history").optJSONArray("conversations");
            if (state.conversations == null) state.conversations = new JSONArray();
            if ((!state.loaded || !state.currentID.isEmpty()) && current().optString("id").isEmpty()) state.currentID = state.conversations.length() > 0 ? state.conversations.optJSONObject(0).optString("id") : "";
            state.loaded = true;
            restoreDraft();
            render();
        }, failure -> { if (sameSession()) { state.error = failure.getMessage(); render(); } });
    }

    private void render() {
        if (!alive() || messages == null || !sameSession()) return;
        int scrollPosition = scroll.getScrollY();
        boolean reveal = !state.currentID.equals(renderedConversationID) || tasks().length() > renderedTaskCount;
        renderedConversationID = state.currentID;
        renderedTaskCount = tasks().length();
        status.setText("管理员连接 · 确认后执行" + (state.settings.optBoolean("configured") ? "" : " · 请先配置连接"));
        messages.removeAllViews();
        if (!state.error.isEmpty()) {
            TextView error = UI.text(requireContext(), state.error, 13, Theme.p().error, Typeface.NORMAL);
            error.setAccessibilityLiveRegion(View.ACCESSIBILITY_LIVE_REGION_POLITE);
            messages.addView(error);
        }
        JSONArray history = current().optJSONArray("messages");
        if (history == null || history.length() == 0) {
            messages.addView(UI.text(requireContext(), "今天想配置什么？", 22, Theme.p().ink, Typeface.BOLD));
            messages.addView(UI.muted(requireContext(), "从隧道、DNS 或监控开始，执行前由你确认。"));
            for (String prompt : new String[]{"查看我现有的隧道与监控项目", "把 NAS 接入域名，先问我所需信息", "创建一个私有监控项目，每 60 秒检查一次"}) {
                TextView suggestion = UI.button(requireContext(), prompt, UI.BTN_SECONDARY);
                suggestion.setGravity(Gravity.START | Gravity.CENTER_VERTICAL);
                suggestion.setTextSize(12);
                suggestion.setOnClickListener(view -> input.setText(prompt));
                messages.addView(suggestion, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));
                UI.margin(suggestion, 0, UI.SM, 0, 0);
            }
        } else {
            for (int index = 0; index < history.length(); index++) {
                JSONObject message = history.optJSONObject(index);
                LinearLayout card = UI.card(requireContext());
                card.addView(UI.label(requireContext(), "user".equals(message.optString("role")) ? "你" : "AI 助手"));
                TextView text = UI.text(requireContext(), message.optString("content"), 14, Theme.p().ink, Typeface.NORMAL);
                text.setTextIsSelectable(true);
                card.addView(text);
                messages.addView(card);
                UI.margin(card, 0, UI.SM, 0, UI.SM);
            }
        }
        renderTasks();
        if (state.sending) messages.addView(UI.muted(requireContext(), "正在准备回复与计划，尚未执行…"));
        historyButton.setEnabled(!state.sending);
        updateSend();
        scroll.post(() -> {
            if (scroll == null || !sameSession()) return;
            if (!state.error.isEmpty()) scroll.scrollTo(0, 0);
            else if (reveal && taskSection != null) revealTasks();
            else scroll.scrollTo(0, scrollPosition);
        });
    }

    private void updateSend() {
        if (send == null || input == null || !sameSession()) return;
        send.setText(state.sending ? "思考中…" : "发送");
        send.setEnabled(!state.sending && state.settings.optBoolean("configured") && !input.getText().toString().trim().isEmpty());
        send.setAlpha(send.isEnabled() ? 1f : .5f);
    }

    private void restoreDraft() {
        if (!alive() || input == null) return;
        String draft = state.drafts.getOrDefault(draftKey(), "");
        if (!draft.equals(input.getText().toString())) {
            input.setText(draft);
            input.setSelection(input.length());
        }
    }

    private void sendMessage() {
        String text = input.getText().toString().trim();
        if (text.isEmpty() || state.sending || !sameSession() || !state.settings.optBoolean("configured")) return;
        String conversationID = state.currentID;
        String oldDraftKey = draftKey();
        boolean resources = state.includeResources;
        state.sending = true;
        state.error = "";
        render();
        if (conversationID.isEmpty()) {
            Api.async(() -> {
                if (!sameSession()) throw new IllegalStateException("会话已变更");
                return Api.post("/api/assistant/conversations", new JSONObject());
            }, created -> {
                if (!sameSession()) return;
                merge(created);
                state.currentID = created.optString("id");
                state.drafts.put(state.currentID, state.drafts.getOrDefault(oldDraftKey, text));
                state.drafts.remove(oldDraftKey);
                performSend(state.currentID, text, resources);
            }, failure -> { if (sameSession()) { state.sending = false; state.error = failure.getMessage(); render(); } });
        } else performSend(conversationID, text, resources);
    }

    private void performSend(String conversationID, String text, boolean resources) {
        Api.async(() -> {
            if (!sameSession()) throw new IllegalStateException("会话已变更");
            return Api.post("/api/assistant/conversations/" + conversationID + "/messages", new JSONObject().put("message", text).put("include_resources", resources));
        }, conversation -> {
            if (!sameSession()) return;
            state.sending = false;
            merge(conversation);
            if (text.equals(state.drafts.getOrDefault(conversationID, "").trim())) state.drafts.put(conversationID, "");
            restoreDraft();
            render();
            if (alive() && scroll != null) {
                JSONArray resultTasks = conversation.optJSONArray("tasks");
                if (resultTasks != null && resultTasks.length() > 0) revealTasks();
                else scroll.post(() -> { if (scroll != null) scroll.fullScroll(View.FOCUS_DOWN); });
            }
        }, failure -> { if (sameSession()) { state.sending = false; state.error = failure.getMessage(); render(); } });
    }

    private void showHistory() {
        if (state.sending) return;
        Sheet.Builder sheet = Sheet.of(requireContext(), "我的会话");
        sheet.action(R.drawable.ic_nav_plus, "新建会话", "已有会话会保留", () -> {
            if (state.sending) return;
            state.currentID = "";
            restoreDraft();
            render();
        });
        for (int index = 0; index < state.conversations.length(); index++) {
            JSONObject conversation = state.conversations.optJSONObject(index);
            String id = conversation.optString("id");
            sheet.action(R.drawable.ic_nav_assistant, conversation.optString("title"), "查看会话与任务", () -> {
                state.currentID = id;
                restoreDraft();
                render();
            });
        }
        if (!state.currentID.isEmpty()) sheet.action(R.drawable.ic_nav_close, "删除当前会话", "不删除已经创建的业务资源", () -> {
            String id = state.currentID;
            Sheet.Builder confirmation = Sheet.of(requireContext(), "删除会话与任务记录？");
            confirmation.content(UI.muted(requireContext(), "不会删除已创建的隧道、DNS 或监控。"));
            confirmation.action(R.drawable.ic_nav_close, "确认删除", "此操作无法撤销", () -> Api.async(() -> Api.delete("/api/assistant/conversations/" + id), result -> {
                if (sameSession()) { state.currentID = ""; state.drafts.remove(id); load(); }
            }, failure -> { if (sameSession()) { state.error = failure.getMessage(); render(); } }));
            confirmation.show();
        });
        sheet.show();
    }

    private void renderTasks() {
        firstPendingCard = null;
        taskSection = null;
        executeButton = null;
        String conversationID = state.currentID;
        JSONArray tasks = tasks();
        List<String> pendingIDs = new ArrayList<>();
        for (int index = 0; index < tasks.length(); index++) {
            JSONObject task = tasks.optJSONObject(index);
            if (task != null && "pending".equals(task.optString("status"))) pendingIDs.add(task.optString("id"));
        }
        selection.sync(conversationID, pendingIDs);
        taskButton.setText(pendingIDs.isEmpty() ? "任务 " + tasks.length() : "待确认 " + pendingIDs.size());
        taskButton.setTextColor(pendingIDs.isEmpty() ? Theme.p().mute : Theme.p().success);
        taskButton.setEnabled(tasks.length() > 0);
        if (tasks.length() == 0) return;
        LinearLayout heading = UI.row(requireContext());
        heading.addView(UI.text(requireContext(), "任务清单 · " + tasks.length(), 16, Theme.p().ink, Typeface.BOLD), new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1));
        TextView refresh = compactButton("刷新", UI.BTN_GHOST);
        refresh.setOnClickListener(view -> load());
        heading.addView(refresh);
        messages.addView(heading);
        taskSection = heading;
        messages.addView(UI.muted(requireContext(), "只执行勾选后再次确认的新增操作。不修改或删除现有资源，正在执行的请求无法撤回。"));
        for (int index = 0; index < tasks.length(); index++) {
            JSONObject task = tasks.optJSONObject(index);
            if (task == null) continue;
            String id = task.optString("id"), taskStatus = task.optString("status");
            LinearLayout card = UI.card(requireContext());
            card.setPadding(UI.dp(14), UI.dp(12), UI.dp(14), UI.dp(12));
            String title = task.optString("title") + " · " + (state.busy.contains(id) ? "处理中" : statusName(taskStatus));
            if ("pending".equals(taskStatus)) {
                if (firstPendingCard == null) firstPendingCard = card;
                CheckBox check = new CheckBox(requireContext());
                check.setText(title);
                check.setTextColor(Theme.p().ink);
                check.setTextSize(14);
                check.setMinHeight(UI.dp(48));
                check.setChecked(selection.contains(id));
                check.setEnabled(!state.busy.contains(id));
                check.setOnCheckedChangeListener((button, checked) -> { selection.select(id, checked); updateExecute(); });
                card.addView(check);
            } else {
                card.addView(UI.text(requireContext(), title, 14, Theme.p().ink, Typeface.BOLD));
            }
            JSONObject arguments = task.optJSONObject("arguments");
            card.addView(UI.muted(requireContext(), "变更前：不修改现有资源\n变更后：新增以下配置"));
            if (arguments != null) {
                Iterator<String> keys = arguments.keys();
                while (keys.hasNext()) {
                    String key = keys.next();
                    Object value = arguments.opt(key);
                    String display = value instanceof Boolean ? ((Boolean) value ? "开启" : "关闭") : String.valueOf(value);
                    TextView detail = UI.text(requireContext(), AssistantTaskSelection.argumentLabel(key) + "：" + display, 13, Theme.p().ink, Typeface.NORMAL);
                    detail.setTextIsSelectable(true);
                    card.addView(detail);
                }
                if ("bind_domain".equals(task.optString("tool"))) card.addView(UI.muted(requireContext(), "将服务公开到互联网，新增代理 CNAME → " + arguments.optString("tunnel_id") + ".cfargotunnel.com，TTL 自动，并添加源站路由。失败时需同时核对 DNS 与路由。"));
            }
            if (!task.optString("result").isEmpty()) card.addView(UI.muted(requireContext(), task.optString("result")));
            LinearLayout controls = UI.row(requireContext());
            if ("pending".equals(taskStatus)) addButton(controls, "暂停", () -> taskAction(conversationID, id, "pause", false, null));
            if ("paused".equals(taskStatus)) addButton(controls, "恢复", () -> taskAction(conversationID, id, "resume", false, null));
            if ("unknown".equals(taskStatus)) addButton(controls, "核对后重试", () -> {
                Sheet.Builder retry = Sheet.of(requireContext(), "请先核对实际资源");
                retry.content(UI.muted(requireContext(), "失败或超时不代表未生效。请先确认没有重复创建，再重试。"));
                retry.action(R.drawable.ic_nav_assistant, "已核对，确认重新执行", "可能创建资源或启动周期探测", () -> taskAction(conversationID, id, "retry", true, null));
                retry.show();
            });
            if (!"running".equals(taskStatus) && !"succeeded".equals(taskStatus) && !"cancelled".equals(taskStatus)) addButton(controls, "取消任务", () -> taskAction(conversationID, id, "cancel", false, null));
            for (int control = 0; control < controls.getChildCount(); control++) controls.getChildAt(control).setEnabled(!state.busy.contains(id));
            if (controls.getChildCount() > 0) card.addView(controls);
            messages.addView(card);
            UI.margin(card, 0, UI.SM, 0, UI.SM);
        }
        if (!pendingIDs.isEmpty()) {
            executeButton = UI.button(requireContext(), "核对并执行 0 项", UI.BTN_PRIMARY);
            executeButton.setOnClickListener(view -> confirmSelected());
            messages.addView(executeButton, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));
            UI.margin(executeButton, 0, UI.SM, 0, UI.SM);
            updateExecute();
        }
    }

    private void updateExecute() {
        if (executeButton == null) return;
        executeButton.setText("核对并执行 " + selection.size() + " 项");
        executeButton.setEnabled(!selection.snapshot(state.currentID, !state.busy.isEmpty()).isEmpty());
        executeButton.setAlpha(executeButton.isEnabled() ? 1f : .5f);
    }

    private void confirmSelected() {
        String conversationID = state.currentID;
        List<String> approved = selection.snapshot(conversationID, !state.busy.isEmpty());
        if (approved.isEmpty() || !sameSession()) return;
        Sheet.Builder confirm = Sheet.of(requireContext(), "确认执行 " + approved.size() + " 项操作？");
        confirm.content(UI.muted(requireContext(), "域名绑定将把服务公开到互联网。可能创建收费资源；监控目标会开始周期探测。执行中的请求不能撤回。"));
        confirm.action(R.drawable.ic_nav_assistant, "确认执行", "仅执行已勾选任务", () -> {
            selection.clear();
            updateExecute();
            executeNext(conversationID, approved);
        });
        confirm.show();
    }

    private void revealTasks() {
        if (!alive() || scroll == null || taskSection == null) return;
        if (input != null) {
            InputMethodManager keyboard = (InputMethodManager) requireContext().getSystemService(android.content.Context.INPUT_METHOD_SERVICE);
            if (keyboard != null) keyboard.hideSoftInputFromWindow(input.getWindowToken(), 0);
            input.clearFocus();
        }
        scroll.post(() -> {
            View target = firstPendingCard == null ? taskSection : firstPendingCard;
            if (scroll != null && target != null) scroll.smoothScrollTo(0, Math.max(0, target.getTop() - UI.dp(8)));
        });
    }

    private void executeNext(String conversationID, List<String> queue) {
        if (queue.isEmpty() || !sameSession()) return;
        String id = queue.remove(0);
        if (!isPendingTask(conversationID, id)) { executeNext(conversationID, queue); return; }
        taskAction(conversationID, id, "execute", true, () -> executeNext(conversationID, queue));
    }

    private boolean isPendingTask(String conversationID, String taskID) {
        for (int index = 0; index < state.conversations.length(); index++) {
            JSONObject conversation = state.conversations.optJSONObject(index);
            if (conversation == null || !conversationID.equals(conversation.optString("id"))) continue;
            JSONArray tasks = conversation.optJSONArray("tasks");
            if (tasks == null) return false;
            for (int taskIndex = 0; taskIndex < tasks.length(); taskIndex++) {
                JSONObject task = tasks.optJSONObject(taskIndex);
                if (task != null && taskID.equals(task.optString("id"))) return "pending".equals(task.optString("status"));
            }
        }
        return false;
    }

    private void taskAction(String conversationID, String id, String action, boolean confirmed, Runnable complete) {
        if (!sameSession() || !state.busy.add(id)) return;
        render();
        Api.async(() -> {
            if (!sameSession()) throw new IllegalStateException("会话已变更");
            return Api.post("/api/assistant/conversations/" + conversationID + "/tasks/" + id, new JSONObject().put("action", action).put("confirmed", confirmed).put("checked_resources", "retry".equals(action)));
        }, result -> {
            if (!sameSession()) return;
            state.busy.remove(id);
            for (int conversationIndex = 0; conversationIndex < state.conversations.length(); conversationIndex++) {
                JSONObject conversation = state.conversations.optJSONObject(conversationIndex);
                if (!conversationID.equals(conversation.optString("id"))) continue;
                JSONArray tasks = conversation.optJSONArray("tasks");
                if (tasks == null) continue;
                for (int index = 0; index < tasks.length(); index++) {
                    if (id.equals(tasks.optJSONObject(index).optString("id"))) try { tasks.put(index, result); } catch (Exception ignored) { }
                }
            }
            render();
            if (complete != null) complete.run();
        }, failure -> { if (sameSession()) { state.busy.remove(id); state.error = failure.getMessage() + "；执行结果不确定时请先核对资源。"; render(); load(); } });
    }

    private void showSettings() { showSettings(state.settings.optBoolean("shared_enabled") && state.settings.optBoolean("is_admin")); }
    private void showSettings(boolean sharedScope) {
        Sheet.Builder sheet = Sheet.of(requireContext(), "AI 连接配置");
        boolean admin = state.settings.optBoolean("is_admin");
        if (state.settings.optBoolean("shared_enabled") && !admin) {
            sheet.content(UI.muted(requireContext(), "由管理员提供共享连接。端点与密钥不可查看，会话和任务仍仅属于你。"));
            sheet.show();
            return;
        }
        JSONObject connection = state.settings.optJSONObject(sharedScope ? "shared" : "personal");
        if (connection == null) connection = new JSONObject();
        LinearLayout form = UI.column(requireContext());
        CheckBox shared = new CheckBox(requireContext());
        shared.setText("启用全站共享");
        shared.setTextColor(Theme.p().ink);
        shared.setChecked(state.settings.optBoolean("shared_enabled"));
        EditText endpoint = UI.input(requireContext(), "https://api.example.com/v1");
        endpoint.setText(connection.optString("endpoint"));
        form.addView(UI.field(requireContext(), "API 端点", endpoint));
        EditText model = UI.input(requireContext(), "服务商支持的模型 ID");
        model.setText(connection.optString("model"));
        form.addView(UI.field(requireContext(), "模型名称", model));
        EditText key = UI.input(requireContext(), connection.optBoolean("key_set") ? "已设置，留空保留原密钥" : "API 密钥");
        key.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_PASSWORD);
        key.setSaveEnabled(false);
        key.setImportantForAutofill(View.IMPORTANT_FOR_AUTOFILL_NO);
        form.addView(UI.field(requireContext(), "API 密钥", key));
        form.addView(UI.muted(requireContext(), "兼容 Chat Completions 与工具调用。支持 HTTP/HTTPS、本机或内网地址及自定义端口；localhost 按后端所在主机或容器解析。HTTP 明文传输密钥与对话，仅在可信网络使用，公网建议 HTTPS。密钥服务端加密、不回显；更换端点需重新填写。"));
        TextView error = UI.muted(requireContext(), "");
        form.addView(error);
        TextView save = UI.button(requireContext(), "保存配置", UI.BTN_PRIMARY);
        save.setOnClickListener(view -> {
            JSONObject payload = new JSONObject();
            try {
                payload.put("scope", sharedScope ? "shared" : "personal").put("endpoint", endpoint.getText().toString().trim()).put("model", model.getText().toString().trim()).put("api_key", key.getText().toString());
                if (sharedScope) payload.put("shared_enabled", shared.isChecked());
            } catch (Exception ignored) { return; }
            save.setEnabled(false);
            sheet.setDismissible(false);
            Api.async(() -> {
                if (!sameSession()) throw new IllegalStateException("会话已变更");
                return Api.put("/api/assistant/settings", payload);
            }, result -> {
                key.setText("");
                sheet.setDismissible(true);
                sheet.dismiss();
                if (sameSession()) { state.settings = result; state.error = ""; render(); }
            }, failure -> { key.setText(""); sheet.setDismissible(true); save.setEnabled(true); error.setText(failure.getMessage()); });
        });
        sheet.content(form).footer(save).onDismiss(() -> key.setText("")).show();
    }

    private void addButton(LinearLayout row, String title, Runnable action) {
        TextView button = UI.button(requireContext(), title, UI.BTN_GHOST);
        button.setOnClickListener(view -> action.run());
        row.addView(button, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1));
    }
    private TextView compactButton(String title, int kind) {
        TextView button = UI.button(requireContext(), title, kind);
        button.setMinHeight(UI.dp(44));
        button.setPadding(UI.dp(10), UI.dp(6), UI.dp(10), UI.dp(6));
        button.setTextSize(12);
        return button;
    }
    private static String statusName(String status) {
        switch (status) {
            case "pending": return "待确认";
            case "paused": return "已暂停";
            case "running": return "执行中";
            case "succeeded": return "已完成";
            case "unknown": return "需核对";
            default: return "已取消";
        }
    }
    @Override public void onResume() { super.onResume(); poller.postDelayed(this::poll, 5000); }
    private void poll() {
        if (!alive() || !sameSession()) return;
        boolean running = false;
        JSONArray tasks = tasks();
        for (int index = 0; index < tasks.length(); index++) if ("running".equals(tasks.optJSONObject(index).optString("status"))) running = true;
        if (running && state.busy.isEmpty()) load();
        poller.postDelayed(this::poll, 5000);
    }
    @Override public void onPause() { poller.removeCallbacksAndMessages(null); super.onPause(); }
    @Override public void onDestroyView() {
        poller.removeCallbacksAndMessages(null);
        selection.clear();
        renderedConversationID = "";
        renderedTaskCount = 0;
        messages = null;
        input = null;
        scroll = null;
        send = null;
        status = null;
        historyButton = null;
        taskButton = null;
        executeButton = null;
        taskSection = null;
        firstPendingCard = null;
        super.onDestroyView();
    }
}
