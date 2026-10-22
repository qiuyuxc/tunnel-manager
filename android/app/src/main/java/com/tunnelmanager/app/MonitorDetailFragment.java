package com.tunnelmanager.app;

import android.content.ClipData;
import android.content.ClipboardManager;
import android.content.Context;
import android.graphics.Typeface;
import android.os.Bundle;
import android.text.Editable;
import android.text.InputType;
import android.text.TextWatcher;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.EditText;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.lifecycle.ViewModelProvider;
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout;

import org.json.JSONArray;
import org.json.JSONObject;

import java.util.function.Consumer;

public class MonitorDetailFragment extends PageFragment {
    private static final String ARG_ID = "id";
    private MonitorDetailState state;
    private SwipeRefreshLayout refresh;
    private ScrollView scroll;
    private LinearLayout body;
    private Sheet.Builder sheet;
    private Sheet.Builder picker;
    private LinearLayout formBody;
    private TextView formError;
    private TextView submit;
    private String displayedForm = "";
    private boolean destroyingView;

    static MonitorDetailFragment forId(String id) {
        MonitorDetailFragment fragment = new MonitorDetailFragment();
        Bundle arguments = new Bundle();
        arguments.putString(ARG_ID, id);
        fragment.setArguments(arguments);
        return fragment;
    }

    @Override public void onCreate(@Nullable Bundle saved) {
        super.onCreate(saved);
        state = new ViewModelProvider(this).get(MonitorDetailState.class);
        if (getArguments() != null) state.monitorId = getArguments().getString(ARG_ID, "");
        if (saved != null && state.monitor == null && !state.busy) {
            state.selectedId = saved.getString("monitor_selected", "");
            state.form = saved.getString("monitor_form", "");
            state.targetId = saved.getString("monitor_target", "");
            state.name = saved.getString("monitor_name", "");
            state.url = saved.getString("monitor_url", "");
            state.interval = saved.getString("monitor_interval", "60");
            state.type = Math.max(0, Math.min(2, saved.getInt("monitor_type")));
            state.method = Math.max(0, Math.min(1, saved.getInt("monitor_method")));
            if (saved.getBoolean("monitor_pending")) {
                state.formError = I18n.text("m_c10414d3163f");
                state.error = state.formError;
            }
        }
    }

    @Override public void onSaveInstanceState(@NonNull Bundle saved) {
        super.onSaveInstanceState(saved);
        saved.putString("monitor_selected", state.selectedId);
        saved.putString("monitor_form", state.form);
        saved.putString("monitor_target", state.targetId);
        saved.putString("monitor_name", state.name);
        saved.putString("monitor_url", state.url);
        saved.putString("monitor_interval", state.interval);
        saved.putInt("monitor_type", state.type);
        saved.putInt("monitor_method", state.method);
        saved.putBoolean("monitor_pending", state.busy);
    }

    @Override String route() {
        return "/monitors/" + (state == null ? requireArguments().getString(ARG_ID, "") : state.monitorId);
    }

    @Override protected View build(@NonNull LayoutInflater inflater, @Nullable ViewGroup container) {
        destroyingView = false;
        refresh = new SwipeRefreshLayout(requireContext());
        refresh.setOnRefreshListener(state::load);
        scroll = new ScrollView(requireContext());
        body = UI.column(requireContext());
        UI.pagePadding(body);
        scroll.addView(body);
        refresh.addView(scroll);
        return refresh;
    }

    @Override public void onViewCreated(@NonNull View view, @Nullable Bundle saved) {
        super.onViewCreated(view, saved);
        state.revision.observe(getViewLifecycleOwner(), ignored -> render());
    }

    @Override public void onResume() {
        super.onResume();
        state.load();
    }

    @Override public void onDestroyView() {
        destroyingView = true;
        if (sheet != null) sheet.dismiss();
        if (picker != null) picker.dismiss();
        sheet = null;
        picker = null;
        formBody = null;
        formError = null;
        submit = null;
        body = null;
        scroll = null;
        refresh = null;
        super.onDestroyView();
    }

    private void render() {
        if (body == null || !alive()) return;
        if (!state.notice.isEmpty()) {
            android.widget.Toast.makeText(requireContext(), state.notice, android.widget.Toast.LENGTH_SHORT).show();
            state.notice = "";
        }
        if (state.deleted) {
            openRoute("/monitors");
            return;
        }
        int offset = scroll.getScrollY();
        refresh.setRefreshing(state.loading);
        refresh.setEnabled(!state.busy);
        body.removeAllViews();
        JSONObject monitor = state.monitor;
        body.addView(UI.pageTitle(requireContext(), monitor == null ? I18n.text("m_ce0d1c8bffa4") : monitor.optString("name", I18n.text("m_ce0d1c8bffa4"))));
        if (!state.error.isEmpty()) {
            UI.addRow(body, UI.banner(requireContext(), state.error, true), UI.MD);
            UI.addRow(body, button(I18n.text("m_7bdd5ce1e298"), UI.BTN_SECONDARY, state::load), UI.SM);
        }
        if (monitor == null) {
            if (state.loading) UI.addRow(body, UI.muted(requireContext(), I18n.text("m_89cfe1364c02")), UI.MD);
            renderForm();
            return;
        }
        JSONArray targets = monitor.optJSONArray("targets");
        MonitorHistory.Counts counts = MonitorViews.counts(targets);
        UI.addRow(body, UI.muted(requireContext(), I18n.text("m_15bb0aa7dbc6") + counts.total + I18n.text("m_4c9145a693dc")
                + (monitor.optBoolean("publish_enabled") ? I18n.text("m_9f1e25609b1a") : I18n.text("m_33c7823a56bd"))), UI.XS);
        UI.addRow(body, trendCard(monitor), UI.LG);
        UI.addRow(body, button(state.busy && I18n.text("m_14ca09ce1fd2").equals(state.operation) ? I18n.text("m_e8b970bc3e05") : I18n.text("m_14ca09ce1fd2"),
                UI.BTN_PRIMARY, state::checkNow), UI.MD);
        UI.addRow(body, MonitorViews.heading(requireContext(), I18n.text("m_ec309ab207ef"), UI.muted(requireContext(), counts.total + I18n.text("m_7aaf96ad52e0"))), UI.LG);
        UI.addRow(body, targetsCard(targets), UI.SM);
        UI.addRow(body, button(I18n.text("m_6f760cbd09bb"), UI.BTN_SECONDARY, () -> state.openForm("add", null)), UI.SM);
        UI.addRow(body, UI.strong(requireContext(), I18n.text("m_e9ceec924cbf")), UI.LG);
        UI.addRow(body, settingsCard(monitor), UI.SM);
        UI.addRow(body, UI.strong(requireContext(), I18n.text("m_cd84121acb17")), UI.LG);
        UI.addRow(body, UI.muted(requireContext(), I18n.text("m_dc7165aa4ae4")), UI.SM);
        UI.addRow(body, button(I18n.text("m_b2fc749edfa3"), UI.BTN_DANGER, () -> askDelete(monitor.optString("name"))), UI.SM);
        UI.restoreScroll(scroll, offset);
        renderForm();
    }

    private View trendCard(JSONObject monitor) {
        LinearLayout card = UI.card(requireContext());
        JSONArray targets = monitor.optJSONArray("targets");
        JSONObject target = MonitorViews.selectedTarget(targets, state.selectedId);
        card.addView(MonitorViews.heading(requireContext(), I18n.text("m_0b3c9b53d11d"), UI.tag(requireContext(), I18n.text("m_2e88c6f01581"), false)));
        if (target == null) {
            UI.addRow(card, UI.muted(requireContext(), I18n.text("m_91b80573f4e1")), UI.MD);
            return card;
        }
        state.selectedId = target.optString("id");
        if (MonitorViews.counts(targets).total > 1) {
            UI.addRow(card, button(target.optString("name", I18n.text("m_ec309ab207ef")) + I18n.text("m_48f3c2008a29"), UI.BTN_GHOST,
                    () -> chooseTarget(targets)), UI.SM);
        } else {
            UI.addRow(card, UI.muted(requireContext(), target.optString("name", I18n.text("m_ec309ab207ef"))), UI.SM);
        }
        LinearLayout reading = UI.row(requireContext());
        TextView value = UI.text(requireContext(), MonitorViews.latency(target), 30, Theme.p().ink, Typeface.BOLD);
        value.setFontFeatureSettings("tnum");
        reading.addView(value, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        reading.addView(UI.statusPill(requireContext(), target.optString("state")));
        UI.addRow(card, reading, UI.SM);
        MonitorHistory history = MonitorViews.history(target);
        UI.addRow(card, new MonitorHistoryView(requireContext(), history, true, monitor.optInt("interval_sec")), UI.MD);
        UI.addRow(card, UI.muted(requireContext(), MonitorViews.range(history)), UI.XS);
        UI.addRow(card, UI.muted(requireContext(), history.samples.size() + I18n.text("m_bfb21351bb46")), UI.XS);
        UI.addRow(card, UI.muted(requireContext(), I18n.text("m_bba8ce728846") + MonitorViews.uptime(target, history)), UI.SM);
        return card;
    }

    private View targetsCard(JSONArray targets) {
        LinearLayout card = UI.card(requireContext());
        if (MonitorViews.counts(targets).total == 0) {
            card.addView(UI.strong(requireContext(), I18n.text("m_142a72654927")));
            UI.addRow(card, UI.muted(requireContext(), I18n.text("m_38d7edabc845")), UI.SM);
            return card;
        }
        int rendered = 0;
        for (int index = 0; index < targets.length(); index++) {
            JSONObject target = targets.optJSONObject(index);
            if (target == null) continue;
            if (rendered++ > 0) card.addView(UI.divider(requireContext()));
            View action = MonitorViews.action(requireContext(), R.drawable.ic_nav_monitor,
                    target.optString("name", I18n.text("m_ec309ab207ef")), target.optString("url"), () -> state.openForm("edit", target));
            action.setEnabled(!state.busy);
            card.addView(action);
            card.addView(MonitorViews.heading(requireContext(), MonitorViews.latency(target),
                    UI.statusPill(requireContext(), target.optString("state"))));
            UI.addRow(card, UI.muted(requireContext(), I18n.text("m_7c3b8d62df06")
                    + MonitorViews.uptime(target, MonitorViews.history(target))), UI.XS);
            if (!target.optString("error").isEmpty()) {
                UI.addRow(card, UI.text(requireContext(), target.optString("error"), 12, Theme.p().error, Typeface.NORMAL), UI.SM);
            }
        }
        return card;
    }

    private View settingsCard(JSONObject monitor) {
        LinearLayout card = UI.card(requireContext());
        View interval = MonitorViews.action(requireContext(), R.drawable.ic_nav_settings, I18n.text("m_d5138f083ff0"),
                I18n.text("m_0a97ef128fc4") + monitor.optInt("interval_sec") + I18n.text("m_2241468b7ec6"), () -> state.openForm("interval", null));
        interval.setEnabled(!state.busy);
        card.addView(interval);
        card.addView(UI.divider(requireContext()));
        View publish = MonitorViews.action(requireContext(), R.drawable.ic_nav_monitor, I18n.text("m_6e3331e264ce"),
                monitor.optBoolean("publish_enabled") ? I18n.text("m_2b8257bdb6f9") : I18n.text("m_ffd259cf9655"),
                () -> state.openForm("public", null));
        publish.setEnabled(!state.busy);
        card.addView(publish);
        return card;
    }

    private TextView button(String label, int kind, Runnable action) {
        TextView button = UI.button(requireContext(), label, kind);
        button.setSingleLine(false);
        UI.fill(button);
        button.setEnabled(!state.busy);
        button.setOnClickListener(view -> action.run());
        return button;
    }

    private void chooseTarget(JSONArray targets) {
        picker = Sheet.of(requireContext(), I18n.text("m_b44c04b58c8b"));
        for (int index = 0; index < targets.length(); index++) {
            JSONObject target = targets.optJSONObject(index);
            if (target == null) continue;
            picker.action(R.drawable.ic_nav_monitor, target.optString("name"), target.optString("url"), () -> {
                state.selectedId = target.optString("id");
                state.changed();
            });
        }
        picker.show();
    }

    private void renderForm() {
        if (state.form.isEmpty()) {
            if (sheet != null) sheet.dismiss();
            sheet = null;
            return;
        }
        if ("public".equals(state.form) && state.monitor == null) return;
        if (sheet == null || !sheet.isShowing() || !displayedForm.equals(state.form)) buildForm();
        sheet.setDismissible(!state.busy);
        enableTree(formBody, !state.busy);
        submit.setEnabled(!state.busy);
        String label = "add".equals(state.form) ? I18n.text("m_6f760cbd09bb") : I18n.text("m_991bb7cfe5a8");
        if ("public".equals(state.form)) label = state.monitor.optBoolean("publish_enabled") ? I18n.text("m_eaa8674cecad") : I18n.text("m_69d72c345aa1");
        submit.setText(state.busy ? state.operation + I18n.text("m_75e39da8d1d1") : label);
        formError.setText(state.formError);
        formError.setVisibility(state.formError.isEmpty() ? View.GONE : View.VISIBLE);
    }

    private void buildForm() {
        displayedForm = state.form;
        formBody = UI.column(requireContext());
        String title;
        if ("public".equals(state.form)) {
            title = I18n.text("m_6e3331e264ce");
            publicContent();
        } else if ("interval".equals(state.form)) {
            title = I18n.text("m_ee980d95ddb6");
            formBody.addView(UI.muted(requireContext(), I18n.text("m_1408793bfd69")));
            EditText interval = input("60", state.interval, value -> state.interval = value);
            interval.setInputType(InputType.TYPE_CLASS_NUMBER);
            formBody.addView(UI.field(requireContext(), I18n.text("m_5cf14716875d"), interval, UI.MD));
        } else {
            title = "add".equals(state.form) ? I18n.text("m_6f760cbd09bb") : I18n.text("m_a1944feec290");
            targetContent();
        }
        formError = UI.banner(requireContext(), "", true);
        formError.setAccessibilityLiveRegion(View.ACCESSIBILITY_LIVE_REGION_POLITE);
        UI.addRow(formBody, formError, UI.MD);
        if ("edit".equals(state.form)) {
            UI.addRow(formBody, button(I18n.text("m_df229463d6e0"), UI.BTN_DANGER, () -> Modal.of(requireContext(), I18n.text("m_70f2105700a9"))
                    .message(I18n.text("m_c843b8c4bd27") + state.name + I18n.text("m_001bd11835cd"))
                    .cancel(I18n.text("m_2cd0f3be8738")).confirm(I18n.text("m_ee7dea070d88"), true, state::removeTarget).show()), UI.LG);
        }
        submit = button(I18n.text("m_991bb7cfe5a8"), UI.BTN_PRIMARY, () -> {
            if ("public".equals(state.form)) state.setPublish(!state.monitor.optBoolean("publish_enabled"));
            else state.submitForm();
        });
        sheet = Sheet.of(requireContext(), title).content(formBody).footer(submit).onDismiss(() -> {
            if (!destroyingView) {
                state.form = "";
                state.formError = "";
            }
        });
        sheet.show();
    }

    private void publicContent() {
        formBody.addView(UI.muted(requireContext(), I18n.text("m_846f1d2c9473")));
        JSONObject monitor = state.monitor;
        boolean published = monitor.optBoolean("publish_enabled");
        UI.addRow(formBody, UI.strong(requireContext(), published ? I18n.text("m_052141af09cf") : I18n.text("m_f51c367762c6")), UI.MD);
        String token = monitor.optString("public_token");
        if (published && !token.isEmpty()) {
            String link = Session.server() + "/status/" + token;
            TextView url = UI.mono(requireContext(), link, Theme.p().link);
            url.setTextIsSelectable(true);
            UI.addRow(formBody, url, UI.MD);
            UI.addRow(formBody, button(I18n.text("m_d858f277030c"), UI.BTN_SECONDARY, () -> copyLink(link)), UI.SM);
        } else if (published) {
            UI.addRow(formBody, UI.muted(requireContext(), I18n.text("m_e51f91802c7b")), UI.SM);
        }
    }

    private void targetContent() {
        formBody.addView(UI.muted(requireContext(), "add".equals(state.form)
                ? I18n.text("m_2248af10ab71") : I18n.text("m_5de28a81ee68")));
        EditText name = input(I18n.text("m_ba81b85074cf"), state.name, value -> state.name = value);
        formBody.addView(UI.field(requireContext(), I18n.text("m_5df7bfb4b708"), name, UI.MD));
        EditText url = input("https://example.com", state.url, value -> state.url = value);
        url.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_URI);
        formBody.addView(UI.field(requireContext(), I18n.text("m_02739d1bae8b"), url, UI.MD));
        if (!"add".equals(state.form)) return;
        LinearLayout method = UI.segmented(requireContext(), new String[]{"GET", "POST"}, state.method, value -> state.method = value);
        LinearLayout methodField = UI.field(requireContext(), I18n.text("m_cf206666a6a5"), method, UI.MD);
        methodField.setVisibility(state.type == 0 ? View.VISIBLE : View.GONE);
        LinearLayout type = UI.segmented(requireContext(), new String[]{"HTTP", "TCP", "ICMP"}, state.type, value -> {
            state.type = value;
            methodField.setVisibility(value == 0 ? View.VISIBLE : View.GONE);
            url.setHint(new String[]{"https://example.com", "example.com:443", I18n.text("m_938f4bae209e")}[value]);
        });
        url.setHint(new String[]{"https://example.com", "example.com:443", I18n.text("m_938f4bae209e")}[state.type]);
        formBody.addView(UI.field(requireContext(), I18n.text("m_86b4208b5178"), type, UI.MD));
        formBody.addView(methodField);
    }

    private EditText input(String hint, String value, Consumer<String> changed) {
        EditText input = UI.input(requireContext(), hint);
        input.setSingleLine(true);
        input.setSaveEnabled(false);
        input.setText(value);
        input.addTextChangedListener(new TextWatcher() {
            @Override public void beforeTextChanged(CharSequence text, int start, int count, int after) {}
            @Override public void onTextChanged(CharSequence text, int start, int before, int count) {}
            @Override public void afterTextChanged(Editable text) { changed.accept(text.toString()); }
        });
        return input;
    }

    private void enableTree(View view, boolean enabled) {
        view.setEnabled(enabled);
        if (view instanceof ViewGroup) {
            ViewGroup group = (ViewGroup) view;
            for (int index = 0; index < group.getChildCount(); index++) enableTree(group.getChildAt(index), enabled);
        }
    }

    private void askDelete(String name) {
        Modal.of(requireContext(), I18n.text("m_b2fc749edfa3"))
                .message(I18n.text("m_fbc9178f86b7") + name + I18n.text("m_1b54e6f260d4"))
                .cancel(I18n.text("m_2cd0f3be8738")).confirm(I18n.text("m_a3ea3c17b401"), true, state::deleteMonitor).show();
    }

    private void copyLink(String link) {
        ClipboardManager clipboard = (ClipboardManager) requireContext().getSystemService(Context.CLIPBOARD_SERVICE);
        if (clipboard != null) {
            clipboard.setPrimaryClip(ClipData.newPlainText(I18n.text("m_6e3331e264ce"), link));
            if (android.os.Build.VERSION.SDK_INT < 33) {
                android.widget.Toast.makeText(requireContext(), I18n.text("m_6ce7dfa5ecf5"), android.widget.Toast.LENGTH_SHORT).show();
            }
        }
    }
}
