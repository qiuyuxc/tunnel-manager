package com.tunnelmanager.app;

import android.graphics.Typeface;
import android.view.Gravity;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.EditText;
import android.widget.ImageView;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout;

import org.json.JSONArray;
import org.json.JSONObject;

import java.util.Locale;

public class MonitorsFragment extends PageFragment {
    private SwipeRefreshLayout refresh;
    private ScrollView scroll;
    private LinearLayout body;
    private JSONArray monitors;
    private JSONObject overview;
    private String error = "";
    private String summaryError = "";
    private boolean loading;
    private boolean creating;
    private int generation;

    @Override String route() {
        return "/monitors";
    }

    @Override protected View build(@NonNull LayoutInflater inflater, @Nullable ViewGroup container) {
        refresh = new SwipeRefreshLayout(requireContext());
        refresh.setOnRefreshListener(this::load);
        scroll = new ScrollView(requireContext());
        body = UI.column(requireContext());
        UI.pagePadding(body);
        scroll.addView(body);
        refresh.addView(scroll);
        render();
        return refresh;
    }

    @Override public void onResume() {
        super.onResume();
        load();
    }

    @Override public void onDestroyView() {
        generation++;
        loading = false;
        refresh = null;
        scroll = null;
        body = null;
        super.onDestroyView();
    }

    private void load() {
        if (!alive() || refresh == null || loading) return;
        int request = ++generation;
        loading = true;
        error = "";
        render();
        Api.async(() -> {
            JSONObject result = new JSONObject();
            result.put("monitors", Api.getArray("/api/monitors"));
            try {
                result.put("overview", Api.get("/api/monitors/overview"));
            } catch (Exception failure) {
                result.put("summary_error", "七天统计暂不可用，下拉可重试。项目状态仍按已获取的数据展示。");
            }
            return result;
        }, result -> {
            if (request != generation || body == null || !alive()) return;
            loading = false;
            monitors = result.optJSONArray("monitors");
            overview = result.optJSONObject("overview");
            summaryError = result.optString("summary_error");
            render();
        }, failure -> {
            if (request != generation || body == null || !alive()) return;
            loading = false;
            error = failure.getMessage() == null ? "加载失败，下拉重试" : failure.getMessage();
            render();
        });
    }

    private void render() {
        if (body == null || !alive()) return;
        int offset = scroll.getScrollY();
        refresh.setRefreshing(loading);
        body.removeAllViews();
        LinearLayout head = UI.row(requireContext());
        head.addView(UI.pageTitle(requireContext(), "每一次心跳"),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        ImageView create = UI.iconButton(requireContext(), R.drawable.ic_nav_plus, Theme.p().success, view -> askCreate());
        create.setContentDescription("创建监控项目");
        create.setEnabled(!creating);
        head.addView(create);
        body.addView(head);
        UI.addRow(body, UI.muted(requireContext(), "所有服务的状态，一眼看清。"), UI.XS);
        if (!error.isEmpty()) {
            UI.addRow(body, UI.banner(requireContext(), error + (monitors == null ? "" : "\n当前显示上次获取的数据。"), true), UI.MD);
            TextView retry = UI.button(requireContext(), "重新加载", UI.BTN_SECONDARY);
            retry.setOnClickListener(view -> load());
            UI.addRow(body, retry, UI.SM);
        }
        if (monitors == null) {
            if (error.isEmpty()) UI.addRow(body, UI.muted(requireContext(), "正在读取监控项目…"), UI.LG);
            return;
        }
        UI.addRow(body, healthCard(), UI.LG);
        if (monitors.length() == 0) {
            UI.addRow(body, emptyCard(), UI.MD);
        } else {
            UI.addRow(body, MonitorViews.heading(requireContext(), "监控项目", UI.muted(requireContext(), monitors.length() + " 个")), UI.LG);
            for (int index = 0; index < monitors.length(); index++) {
                JSONObject monitor = monitors.optJSONObject(index);
                if (monitor != null) UI.addRow(body, monitorCard(monitor), UI.MD);
            }
        }
        UI.restoreScroll(scroll, offset);
    }

    private View healthCard() {
        MonitorHistory.Counts counts = new MonitorHistory.Counts();
        for (int index = 0; index < monitors.length(); index++) {
            JSONObject monitor = monitors.optJSONObject(index);
            JSONArray targets = monitor == null ? null : monitor.optJSONArray("targets");
            if (targets == null) continue;
            for (int targetIndex = 0; targetIndex < targets.length(); targetIndex++) {
                JSONObject target = targets.optJSONObject(targetIndex);
                if (target != null) counts.add(target.optString("state"));
            }
        }
        String headline = counts.total == 0 ? "等待添加服务" : counts.down > 0 ? "有服务需要关注"
                : counts.warn > 0 ? "部分服务响应降级" : counts.unknown > 0 ? "部分服务状态待确认" : "全部服务运行正常";
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.text(requireContext(), headline, 14, MonitorViews.color(counts.state()), Typeface.BOLD));
        long checks = 0;
        JSONArray buckets = overview == null ? null : overview.optJSONArray("buckets");
        if (buckets != null) {
            for (int index = 0; index < buckets.length(); index++) {
                JSONObject bucket = buckets.optJSONObject(index);
                if (bucket != null) checks += bucket.optLong("total");
            }
        }
        double uptime = overview == null ? Double.NaN : overview.optDouble("uptime", Double.NaN);
        boolean available = checks > 0 && Double.isFinite(uptime) && uptime >= 0 && uptime <= 100;
        TextView value = UI.text(requireContext(), available ? String.format(Locale.US, "%.2f%%", uptime)
                : overview == null ? "统计暂不可用" : "暂无检测样本", available ? 38 : 24, Theme.p().ink, Typeface.BOLD);
        value.setFontFeatureSettings("tnum");
        UI.addRow(card, value, UI.SM);
        UI.addRow(card, UI.muted(requireContext(), "过去 7 天可用率 · " + monitors.length() + " 个项目 · " + counts.total + " 个服务"), UI.XS);
        if (counts.total > 0) UI.addRow(card, UI.muted(requireContext(), MonitorViews.summary(counts)), UI.SM);
        if (!summaryError.isEmpty()) UI.addRow(card, UI.muted(requireContext(), summaryError), UI.SM);
        return card;
    }

    private View emptyCard() {
        LinearLayout card = UI.card(requireContext());
        card.setGravity(Gravity.CENTER_HORIZONTAL);
        card.addView(UI.strong(requireContext(), "还没有监控项目"));
        TextView copy = UI.muted(requireContext(), "创建项目，再添加服务地址。系统会按固定间隔持续检测。 ");
        copy.setGravity(Gravity.CENTER);
        UI.addRow(card, copy, UI.SM);
        TextView create = UI.button(requireContext(), creating ? "正在创建…" : "创建第一个监控", UI.BTN_PRIMARY);
        create.setEnabled(!creating);
        create.setOnClickListener(view -> askCreate());
        UI.addRow(card, create, UI.MD);
        return card;
    }

    private View monitorCard(JSONObject monitor) {
        String id = monitor.optString("id");
        JSONArray targets = monitor.optJSONArray("targets");
        MonitorHistory.Counts counts = MonitorViews.counts(targets);
        LinearLayout card = UI.card(requireContext());
        TextView status = UI.statusPill(requireContext(), counts.state());
        if (counts.total == 0) status.setText("未配置");
        card.addView(MonitorViews.heading(requireContext(), monitor.optString("name", "监控项目"), status));
        UI.addRow(card, UI.muted(requireContext(), counts.total + " 个服务 · 每 " + monitor.optInt("interval_sec")
                + " 秒检查 · " + (monitor.optBoolean("publish_enabled") ? "已公开" : "未公开")), UI.SM);
        if (counts.total > 0) UI.addRow(card, UI.muted(requireContext(), MonitorViews.summary(counts)), UI.XS);
        JSONObject target = MonitorViews.selectedTarget(targets, "");
        if (target != null) {
            MonitorHistory history = MonitorViews.history(target);
            UI.addRow(card, UI.muted(requireContext(), "历史预览 · " + target.optString("name", "服务")), UI.MD);
            UI.addRow(card, new MonitorHistoryView(requireContext(), history, false, 0), UI.XS);
            UI.addRow(card, UI.muted(requireContext(), history.samples.size() + " 个抽样点 · 最近样本 "
                    + MonitorViews.time(history.lastTime())), UI.XS);
            UI.addRow(card, UI.text(requireContext(), "当前响应 " + MonitorViews.latency(target) + " · "
                    + MonitorViews.stateLabel(target.optString("state")), 12, MonitorViews.color(target.optString("state")), Typeface.NORMAL), UI.XS);
        } else {
            UI.addRow(card, UI.muted(requireContext(), "添加服务后开始记录健康历史。"), UI.MD);
        }
        TextView open = UI.text(requireContext(), "查看项目 ›", 13, Theme.p().success, Typeface.BOLD);
        UI.addRow(card, open, UI.MD);
        card.setBackground(UI.pressable(UI.roundedStroke(Theme.p().canvasRaised, UI.RADIUS_LG, Theme.p().hairline, 1), Theme.p().btnGhostHover));
        card.setFocusable(true);
        card.setOnClickListener(view -> openRoute("/monitors/" + id));
        card.setOnLongClickListener(view -> {
            askDelete(id, monitor.optString("name"));
            return true;
        });
        return card;
    }

    private void askCreate() {
        if (creating) return;
        EditText name = UI.input(requireContext(), "例如：生产环境服务");
        name.setSingleLine(true);
        name.setFilters(new android.text.InputFilter[]{new android.text.InputFilter.LengthFilter(60)});
        Modal.of(requireContext(), "创建监控项目")
                .message("给这组服务起个名字，建好后可以继续添加要探测的地址。")
                .content(name).cancel("取消")
                .confirm("创建", false, () -> create(name.getText().toString().trim())).show();
    }

    private void create(String name) {
        if (creating) return;
        if (name.isEmpty()) {
            toast("请填写项目名称");
            return;
        }
        creating = true;
        render();
        Api.async(() -> Api.post("/api/monitors", new JSONObject().put("name", name)), created -> {
            creating = false;
            if (body != null && alive()) openRoute("/monitors/" + created.optString("id"));
        }, failure -> {
            creating = false;
            if (body == null || !alive()) return;
            toast(failure.getMessage());
            render();
        });
    }

    private void askDelete(String id, String name) {
        Modal.of(requireContext(), "删除监控项目")
                .message("将删除「" + name + "」及其全部检测历史，公开链接同步失效。")
                .cancel("取消").confirm("确认删除", true, () -> Api.async(() -> Api.delete("/api/monitors/" + id),
                        result -> load(), failure -> toast(failure.getMessage()))).show();
    }

    private void toast(String message) {
        if (message != null && !message.isEmpty() && alive()) {
            android.widget.Toast.makeText(requireContext(), message, android.widget.Toast.LENGTH_LONG).show();
        }
    }
}
