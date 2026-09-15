package com.tunnelmanager.app;

import android.graphics.Typeface;
import android.text.TextUtils;
import android.view.Gravity;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout;

import org.json.JSONArray;
import org.json.JSONObject;

import java.util.Locale;

/**
 * 控制面板 — the console's landing page.
 *
 * Two requests behind one pull-to-refresh: /api/monitors/overview for the
 * counters and the twelve-hour chart, /api/monitors for the per-target states.
 * Both are the same calls the web dashboard makes, so the two agree by
 * construction rather than by being kept in sync.
 */
public class DashboardFragment extends PageFragment {

    private SwipeRefreshLayout refresh;
    private LinearLayout body;

    @Override
    String route() {
        return "/dashboard";
    }

    @Override
    protected View build(@NonNull LayoutInflater inflater, @Nullable ViewGroup container) {
        refresh = new SwipeRefreshLayout(requireContext());
        refresh.setColorSchemeColors(Theme.p().ink);
        refresh.setProgressBackgroundColorSchemeColor(Theme.p().canvasRaised);
        refresh.setOnRefreshListener(this::load);

        ScrollView scroll = new ScrollView(requireContext());
        body = UI.column(requireContext());
        UI.pagePadding(body);
        scroll.addView(body);
        refresh.addView(scroll);

        render(null, null, true);
        load();
        return refresh;
    }

    private void load() {
        Api.async(() -> {
            JSONObject overview = Api.get("/api/monitors/overview");
            JSONArray monitors = Api.getArray("/api/monitors");
            JSONObject pair = new JSONObject();
            pair.put("overview", overview);
            pair.put("monitors", monitors);
            return pair;
        }, result -> {
            refresh.setRefreshing(false);
            render(result.optJSONObject("overview"), result.optJSONArray("monitors"), false);
        }, failure -> {
            refresh.setRefreshing(false);
            render(null, null, false, failure.getMessage());
        });
    }

    // --------------------------------------------------------------- rendering

    private void render(@Nullable JSONObject overview, @Nullable JSONArray monitors, boolean loading) {
        render(overview, monitors, loading, null);
    }

    private void render(@Nullable JSONObject overview, @Nullable JSONArray monitors,
                        boolean loading, @Nullable String error) {
        if (body == null || !alive()) return;
        body.removeAllViews();

        body.addView(UI.pageTitle(requireContext(), "控制面板"));
        TextView subtitle = UI.muted(requireContext(), "监控目标状态、可用率与延迟概览");
        UI.margin(subtitle, 0, UI.XS, 0, UI.LG);
        body.addView(subtitle);

        if (error != null) {
            body.addView(UI.banner(requireContext(), error, true));
            return;
        }
        if (loading) {
            body.addView(UI.muted(requireContext(), "加载中…"));
            return;
        }
        if (overview == null) return;

        body.addView(stats(overview));
        body.addView(UI.spacer(requireContext(), UI.LG));
        body.addView(chartCard(overview));
        body.addView(UI.spacer(requireContext(), UI.LG));
        body.addView(monitorList(monitors));
    }

    private View stats(JSONObject overview) {
        // uptime is the current field; uptime_24h is what servers before the
        // seven-day window sent, and still send alongside it.
        double uptime = overview.has("uptime")
                ? overview.optDouble("uptime")
                : overview.optDouble("uptime_24h");
        LinearLayout grid = UI.column(requireContext());
        grid.addView(statRow(
                stat("监控目标", String.valueOf(overview.optInt("targets")), null),
                stat("7 天可用率", percent(uptime), null)));
        grid.addView(UI.spacer(requireContext(), UI.SM));
        grid.addView(statRow(
                stat("正常 / 异常",
                        overview.optInt("ok") + " / " + overview.optInt("down"),
                        overview.optInt("down") > 0 ? Theme.p().error : null),
                stat("平均延迟", overview.optInt("avg_latency_ms") + " ms", null)));
        return grid;
    }

    private LinearLayout statRow(View left, View right) {
        LinearLayout row = UI.row(requireContext());
        LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
        left.setLayoutParams(lp);
        LinearLayout.LayoutParams rp = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
        rp.leftMargin = UI.dp(UI.SM);
        right.setLayoutParams(rp);
        row.addView(left);
        row.addView(right);
        return row;
    }

    private View stat(String label, String value, @Nullable Integer accent) {
        Palette p = Theme.p();
        LinearLayout card = UI.card(requireContext());
        TextView caption = UI.label(requireContext(), label);
        card.addView(caption);
        TextView number = UI.text(requireContext(), value, 24, accent == null ? p.ink : accent, Typeface.BOLD);
        UI.margin(number, 0, UI.SM, 0, 0);
        card.addView(number);
        return card;
    }

    private View chartCard(JSONObject overview) {
        JSONArray buckets = overview.optJSONArray("buckets");
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), "近 7 天"));
        TextView hint = UI.muted(requireContext(), "每根柱子的高度是当天的峰值延迟，颜色代表当天最差状态");
        UI.margin(hint, 0, UI.XS, 0, 0);
        card.addView(hint);
        if (buckets == null || buckets.length() == 0) {
            UI.margin(card.getChildAt(card.getChildCount() - 1), 0, UI.XS, 0, UI.LG);
            card.addView(UI.muted(requireContext(), "暂无心跳数据"));
            return card;
        }

        ChartView chart = new ChartView(requireContext());
        chart.setBuckets(buckets, overview.optInt("bucket_sec", 3600));
        card.addView(chart);
        card.addView(legend());
        return card;
    }

    /**
     * What the two bar layers mean. The web puts this under the chart; four
     * short labels fit the phone width where the web's long wording would not.
     */
    private View legend() {
        Palette p = Theme.p();
        LinearLayout row = UI.row(requireContext());
        row.setGravity(Gravity.END);
        UI.margin(row, 0, 10, 0, 0);
        row.addView(legendItem("峰值", p.ink, false));
        row.addView(legendItem("平均", p.ink, true));
        row.addView(legendItem("降级", p.warning, true));
        row.addView(legendItem("不可达", p.error, true));
        return row;
    }

    private View legendItem(String label, int color, boolean solid) {
        Palette p = Theme.p();
        LinearLayout item = UI.row(requireContext());
        View swatch = new View(requireContext());
        swatch.setBackground(solid
                ? UI.rounded(color, 3)
                : UI.roundedStroke(withAlpha(color, 0.12f), 3, withAlpha(color, 0.20f), 1));
        LinearLayout.LayoutParams swatchLp = new LinearLayout.LayoutParams(UI.dp(10), UI.dp(10));
        swatchLp.rightMargin = UI.dp(5);
        item.addView(swatch, swatchLp);

        item.addView(UI.text(requireContext(), label, 11, p.mute, Typeface.NORMAL));
        LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);
        lp.leftMargin = UI.dp(UI.MD);
        item.setLayoutParams(lp);
        return item;
    }

    private static int withAlpha(int color, float value) {
        return (color & 0x00FFFFFF) | (Math.round(255 * value) << 24);
    }

    private View monitorList(@Nullable JSONArray monitors) {
        LinearLayout box = UI.column(requireContext());
        if (monitors == null || monitors.length() == 0) {
            LinearLayout card = UI.card(requireContext());
            card.addView(UI.cardTitle(requireContext(), "监控项目"));
            TextView empty = UI.muted(requireContext(), "还没有监控项目，去「服务监控」新建一个");
            UI.margin(empty, 0, UI.SM, 0, 0);
            card.addView(empty);
            box.addView(card);
            return box;
        }
        for (int i = 0; i < monitors.length(); i++) {
            JSONObject monitor = monitors.optJSONObject(i);
            if (i > 0) box.addView(UI.spacer(requireContext(), UI.SM));
            box.addView(monitorCard(monitor));
        }
        return box;
    }

    private View monitorCard(JSONObject monitor) {
        Palette p = Theme.p();
        LinearLayout card = UI.card(requireContext());

        LinearLayout header = UI.row(requireContext());
        header.addView(UI.text(requireContext(), monitor.optString("name", "监控项目"), 15, p.ink, Typeface.BOLD));
        View spacer = new View(requireContext());
        header.addView(spacer, new LinearLayout.LayoutParams(0, 1, 1f));
        header.addView(UI.mono(requireContext(), monitor.optInt("interval_sec") + "s", p.mute));
        card.addView(header);

        JSONArray targets = monitor.optJSONArray("targets");
        if (targets == null || targets.length() == 0) {
            TextView empty = UI.muted(requireContext(), "未配置探测目标");
            UI.margin(empty, 0, UI.SM, 0, 0);
            card.addView(empty);
            return card;
        }
        for (int i = 0; i < targets.length(); i++) {
            if (i > 0) card.addView(UI.divider(requireContext()));
            card.addView(targetRow(targets.optJSONObject(i)));
        }
        return card;
    }

    /**
     * One probe target. The status pill sits on the first line and the latency
     * hangs off the right, so a long socket error wraps under the name instead
     * of dragging both of them around the card.
     */
    private View targetRow(JSONObject target) {
        Palette p = Theme.p();
        LinearLayout row = UI.row(requireContext());
        row.setGravity(Gravity.TOP);

        LinearLayout.LayoutParams pillLp = new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);
        pillLp.topMargin = UI.dp(1);
        row.addView(UI.statusPill(requireContext(), target.optString("state", "")), pillLp);

        LinearLayout names = UI.column(requireContext());
        LinearLayout.LayoutParams namesLp = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
        namesLp.leftMargin = UI.dp(UI.SM);
        namesLp.rightMargin = UI.dp(UI.SM);
        row.addView(names, namesLp);

        LinearLayout title = UI.row(requireContext());
        title.addView(UI.text(requireContext(), target.optString("name", ""), 14, p.ink, Typeface.NORMAL),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        long latency = target.optLong("latency_ms");
        if (latency > 0) {
            title.addView(UI.mono(requireContext(), latency + " ms", p.mute));
        }
        names.addView(title);

        String detail = target.optString("error", "");
        if (detail.isEmpty() && target.optInt("http_code") > 0) {
            detail = "HTTP " + target.optInt("http_code");
        }
        if (!detail.isEmpty()) {
            TextView line = UI.muted(requireContext(), detail);
            line.setMaxLines(2);
            line.setEllipsize(TextUtils.TruncateAt.END);
            UI.margin(line, 0, 2, 0, 0);
            names.addView(line);
        }
        return row;
    }

    private static String percent(double value) {
        return String.format(Locale.US, "%.2f%%", value);
    }
}
