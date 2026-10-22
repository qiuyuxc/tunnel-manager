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
    static final class LiveTurn {
        String conversationID, message, content = "", reasoning = "", status = I18n.text("m_22eb0f8779ab"), error = "";
        boolean active = true, expanded, resources;
    }
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
        final List<LiveTurn> turns = new ArrayList<>();
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
    private boolean streamRenderScheduled;
    private final Handler poller = new Handler(Looper.getMainLooper());

    @Override String route() { return "/assistant"; }

    @Override protected View build(@NonNull LayoutInflater inflater, @Nullable ViewGroup container) {
        state = console().assistantState;
        LinearLayout root = UI.column(requireContext());
        root.setPadding(UI.dp(12), UI.dp(8), UI.dp(12), UI.dp(8));
        root.setFocusableInTouchMode(true);
        LinearLayout header = UI.row(requireContext());
        header.addView(UI.text(requireContext(), I18n.text("m_1b0049fcfd41"), 22, Theme.p().ink, Typeface.BOLD), new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1));
        historyButton = compactButton(I18n.text("m_a63280253f17"), UI.BTN_GHOST);
        historyButton.setOnClickListener(view -> showHistory());
        header.addView(historyButton);
        TextView settings = compactButton(I18n.text("m_a5574109f020"), UI.BTN_GHOST);
        settings.setContentDescription(I18n.text("m_036413404922"));
        settings.setOnClickListener(view -> showSettings());
        header.addView(settings);
        root.addView(header);
        LinearLayout summary = UI.row(requireContext());
        status = UI.muted(requireContext(), I18n.text("m_9db4ea0e84ae"));
        status.setTextSize(11);
        summary.addView(status, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1));
        taskButton = compactButton(I18n.text("m_5ccf82a671d0"), UI.BTN_GHOST);
        taskButton.setOnClickListener(view -> revealTasks());
        summary.addView(taskButton);
        root.addView(summary);
        scroll = new ScrollView(requireContext());
        messages = UI.column(requireContext());
        scroll.addView(messages);
        root.addView(scroll, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, 0, 1));
        LinearLayout options = UI.row(requireContext());
        CheckBox resources = new CheckBox(requireContext());
        resources.setText(I18n.text("m_03e9099ca5f2"));
        resources.setMinHeight(UI.dp(44));
        resources.setTextColor(Theme.p().mute);
        resources.setTextSize(11);
        resources.setChecked(state.includeResources);
        resources.setOnCheckedChangeListener((button, checked) -> state.includeResources = checked);
        options.addView(resources, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1));
        TextView privacy = compactButton(I18n.text("m_9d06d61ee41f"), UI.BTN_GHOST);
        privacy.setTextSize(11);
        privacy.setOnClickListener(view -> Sheet.of(requireContext(), I18n.text("m_de66e65c470b"))
                .content(UI.muted(requireContext(), I18n.text("m_6a78a7d91819")))
                .show());
        options.addView(privacy);
        root.addView(options);
        LinearLayout composer = UI.row(requireContext());
        composer.setGravity(Gravity.BOTTOM);
        input = UI.textArea(requireContext(), I18n.text("m_c3a37f287970"));
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
        input.setContentDescription(I18n.text("m_edcac8607f16"));
        input.setText(state.drafts.getOrDefault(draftKey(), ""));
        input.addTextChangedListener(new TextWatcher() {
            public void beforeTextChanged(CharSequence value, int start, int count, int after) { }
            public void onTextChanged(CharSequence value, int start, int before, int count) { state.drafts.put(draftKey(), value.toString()); }
            public void afterTextChanged(Editable value) { updateSend(); }
        });
        composer.addView(input, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1));
        send = compactButton(I18n.text("m_edecf0ae6e51"), UI.BTN_PRIMARY);
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
            if (!sameSession()) throw new IllegalStateException(I18n.text("m_c99b9db61d79"));
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
        status.setText((state.settings.optBoolean("shared_enabled") ? I18n.text("m_cb87e5beac35") : I18n.text("m_07240a74e246")) + I18n.text("m_2e8adec50be7") + (state.settings.optBoolean("configured") ? "" : I18n.text("m_99cad4f6e190")));
        messages.removeAllViews();
        if (!state.error.isEmpty()) {
            TextView error = UI.text(requireContext(), state.error, 13, Theme.p().error, Typeface.NORMAL);
            error.setAccessibilityLiveRegion(View.ACCESSIBILITY_LIVE_REGION_POLITE);
            messages.addView(error);
        }
        JSONArray history = current().optJSONArray("messages");
        boolean hasLiveTurn = false;
        for (LiveTurn turn : state.turns) if (turn.conversationID.equals(state.currentID)) hasLiveTurn = true;
        if ((history == null || history.length() == 0) && !hasLiveTurn) {
            messages.addView(UI.text(requireContext(), I18n.text("m_ceb885e52275"), 22, Theme.p().ink, Typeface.BOLD));
            messages.addView(UI.muted(requireContext(), I18n.text("m_1d096551d401")));
            for (String prompt : new String[]{I18n.text("m_b1eeed66e829"), I18n.text("m_eb810bee3888"), I18n.text("m_838187f984bc")}) {
                TextView suggestion = UI.button(requireContext(), prompt, UI.BTN_SECONDARY);
                suggestion.setGravity(Gravity.START | Gravity.CENTER_VERTICAL);
                suggestion.setTextSize(12);
                suggestion.setOnClickListener(view -> input.setText(prompt));
                messages.addView(suggestion, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));
                UI.margin(suggestion, 0, UI.SM, 0, 0);
            }
        } else if (history != null) {
            for (int index = 0; index < history.length(); index++) {
                JSONObject message = history.optJSONObject(index);
                LinearLayout card = UI.card(requireContext());
                card.addView(UI.label(requireContext(), "user".equals(message.optString("role")) ? I18n.text("m_a0c7716669b5") : I18n.text("m_1b0049fcfd41")));
                TextView text = UI.text(requireContext(), message.optString("content"), 14, Theme.p().ink, Typeface.NORMAL);
                text.setTextIsSelectable(true);
                if ("assistant".equals(message.optString("role")) && !message.optString("reasoning").isEmpty()) addThinking(card, message.optString("reasoning"), "", false, null);
                card.addView(text);
                messages.addView(card);
                UI.margin(card, 0, UI.SM, 0, UI.SM);
            }
        }
        for (LiveTurn turn : state.turns) {
            if (!turn.conversationID.equals(state.currentID)) continue;
            LinearLayout userCard = UI.card(requireContext());
            userCard.addView(UI.label(requireContext(), I18n.text("m_a0c7716669b5")));
            userCard.addView(UI.text(requireContext(), turn.message, 14, Theme.p().ink, Typeface.NORMAL));
            messages.addView(userCard); UI.margin(userCard, 0, UI.SM, 0, UI.SM);
            LinearLayout reply = UI.card(requireContext());
            reply.addView(UI.label(requireContext(), I18n.text("m_1b0049fcfd41")));
            addThinking(reply, turn.reasoning, turn.status, turn.active, turn);
            if (!turn.content.isEmpty()) reply.addView(UI.text(requireContext(), turn.content, 14, Theme.p().ink, Typeface.NORMAL));
            if (!turn.error.isEmpty()) {
                reply.addView(UI.muted(requireContext(), turn.error));
                TextView retry = UI.button(requireContext(), I18n.text("m_723588bf300c"), UI.BTN_SECONDARY);
                retry.setEnabled(!state.sending);
                retry.setOnClickListener(view -> { if (!state.sending) startTurn(turn); });
                reply.addView(retry);
            }
            messages.addView(reply); UI.margin(reply, 0, UI.SM, 0, UI.SM);
        }
        renderTasks();
        if (state.sending) messages.addView(UI.muted(requireContext(), I18n.text("m_44bb7b76e0d3")));
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
        send.setText(state.sending ? I18n.text("m_64088d8cd78a") : I18n.text("m_edecf0ae6e51"));
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

    private void addThinking(LinearLayout card, String reasoning, String phase, boolean active, LiveTurn turn) {
        if (!active && reasoning.isEmpty()) return;
        TextView toggle = UI.button(requireContext(), (active ? I18n.text("m_dfd317d5828c") : I18n.text("m_c71d5942ebbc")) + "  ⌄", UI.BTN_GHOST);
        TextView detail = UI.muted(requireContext(), reasoning.isEmpty() ? phase : reasoning);
        detail.setMaxLines(12); detail.setMovementMethod(new android.text.method.ScrollingMovementMethod());
        detail.setTextIsSelectable(true);
        detail.setVisibility(turn != null && turn.expanded ? View.VISIBLE : View.GONE);
        toggle.setOnClickListener(view -> { boolean expanded = detail.getVisibility() != View.VISIBLE; detail.setVisibility(expanded ? View.VISIBLE : View.GONE); if (turn != null) turn.expanded = expanded; });
        card.addView(toggle); card.addView(detail);
    }

    private void sendMessage() {
        String text = input.getText().toString().trim();
        if (text.isEmpty() || state.sending || !sameSession() || !state.settings.optBoolean("configured")) return;
        LiveTurn turn = new LiveTurn();
        turn.conversationID = state.currentID; turn.message = text; turn.resources = state.includeResources;
        state.turns.add(turn);
        state.drafts.put(draftKey(), "");
        input.setText("");
        startTurn(turn);
    }

    private void startTurn(LiveTurn turn) {
        state.sending = true; state.error = "";
        turn.active = true; turn.error = ""; turn.content = ""; turn.reasoning = ""; turn.status = I18n.text("m_22eb0f8779ab");
        render();
        if (alive() && scroll != null) scroll.post(() -> { if (scroll != null) scroll.fullScroll(View.FOCUS_DOWN); });
        if (turn.conversationID.isEmpty()) {
            Api.async(() -> { if (!sameSession()) throw new IllegalStateException(I18n.text("m_c99b9db61d79")); return Api.post("/api/assistant/conversations", new JSONObject()); }, created -> {
                if (!sameSession()) return;
                merge(created); state.currentID = created.optString("id"); turn.conversationID = state.currentID;
                String draft = state.drafts.remove("new"); if (draft != null) state.drafts.put(state.currentID, draft);
                performSend(turn);
            }, failure -> { if (sameSession()) { state.sending = false; turn.active = false; turn.error = failure.getMessage(); render(); } });
        } else performSend(turn);
    }

    private void performSend(LiveTurn turn) {
        RequestScope scope = new RequestScope(Session::snapshot);
        Api.async(() -> Api.streamAssistant(scope, "/api/assistant/conversations/" + turn.conversationID + "/messages", new JSONObject().put("message", turn.message).put("include_resources", turn.resources).put("stream", true), event -> {
            if (!sameSession()) return;
            String kind = event.optString("event");
            if ("text".equals(kind)) { turn.content += event.optString("delta"); turn.status = I18n.text("m_13f41d1066f6"); }
            if ("reasoning".equals(kind)) turn.reasoning += event.optString("delta");
            if ("status".equals(kind)) turn.status = event.optString("message");
            if (!"ping".equals(kind) && !streamRenderScheduled) {
                streamRenderScheduled = true;
                poller.postDelayed(() -> { streamRenderScheduled = false; boolean follow = alive() && scroll != null && messages != null && messages.getHeight() - scroll.getScrollY() - scroll.getHeight() < UI.dp(140); render(); if (follow && scroll != null) scroll.post(() -> { if (scroll != null) scroll.fullScroll(View.FOCUS_DOWN); }); }, 60);
            }
        }), conversation -> {
            if (!sameSession()) return;
            state.sending = false; state.turns.remove(turn); merge(conversation); render();
            if (alive() && scroll != null) { JSONArray tasks = conversation.optJSONArray("tasks"); if (tasks != null && tasks.length() > 0) revealTasks(); else scroll.post(() -> { if (scroll != null) scroll.fullScroll(View.FOCUS_DOWN); }); }
        }, failure -> { if (sameSession()) { state.sending = false; turn.active = false; turn.error = failure.getMessage(); turn.status = I18n.text("m_a903ba39af8f"); render(); } });
    }

    private void showHistory() {
        if (state.sending) return;
        Sheet.Builder sheet = Sheet.of(requireContext(), I18n.text("m_ed008a400d30"));
        sheet.action(R.drawable.ic_nav_plus, I18n.text("m_58e21b8737cc"), I18n.text("m_cfa6cece711c"), () -> {
            if (state.sending) return;
            state.currentID = "";
            restoreDraft();
            render();
        });
        for (int index = 0; index < state.conversations.length(); index++) {
            JSONObject conversation = state.conversations.optJSONObject(index);
            String id = conversation.optString("id");
            sheet.action(R.drawable.ic_nav_assistant, conversation.optString("title"), I18n.text("m_e978876d1ad6"), () -> {
                state.currentID = id;
                restoreDraft();
                render();
            });
        }
        if (!state.currentID.isEmpty()) sheet.action(R.drawable.ic_nav_close, I18n.text("m_3c50ed52eef5"), I18n.text("m_3bc480a5db25"), () -> {
            String id = state.currentID;
            Sheet.Builder confirmation = Sheet.of(requireContext(), I18n.text("m_6a825121f6bc"));
            confirmation.content(UI.muted(requireContext(), I18n.text("m_ac27ba239983")));
            confirmation.action(R.drawable.ic_nav_close, I18n.text("m_a3ea3c17b401"), I18n.text("m_1f1a8e3dece2"), () -> Api.async(() -> Api.delete("/api/assistant/conversations/" + id), result -> {
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
        taskButton.setText(pendingIDs.isEmpty() ? I18n.text("m_caa71bda0f8f") + tasks.length() : I18n.text("m_b95b857b5205") + pendingIDs.size());
        taskButton.setTextColor(pendingIDs.isEmpty() ? Theme.p().mute : Theme.p().success);
        taskButton.setEnabled(tasks.length() > 0);
        if (tasks.length() == 0) return;
        LinearLayout heading = UI.row(requireContext());
        heading.addView(UI.text(requireContext(), I18n.text("m_7e44c72a7dae") + tasks.length(), 16, Theme.p().ink, Typeface.BOLD), new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1));
        TextView refresh = compactButton(I18n.text("m_aee887434131"), UI.BTN_GHOST);
        refresh.setOnClickListener(view -> load());
        heading.addView(refresh);
        messages.addView(heading);
        taskSection = heading;
        messages.addView(UI.muted(requireContext(), I18n.text("m_389f2c7a0beb")));
        for (int index = 0; index < tasks.length(); index++) {
            JSONObject task = tasks.optJSONObject(index);
            if (task == null) continue;
            String id = task.optString("id"), taskStatus = task.optString("status");
            LinearLayout card = UI.card(requireContext());
            card.setPadding(UI.dp(14), UI.dp(12), UI.dp(14), UI.dp(12));
            String title = task.optString("title") + " · " + (state.busy.contains(id) ? I18n.text("m_694b71bc8013") : statusName(taskStatus));
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
            JSONObject before = task.optJSONObject("before");
            boolean deleting = task.optString("tool").startsWith("delete_");
            if (before != null) {
                card.addView(UI.muted(requireContext(), I18n.text("m_39f615acdc2b")));
                Iterator<String> beforeKeys = before.keys();
                while (beforeKeys.hasNext()) {
                    String key = beforeKeys.next();
                    Object value = before.opt(key);
                    String display = value instanceof Boolean ? ((Boolean) value ? I18n.text("m_8da97ddda990") : I18n.text("m_3fd47edce45b")) : String.valueOf(value);
                    TextView detail = UI.text(requireContext(), AssistantTaskSelection.argumentLabel(key) + "：" + display, 13, Theme.p().ink, Typeface.NORMAL);
                    detail.setTextIsSelectable(true);
                    card.addView(detail);
                }
            }
            card.addView(UI.muted(requireContext(), deleting ? I18n.text("m_002c9310d60f") : before != null ? I18n.text("m_5a190cb19502") : I18n.text("m_ea02a85652f7")));
            if (arguments != null) {
                Iterator<String> keys = arguments.keys();
                while (keys.hasNext()) {
                    String key = keys.next();
                    Object value = arguments.opt(key);
                    String display = value instanceof Boolean ? ((Boolean) value ? I18n.text("m_8da97ddda990") : I18n.text("m_3fd47edce45b")) : String.valueOf(value);
                    TextView detail = UI.text(requireContext(), AssistantTaskSelection.argumentLabel(key) + "：" + display, 13, Theme.p().ink, Typeface.NORMAL);
                    detail.setTextIsSelectable(true);
                    card.addView(detail);
                }
                if ("bind_domain".equals(task.optString("tool"))) card.addView(UI.muted(requireContext(), I18n.text("m_581906cf0bd4") + arguments.optString("tunnel_id") + I18n.text("m_fb99768f60b1")));
            }
            if (!task.optString("result").isEmpty()) card.addView(UI.muted(requireContext(), task.optString("result")));
            LinearLayout controls = UI.row(requireContext());
            if ("pending".equals(taskStatus) && !task.optString("tool").startsWith("delete_")) addButton(controls, I18n.text("m_87615fbf148a"), () -> editTask(conversationID, task));
            if ("pending".equals(taskStatus)) addButton(controls, I18n.text("m_8d12fc0d4eb2"), () -> taskAction(conversationID, id, "pause", false, null));
            if ("paused".equals(taskStatus)) addButton(controls, I18n.text("m_e0534b8a4e46"), () -> taskAction(conversationID, id, "resume", false, null));
            if ("unknown".equals(taskStatus)) addButton(controls, I18n.text("m_276ce6b7b87a"), () -> {
                int revision = task.optInt("revision", 0);
                Sheet.Builder retry = Sheet.of(requireContext(), I18n.text("m_261d556d04c1"));
                retry.content(UI.muted(requireContext(), I18n.text("m_403c05a15d9a")));
                retry.action(R.drawable.ic_nav_assistant, I18n.text("m_d26eea749abf"), I18n.text("m_1b33150543c1"), () -> taskAction(conversationID, id, "retry", true, revision, null));
                retry.show();
            });
            if (!"running".equals(taskStatus) && !"succeeded".equals(taskStatus) && !"cancelled".equals(taskStatus)) addButton(controls, I18n.text("m_537d17f1c531"), () -> taskAction(conversationID, id, "cancel", false, null));
            for (int control = 0; control < controls.getChildCount(); control++) controls.getChildAt(control).setEnabled(!state.busy.contains(id));
            if (controls.getChildCount() > 0) card.addView(controls);
            messages.addView(card);
            UI.margin(card, 0, UI.SM, 0, UI.SM);
        }
        if (!pendingIDs.isEmpty()) {
            executeButton = UI.button(requireContext(), I18n.text("m_6800fd12e118"), UI.BTN_PRIMARY);
            executeButton.setOnClickListener(view -> confirmSelected());
            messages.addView(executeButton, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));
            UI.margin(executeButton, 0, UI.SM, 0, UI.SM);
            updateExecute();
        }
    }

    private void updateExecute() {
        if (executeButton == null) return;
        executeButton.setText(I18n.text("m_8eef47626a3e") + selection.size() + I18n.text("m_f9d529eacdf3"));
        executeButton.setEnabled(!selection.snapshot(state.currentID, !state.busy.isEmpty()).isEmpty());
        executeButton.setAlpha(executeButton.isEnabled() ? 1f : .5f);
    }

    private void confirmSelected() {
        String conversationID = state.currentID;
        List<String> approved = selection.snapshot(conversationID, !state.busy.isEmpty());
        if (approved.isEmpty() || !sameSession()) return;
        Map<String, Integer> revisions = new HashMap<>();
        for (int i = 0; i < tasks().length(); i++) { JSONObject task = tasks().optJSONObject(i); if (task != null) revisions.put(task.optString("id"), task.optInt("revision", 0)); }
        Sheet.Builder confirm = Sheet.of(requireContext(), I18n.text("m_bab8ac410782") + approved.size() + I18n.text("m_3fbf7b0f5a92"));
        confirm.content(UI.muted(requireContext(), I18n.text("m_28557d2026fc")));
        confirm.action(R.drawable.ic_nav_assistant, I18n.text("m_68a35e224268"), I18n.text("m_f837f51681bf"), () -> {
            selection.clear();
            updateExecute();
            executeNext(conversationID, approved, revisions);
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

    private void executeNext(String conversationID, List<String> queue, Map<String, Integer> revisions) {
        if (queue.isEmpty() || !sameSession()) return;
        String id = queue.remove(0);
        if (!isPendingTask(conversationID, id)) { executeNext(conversationID, queue, revisions); return; }
        taskAction(conversationID, id, "execute", true, revisions.getOrDefault(id, 0), () -> executeNext(conversationID, queue, revisions));
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
        taskAction(conversationID, id, action, confirmed, 0, complete);
    }

    private void taskAction(String conversationID, String id, String action, boolean confirmed, int revision, Runnable complete) {
        if (!sameSession() || !state.busy.add(id)) return;
        render();
        Api.async(() -> {
            if (!sameSession()) throw new IllegalStateException(I18n.text("m_c99b9db61d79"));
            return Api.post("/api/assistant/conversations/" + conversationID + "/tasks/" + id, new JSONObject().put("action", action).put("confirmed", confirmed).put("checked_resources", "retry".equals(action)).put("expected_revision", revision));
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
        }, failure -> { if (sameSession()) { state.busy.remove(id); state.error = failure.getMessage() + I18n.text("m_b75bac006d14"); render(); load(); } });
    }

    private void editTask(String conversationID, JSONObject task) {
        String id = task.optString("id");
        int revision = task.optInt("revision", 0);
        selection.select(id, false);
        updateExecute();
        final AssistantTaskEditor editor;
        try { editor = new AssistantTaskEditor(requireContext(), task); }
        catch (Exception invalid) { state.error = I18n.text("m_d345e1fe6071"); render(); return; }
        Sheet.Builder sheet = Sheet.of(requireContext(), I18n.text("m_32b1a87a33e8"));
        TextView error = UI.muted(requireContext(), "");
        LinearLayout content = UI.column(requireContext());
        content.addView(editor.form); content.addView(error);
        TextView save = UI.button(requireContext(), I18n.text("m_991bb7cfe5a8"), UI.BTN_PRIMARY);
        save.setOnClickListener(view -> {
            final JSONObject payload;
            try { payload = new JSONObject().put("action", "edit").put("expected_revision", revision).put("arguments", new JSONObject(editor.draft.toString())); }
            catch (Exception invalid) { error.setText(I18n.text("m_161364fcc2b4")); return; }
            if (!sameSession() || !state.busy.add(id)) return;
            save.setEnabled(false); editor.setEnabled(false); sheet.setDismissible(false);
            Api.async(() -> {
                if (!sameSession()) throw new IllegalStateException(I18n.text("m_c99b9db61d79"));
                return Api.post("/api/assistant/conversations/" + conversationID + "/tasks/" + id, payload);
            }, result -> {
                sheet.setDismissible(true); sheet.dismiss();
                if (sameSession()) { state.busy.remove(id); state.error = ""; load(); }
            }, failure -> {
                sheet.setDismissible(true); save.setEnabled(true); editor.setEnabled(true);
                if (sameSession()) { state.busy.remove(id); error.setText(failure.getMessage()); }
            });
        });
        sheet.content(content).footer(save).show();
    }

    private void showSettings() { showSettings(state.settings.optBoolean("shared_enabled") && state.settings.optBoolean("is_admin")); }
    private void showSettings(boolean sharedScope) {
        Sheet.Builder sheet = Sheet.of(requireContext(), I18n.text("m_036413404922"));
        boolean admin = state.settings.optBoolean("is_admin");
        if (state.settings.optBoolean("shared_enabled") && !admin) {
            sheet.content(UI.muted(requireContext(), I18n.text("m_7b5dd092dde7")));
            sheet.show();
            return;
        }
        if (admin) sheet.action(R.drawable.ic_nav_settings, sharedScope ? I18n.text("m_114b8e208909") : I18n.text("m_9bb7afb010cb"), I18n.text("m_aaf8d72517bc"), () -> showSettings(!sharedScope));
        JSONObject connection = state.settings.optJSONObject(sharedScope ? "shared" : "personal");
        if (connection == null) connection = new JSONObject();
        LinearLayout form = UI.column(requireContext());
        CheckBox shared = new CheckBox(requireContext());
        shared.setText(I18n.text("m_f6f63440e200"));
        shared.setTextColor(Theme.p().ink);
        shared.setChecked(state.settings.optBoolean("shared_enabled"));
        if (sharedScope) form.addView(shared);
        EditText endpoint = UI.input(requireContext(), "https://api.example.com/v1");
        endpoint.setText(connection.optString("endpoint"));
        form.addView(UI.field(requireContext(), I18n.text("m_3d1ab62be024"), endpoint));
        final String[] apiFormat = {connection.optString("api_format", "chat_completions")};
        form.addView(UI.label(requireContext(), I18n.text("m_77ff2042727b")));
        form.addView(UI.segmented(requireContext(), new String[]{I18n.text("m_0b1863617b31"), I18n.text("m_b193877d0270")}, "responses".equals(apiFormat[0]) ? 1 : 0, index -> apiFormat[0] = index == 1 ? "responses" : "chat_completions"));
        CheckBox reasoningSummary = new CheckBox(requireContext());
        reasoningSummary.setText(I18n.text("m_abfd70997635")); reasoningSummary.setTextColor(Theme.p().ink); reasoningSummary.setChecked(connection.optBoolean("reasoning_summary")); form.addView(reasoningSummary);
        EditText model = UI.input(requireContext(), I18n.text("m_5ec4a61c05d5"));
        model.setText(connection.optString("model"));
        form.addView(UI.field(requireContext(), I18n.text("m_cb2f1709f983"), model));
        EditText key = UI.input(requireContext(), connection.optBoolean("key_set") ? I18n.text("m_a025b0c0b478") : I18n.text("m_5f600b307b4e"));
        key.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_PASSWORD);
        key.setSaveEnabled(false);
        key.setImportantForAutofill(View.IMPORTANT_FOR_AUTOFILL_NO);
        form.addView(UI.field(requireContext(), I18n.text("m_5f600b307b4e"), key));
        form.addView(UI.muted(requireContext(), I18n.text("m_c03625923b75")));
        TextView error = UI.muted(requireContext(), "");
        form.addView(error);
        TextView save = UI.button(requireContext(), I18n.text("m_6e584e3d5ce6"), UI.BTN_PRIMARY);
        save.setOnClickListener(view -> {
            JSONObject payload = new JSONObject();
            try {
                payload.put("scope", sharedScope ? "shared" : "personal").put("endpoint", endpoint.getText().toString().trim()).put("model", model.getText().toString().trim()).put("api_key", key.getText().toString()).put("api_format", apiFormat[0]).put("reasoning_summary", reasoningSummary.isChecked());
                if (sharedScope) payload.put("shared_enabled", shared.isChecked());
            } catch (Exception ignored) { return; }
            save.setEnabled(false);
            sheet.setDismissible(false);
            Api.async(() -> {
                if (!sameSession()) throw new IllegalStateException(I18n.text("m_c99b9db61d79"));
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
            case "pending": return I18n.text("m_a8b0bd9c65f7");
            case "paused": return I18n.text("m_eb0c326b60ae");
            case "running": return I18n.text("m_5026a63b58cd");
            case "succeeded": return I18n.text("m_f28461bb49c8");
            case "unknown": return I18n.text("m_db6343cfca18");
            default: return I18n.text("m_a37778f17c5f");
        }
    }
    @Override public void onResume() { super.onResume(); if (sameSession()) render(); poller.postDelayed(this::poll, 5000); }
    private void poll() {
        if (!alive() || !sameSession()) return;
        boolean running = false;
        JSONArray tasks = tasks();
        for (int index = 0; index < tasks.length(); index++) if ("running".equals(tasks.optJSONObject(index).optString("status"))) running = true;
        if (running && state.busy.isEmpty()) load();
        poller.postDelayed(this::poll, 5000);
    }
    @Override public void onPause() { poller.removeCallbacksAndMessages(null); streamRenderScheduled = false; super.onPause(); }
    @Override public void onDestroyView() {
        poller.removeCallbacksAndMessages(null);
        streamRenderScheduled = false;
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
