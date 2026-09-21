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
                state.formError = "上次提交结果未确认，请先关闭面板并刷新项目核对，避免重复操作。";
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
        body.addView(UI.pageTitle(requireContext(), monitor == null ? "监控项目" : monitor.optString("name", "监控项目")));
        if (!state.error.isEmpty()) {
            UI.addRow(body, UI.banner(requireContext(), state.error, true), UI.MD);
            UI.addRow(body, button("重新加载", UI.BTN_SECONDARY, state::load), UI.SM);
        }
        if (monitor == null) {
            if (state.loading) UI.addRow(body, UI.muted(requireContext(), "正在读取服务状态…"), UI.MD);
            renderForm();
            return;
        }
        JSONArray targets = monitor.optJSONArray("targets");
        MonitorHistory.Counts counts = MonitorViews.counts(targets);
        UI.addRow(body, UI.muted(requireContext(), "监控项目 · " + counts.total + " 个服务 · "
                + (monitor.optBoolean("publish_enabled") ? "公开页已开启" : "仅自己可见")), UI.XS);
        UI.addRow(body, trendCard(monitor), UI.LG);
        UI.addRow(body, button(state.busy && "立即检测".equals(state.operation) ? "正在检测…" : "立即检测",
                UI.BTN_PRIMARY, state::checkNow), UI.MD);
        UI.addRow(body, MonitorViews.heading(requireContext(), "服务", UI.muted(requireContext(), counts.total + " 个")), UI.LG);
        UI.addRow(body, targetsCard(targets), UI.SM);
        UI.addRow(body, button("添加服务", UI.BTN_SECONDARY, () -> state.openForm("add", null)), UI.SM);
        UI.addRow(body, UI.strong(requireContext(), "检测与分享"), UI.LG);
        UI.addRow(body, settingsCard(monitor), UI.SM);
        UI.addRow(body, UI.strong(requireContext(), "危险操作"), UI.LG);
        UI.addRow(body, UI.muted(requireContext(), "删除后，全部检测历史和公开链接将一并失效。"), UI.SM);
        UI.addRow(body, button("删除监控项目", UI.BTN_DANGER, () -> askDelete(monitor.optString("name"))), UI.SM);
        UI.restoreScroll(scroll, offset);
        renderForm();
    }

    private View trendCard(JSONObject monitor) {
        LinearLayout card = UI.card(requireContext());
        JSONArray targets = monitor.optJSONArray("targets");
        JSONObject target = MonitorViews.selectedTarget(targets, state.selectedId);
        card.addView(MonitorViews.heading(requireContext(), "响应时间", UI.tag(requireContext(), "抽样历史", false)));
        if (target == null) {
            UI.addRow(card, UI.muted(requireContext(), "添加服务后，这里会展示真实检测趋势。"), UI.MD);
            return card;
        }
        state.selectedId = target.optString("id");
        if (MonitorViews.counts(targets).total > 1) {
            UI.addRow(card, button(target.optString("name", "服务") + " · 切换服务", UI.BTN_GHOST,
                    () -> chooseTarget(targets)), UI.SM);
        } else {
            UI.addRow(card, UI.muted(requireContext(), target.optString("name", "服务")), UI.SM);
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
        UI.addRow(card, UI.muted(requireContext(), history.samples.size() + " 个抽样点 · 稀疏、异常和缺测处不连线"), UI.XS);
        UI.addRow(card, UI.muted(requireContext(), "24 小时可用率  " + MonitorViews.uptime(target, history)), UI.SM);
        return card;
    }

    private View targetsCard(JSONArray targets) {
        LinearLayout card = UI.card(requireContext());
        if (MonitorViews.counts(targets).total == 0) {
            card.addView(UI.strong(requireContext(), "还没有服务"));
            UI.addRow(card, UI.muted(requireContext(), "支持 HTTP、TCP 和 ICMP，添加地址后开始检测。"), UI.SM);
            return card;
        }
        int rendered = 0;
        for (int index = 0; index < targets.length(); index++) {
            JSONObject target = targets.optJSONObject(index);
            if (target == null) continue;
            if (rendered++ > 0) card.addView(UI.divider(requireContext()));
            View action = MonitorViews.action(requireContext(), R.drawable.ic_nav_monitor,
                    target.optString("name", "服务"), target.optString("url"), () -> state.openForm("edit", target));
            action.setEnabled(!state.busy);
            card.addView(action);
            card.addView(MonitorViews.heading(requireContext(), MonitorViews.latency(target),
                    UI.statusPill(requireContext(), target.optString("state"))));
            UI.addRow(card, UI.muted(requireContext(), "24 小时可用率 "
                    + MonitorViews.uptime(target, MonitorViews.history(target))), UI.XS);
            if (!target.optString("error").isEmpty()) {
                UI.addRow(card, UI.text(requireContext(), target.optString("error"), 12, Theme.p().error, Typeface.NORMAL), UI.SM);
            }
        }
        return card;
    }

    private View settingsCard(JSONObject monitor) {
        LinearLayout card = UI.card(requireContext());
        View interval = MonitorViews.action(requireContext(), R.drawable.ic_nav_settings, "检查间隔",
                "每 " + monitor.optInt("interval_sec") + " 秒检查", () -> state.openForm("interval", null));
        interval.setEnabled(!state.busy);
        card.addView(interval);
        card.addView(UI.divider(requireContext()));
        View publish = MonitorViews.action(requireContext(), R.drawable.ic_nav_monitor, "公开状态页",
                monitor.optBoolean("publish_enabled") ? "只读链接 · 已开启" : "尚未公开 · 仅自己可见",
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
        picker = Sheet.of(requireContext(), "选择趋势服务");
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
        String label = "add".equals(state.form) ? "添加服务" : "保存修改";
        if ("public".equals(state.form)) label = state.monitor.optBoolean("publish_enabled") ? "关闭公开状态页" : "开启公开状态页";
        submit.setText(state.busy ? state.operation + "中…" : label);
        formError.setText(state.formError);
        formError.setVisibility(state.formError.isEmpty() ? View.GONE : View.VISIBLE);
    }

    private void buildForm() {
        displayedForm = state.form;
        formBody = UI.column(requireContext());
        String title;
        if ("public".equals(state.form)) {
            title = "公开状态页";
            publicContent();
        } else if ("interval".equals(state.form)) {
            title = "检测间隔";
            formBody.addView(UI.muted(requireContext(), "按固定间隔探测项目中的所有服务，最短为 30 秒。"));
            EditText interval = input("60", state.interval, value -> state.interval = value);
            interval.setInputType(InputType.TYPE_CLASS_NUMBER);
            formBody.addView(UI.field(requireContext(), "检查间隔（秒）", interval, UI.MD));
        } else {
            title = "add".equals(state.form) ? "添加服务" : "编辑服务";
            targetContent();
        }
        formError = UI.banner(requireContext(), "", true);
        formError.setAccessibilityLiveRegion(View.ACCESSIBILITY_LIVE_REGION_POLITE);
        UI.addRow(formBody, formError, UI.MD);
        if ("edit".equals(state.form)) {
            UI.addRow(formBody, button("移除这个服务", UI.BTN_DANGER, () -> Modal.of(requireContext(), "移除服务")
                    .message("将移除「" + state.name + "」及其检测历史。")
                    .cancel("取消").confirm("确认移除", true, state::removeTarget).show()), UI.LG);
        }
        submit = button("保存修改", UI.BTN_PRIMARY, () -> {
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
        formBody.addView(UI.muted(requireContext(), "开启后，任何持有链接的人都可查看服务名称和运行状态。"));
        JSONObject monitor = state.monitor;
        boolean published = monitor.optBoolean("publish_enabled");
        UI.addRow(formBody, UI.strong(requireContext(), published ? "当前已开启" : "当前未公开"), UI.MD);
        String token = monitor.optString("public_token");
        if (published && !token.isEmpty()) {
            String link = Session.server() + "/status/" + token;
            TextView url = UI.mono(requireContext(), link, Theme.p().link);
            url.setTextIsSelectable(true);
            UI.addRow(formBody, url, UI.MD);
            UI.addRow(formBody, button("复制只读链接", UI.BTN_SECONDARY, () -> copyLink(link)), UI.SM);
        } else if (published) {
            UI.addRow(formBody, UI.muted(requireContext(), "尚未取得公开链接，请刷新后重试。"), UI.SM);
        }
    }

    private void targetContent() {
        formBody.addView(UI.muted(requireContext(), "add".equals(state.form)
                ? "添加一个需要持续关注的服务。检测由服务器执行。" : "修改名称或地址，探测方式保持不变。"));
        EditText name = input("例如：Web 控制台", state.name, value -> state.name = value);
        formBody.addView(UI.field(requireContext(), "服务名称", name, UI.MD));
        EditText url = input("https://example.com", state.url, value -> state.url = value);
        url.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_URI);
        formBody.addView(UI.field(requireContext(), "服务地址", url, UI.MD));
        if (!"add".equals(state.form)) return;
        LinearLayout method = UI.segmented(requireContext(), new String[]{"GET", "POST"}, state.method, value -> state.method = value);
        LinearLayout methodField = UI.field(requireContext(), "请求方法", method, UI.MD);
        methodField.setVisibility(state.type == 0 ? View.VISIBLE : View.GONE);
        LinearLayout type = UI.segmented(requireContext(), new String[]{"HTTP", "TCP", "ICMP"}, state.type, value -> {
            state.type = value;
            methodField.setVisibility(value == 0 ? View.VISIBLE : View.GONE);
            url.setHint(new String[]{"https://example.com", "example.com:443", "example.com 或 IP 地址"}[value]);
        });
        url.setHint(new String[]{"https://example.com", "example.com:443", "example.com 或 IP 地址"}[state.type]);
        formBody.addView(UI.field(requireContext(), "探测方式", type, UI.MD));
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
        Modal.of(requireContext(), "删除监控项目")
                .message("将删除「" + name + "」及其全部检测历史，公开链接同步失效。")
                .cancel("取消").confirm("确认删除", true, state::deleteMonitor).show();
    }

    private void copyLink(String link) {
        ClipboardManager clipboard = (ClipboardManager) requireContext().getSystemService(Context.CLIPBOARD_SERVICE);
        if (clipboard != null) {
            clipboard.setPrimaryClip(ClipData.newPlainText("公开状态页", link));
            if (android.os.Build.VERSION.SDK_INT < 33) {
                android.widget.Toast.makeText(requireContext(), "链接已复制", android.widget.Toast.LENGTH_SHORT).show();
            }
        }
    }
}
