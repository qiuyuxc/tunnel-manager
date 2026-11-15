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
            case "ok": return I18n.text("m_296de0e31f8c");
            case "warn": return I18n.text("m_9f22b41cf5d7");
            case "down": return I18n.text("m_428fb8bfeecf");
            default: return I18n.text("m_c7208bddadac");
        }
    }

    static String summary(MonitorHistory.Counts counts) {
        return I18n.text("m_dcc188d23666") + counts.ok + I18n.text("m_c4fdff8830b6") + counts.warn + I18n.text("m_2d8d945e2794") + counts.down
                + (counts.unknown > 0 ? I18n.text("m_b09ed38f2d2f") + counts.unknown : "");
    }

    static String latency(JSONObject target) {
        if (target == null) return I18n.text("m_497c85690c4c");
        String state = MonitorHistory.normalize(target.optString("state"));
        if (!"ok".equals(state) && !"warn".equals(state)) return I18n.text("m_2c5d6c17493f");
        double latency = target.has("latency_ms") && target.isNull("latency_ms")
                ? Double.NaN : target.optDouble("latency_ms", 0);
        return Double.isFinite(latency) && latency >= 0 ? String.format(Locale.US, "%.0f ms", latency) : I18n.text("m_497c85690c4c");
    }

    static String uptime(JSONObject target, MonitorHistory history) {
        double value = target.optDouble("uptime_24h", Double.NaN);
        if (!history.hasRecentSamples(System.currentTimeMillis(), 24L * 60 * 60 * 1000)
                || !Double.isFinite(value) || value < 0 || value > 100) return I18n.text("m_30bb8dddca7c");
        return String.format(Locale.US, "%.2f%%", value);
    }

    static String time(long timestamp) {
        return timestamp <= 0 ? I18n.text("m_853936696310") : new SimpleDateFormat("MM-dd HH:mm", I18n.locale()).format(new Date(timestamp));
    }

    static String range(MonitorHistory history) {
        if (history.samples.isEmpty()) return I18n.text("m_5a868d538005");
        if (history.samples.size() == 1) return I18n.text("m_2b170a897cce") + time(history.firstTime());
        return time(history.firstTime()) + I18n.text("m_8727f227e2c6") + time(history.lastTime());
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
