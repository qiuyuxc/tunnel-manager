package com.tunnelmanager.app;

import android.content.Context;
import android.graphics.Typeface;
import android.view.View;
import android.view.ViewGroup;
import android.widget.ImageView;
import android.widget.LinearLayout;
import android.widget.TextView;

import org.json.JSONArray;
import org.json.JSONObject;

import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Date;
import java.util.Locale;

final class MonitorViews {
    static MonitorHistory history(JSONObject target) {
        ArrayList<MonitorHistory.Sample> samples = new ArrayList<>();
        JSONArray bars = target == null ? null : target.optJSONArray("bars");
        if (bars != null) {
            for (int index = 0; index < bars.length(); index++) {
                JSONObject bar = bars.optJSONObject(index);
                if (bar == null) continue;
                double latency = bar.has("ms") && bar.isNull("ms") ? Double.NaN : bar.optDouble("ms", 0);
                samples.add(new MonitorHistory.Sample(bar.optLong("t"), bar.optString("s"), latency));
            }
        }
        return new MonitorHistory(samples);
    }

    static MonitorHistory.Counts counts(JSONArray targets) {
        MonitorHistory.Counts counts = new MonitorHistory.Counts();
        if (targets != null) {
            for (int index = 0; index < targets.length(); index++) {
                JSONObject target = targets.optJSONObject(index);
                if (target != null) counts.add(target.optString("state"));
            }
        }
        return counts;
    }

    static JSONObject selectedTarget(JSONArray targets, String selectedId) {
        JSONObject first = null;
        if (targets != null) {
            for (int index = 0; index < targets.length(); index++) {
                JSONObject target = targets.optJSONObject(index);
                if (target == null) continue;
                if (first == null) first = target;
                if (target.optString("id").equals(selectedId)) return target;
            }
        }
        return first;
    }

    static int color(String state) {
        switch (MonitorHistory.normalize(state)) {
            case "ok": return Theme.p().success;
            case "warn": return Theme.p().warning;
            case "down": return Theme.p().error;
            default: return Theme.p().mute;
        }
    }

    static String stateLabel(String state) {
        switch (MonitorHistory.normalize(state)) {
            case "ok": return "正常";
            case "warn": return "降级";
            case "down": return "异常";
            default: return "待检测 / 未知";
        }
    }

    static String summary(MonitorHistory.Counts counts) {
        return "正常 " + counts.ok + " · 降级 " + counts.warn + " · 异常 " + counts.down
                + (counts.unknown > 0 ? " · 未知 " + counts.unknown : "");
    }

    static String latency(JSONObject target) {
        if (target == null) return "暂无数据";
        String state = MonitorHistory.normalize(target.optString("state"));
        if (!"ok".equals(state) && !"warn".equals(state)) return "暂无响应";
        double latency = target.has("latency_ms") && target.isNull("latency_ms")
                ? Double.NaN : target.optDouble("latency_ms", 0);
        return Double.isFinite(latency) && latency >= 0 ? String.format(Locale.US, "%.0f ms", latency) : "暂无数据";
    }

    static String uptime(JSONObject target, MonitorHistory history) {
        double value = target.optDouble("uptime_24h", Double.NaN);
        if (!history.hasRecentSamples(System.currentTimeMillis(), 24L * 60 * 60 * 1000)
                || !Double.isFinite(value) || value < 0 || value > 100) return "暂无近期样本";
        return String.format(Locale.US, "%.2f%%", value);
    }

    static String time(long timestamp) {
        return timestamp <= 0 ? "暂无样本" : new SimpleDateFormat("MM-dd HH:mm", Locale.getDefault()).format(new Date(timestamp));
    }

    static String range(MonitorHistory history) {
        if (history.samples.isEmpty()) return "等待首次检测";
        if (history.samples.size() == 1) return "样本时间 " + time(history.firstTime());
        return time(history.firstTime()) + " 至 " + time(history.lastTime());
    }

    static LinearLayout heading(Context context, String title, View accessory) {
        LinearLayout row = UI.row(context);
        TextView label = UI.strong(context, title);
        row.addView(label, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        if (accessory != null) {
            UI.margin(accessory, UI.SM, 0, 0, 0);
            row.addView(accessory);
        }
        return row;
    }

    static View action(Context context, int iconRes, String title, String subtitle, Runnable action) {
        LinearLayout row = UI.row(context);
        row.setPadding(0, UI.dp(UI.SM), 0, UI.dp(UI.SM));
        row.setMinimumHeight(UI.dp(56));
        ImageView icon = new ImageView(context);
        icon.setImageResource(iconRes);
        UI.tint(icon, Theme.p().success);
        icon.setPadding(UI.dp(10), UI.dp(10), UI.dp(10), UI.dp(10));
        icon.setBackground(UI.rounded(Theme.p().canvasSoft2, UI.RADIUS_MD));
        row.addView(icon, new LinearLayout.LayoutParams(UI.dp(40), UI.dp(40)));
        LinearLayout text = UI.column(context);
        text.addView(UI.text(context, title, 14, Theme.p().ink, Typeface.NORMAL));
        UI.addRow(text, UI.muted(context, subtitle), UI.XS);
        LinearLayout.LayoutParams textLayout = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
        textLayout.setMargins(UI.dp(UI.SM), 0, UI.dp(UI.SM), 0);
        row.addView(text, textLayout);
        ImageView chevron = new ImageView(context);
        chevron.setImageResource(R.drawable.ic_nav_chevron);
        UI.tint(chevron, Theme.p().mute);
        row.addView(chevron, new LinearLayout.LayoutParams(UI.dp(16), UI.dp(16)));
        row.setBackground(UI.pressable(UI.rounded(Theme.p().canvasRaised, UI.RADIUS_MD), Theme.p().btnGhostHover));
        row.setFocusable(true);
        row.setOnClickListener(view -> action.run());
        return row;
    }
}
