package com.tunnelmanager.app;

import android.content.Context;
import android.graphics.Typeface;
import android.graphics.drawable.Drawable;
import android.text.TextUtils;
import android.view.Gravity;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.ImageView;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;
import java.text.SimpleDateFormat;
import java.util.Date;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout;

import org.json.JSONArray;
import org.json.JSONObject;

import java.util.ArrayList;
import java.util.Calendar;
import java.util.List;
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
    private ScrollView scroll;

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

        scroll = new ScrollView(requireContext());
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
        int scrollY = scroll == null ? 0 : scroll.getScrollY();
        body.removeAllViews();
        TextView date = UI.label(requireContext(), java.text.DateFormat.getDateInstance(java.text.DateFormat.FULL, I18n.locale()).format(new Date()));
        body.addView(date);
        UI.margin(date, 0, 0, 0, UI.SM);
        body.addView(UI.pageTitle(requireContext(), I18n.text("m_0db8ec03228d")));
        TextView subtitle = UI.muted(requireContext(), I18n.text("m_8ec0d0243772"));
        UI.margin(subtitle, 0, UI.XS, 0, UI.LG);
        body.addView(subtitle);

        if (error != null) {
            body.addView(UI.banner(requireContext(), error, true));
            return;
        }
        if (loading) {
            body.addView(UI.muted(requireContext(), I18n.text("m_4927a53bcc88")));
            return;
        }
        if (overview == null) return;
        body.addView(healthCard(overview));
        body.addView(UI.spacer(requireContext(), UI.LG));
        body.addView(stats(overview));
        body.addView(UI.spacer(requireContext(), UI.LG));
        if (Session.hasPerm("tunnels") || Session.hasPerm("domain_bind")) {
            body.addView(quickActions());
            body.addView(UI.spacer(requireContext(), UI.LG));
        }
        body.addView(chartCard(overview));
        body.addView(UI.spacer(requireContext(), UI.LG));
        body.addView(monitorList(monitors));
        UI.restoreScroll(scroll, scrollY);
    }

    private View healthCard(JSONObject overview) {
        Palette palette = Theme.p();
        int targets = overview.optInt("targets");
        int healthy = overview.optInt("ok");
        int down = overview.optInt("down");
        int warning = overview.optInt("warn");
        int unknown = Math.max(0, targets - healthy - down - warning);
        String heading = targets == 0 ? I18n.text("m_2155118bef9d") : down > 0 ? I18n.text("m_d82e21620f20")
                : warning > 0 ? I18n.text("m_52daa7b71afd") : unknown > 0 ? I18n.text("m_c463b68e64bb") : I18n.text("m_bc82d5ed1265");
        int accent = down > 0 ? palette.error : warning > 0 || unknown > 0 ? palette.warning : palette.success;
        LinearLayout card = UI.card(requireContext());
        TextView label = UI.label(requireContext(), I18n.text("m_7b9f9de918f0"));
        label.setTextColor(accent);
        card.addView(label);
        LinearLayout summary = UI.row(requireContext());
        LinearLayout words = UI.column(requireContext());
        words.addView(UI.text(requireContext(), heading, 22, palette.ink, Typeface.BOLD));
        TextView status = UI.muted(requireContext(), targets == 0 ? I18n.text("m_bccad81c9ac0")
                : healthy + I18n.text("m_8163fb8824fd") + down + I18n.text("m_ef5bc8130038")
                + (warning > 0 ? " · " + warning + I18n.text("m_c2235f6c0f6d") : "")
                + (unknown > 0 ? " · " + unknown + I18n.text("m_e348103e296a") : ""));
        UI.margin(status, 0, UI.SM, 0, 0);
        words.addView(status);
        summary.addView(words, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        HealthRingView ring = new HealthRingView(requireContext(), healthy, targets);
        LinearLayout.LayoutParams ringParams = new LinearLayout.LayoutParams(UI.dp(76), UI.dp(76));
        ringParams.leftMargin = UI.dp(UI.MD);
        summary.addView(ring, ringParams);
        UI.addRow(card, summary, UI.MD);
        double uptime = overview.has("uptime") ? overview.optDouble("uptime") : overview.optDouble("uptime_24h");
        TextView availability = UI.muted(requireContext(), I18n.text("m_acdb2a150295") + (targets == 0 ? I18n.text("m_497c85690c4c") : percent(uptime)));
        UI.addRow(card, availability, UI.LG);
        return card;
    }

    private View quickActions() {
        LinearLayout actions = UI.row(requireContext());
        if (Session.hasPerm("tunnels")) {
            TextView tunnels = UI.button(requireContext(), I18n.text("m_e920f47d0c05"), UI.BTN_SECONDARY);
            actions.addView(tunnels, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
            tunnels.setOnClickListener(view -> openRoute("/tunnels"));
        }
        if (Session.hasPerm("domain_bind")) {
            TextView domains = UI.button(requireContext(), I18n.text("m_dd799f925f94"), UI.BTN_SECONDARY);
            LinearLayout.LayoutParams params = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
            if (actions.getChildCount() > 0) params.leftMargin = UI.dp(UI.MD);
            actions.addView(domains, params);
            domains.setOnClickListener(view -> openRoute("/domain"));
        }
        return actions;
    }

    private View stats(JSONObject overview) {
        LinearLayout grid = UI.column(requireContext());
        grid.addView(statRow(
                stat(I18n.text("m_12eeb7ca4534"), String.valueOf(overview.optInt("targets")), null),
                stat(I18n.text("m_24e14c0ff18c"), overview.optInt("avg_latency_ms") + " ms", null)));
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
        TextView number = UI.text(requireContext(), value, 28, accent == null ? p.ink : accent, Typeface.NORMAL);
        UI.margin(number, 0, UI.SM, 0, 0);
        card.addView(number);
        return card;
    }

    private View chartCard(JSONObject overview) {
        JSONArray buckets = overview.optJSONArray("buckets");
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), I18n.text("m_b1136b80c98b")));
        TextView hint = UI.muted(requireContext(), I18n.text("m_373302d2286a"));
        UI.margin(hint, 0, UI.XS, 0, 0);
        card.addView(hint);
        if (buckets == null || buckets.length() == 0) {
            UI.margin(card.getChildAt(card.getChildCount() - 1), 0, UI.XS, 0, UI.LG);
            card.addView(UI.muted(requireContext(), I18n.text("m_47ecaccb61dc")));
            return card;
        }

	ChartView chart = new ChartView(requireContext());
	chart.setBuckets(buckets, overview.optInt("bucket_sec", 3600));
	// The bar only says the day went wrong; tapping it opens who and when.
	chart.setOnColumnTap(this::showBucketDetail);
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
        row.addView(legendItem(I18n.text("m_5fa34377ddca"), p.ink, false));
        row.addView(legendItem(I18n.text("m_a49ffd73bc85"), p.ink, true));
        row.addView(legendItem(I18n.text("m_9f22b41cf5d7"), p.warning, true));
        row.addView(legendItem(I18n.text("m_8d7c03019f1d"), p.error, true));
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
            card.addView(UI.cardTitle(requireContext(), I18n.text("m_ce0d1c8bffa4")));
            TextView empty = UI.muted(requireContext(), I18n.text("m_7fba8a0e8167"));
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
        header.addView(UI.text(requireContext(), monitor.optString("name", I18n.text("m_ce0d1c8bffa4")), 15, p.ink, Typeface.BOLD),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        header.addView(UI.mono(requireContext(), monitor.optInt("interval_sec") + "s", p.mute));
        card.addView(header);

        JSONArray targets = monitor.optJSONArray("targets");
        if (targets == null || targets.length() == 0) {
            TextView empty = UI.muted(requireContext(), I18n.text("m_c49ad918e1c5"));
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

    // ------------------------------------------------------- chart drill-down

    /**
     * The bar says a day went wrong; this says who and when. Built from the
     * per-target {@code issues} the overview returns with every bucket, so the
     * sheet and the bar cannot disagree about the day.
     */
    private void showBucketDetail(JSONObject bucket) {
	Context ctx = requireContext();
	Sheet.Builder sheet = Sheet.of(ctx, dayLabel(bucket.optLong("hour")) + I18n.text("m_70bcee5995c2"));
	sheet.content(bucketStats(bucket));

	JSONArray issues = bucket.optJSONArray("issues");
	if (issues == null || issues.length() == 0) {
		sheet.content(UI.muted(ctx, I18n.text("m_5435dc68f0ff")));
		sheet.show();
		return;
	}
	sheet.label(I18n.text("m_04656c447ef6") + issues.length() + I18n.text("m_bfccaf26c23e"));
	for (int i = 0; i < issues.length(); i++) {
		JSONObject issue = issues.optJSONObject(i);
		if (issue != null) sheet.content(issueCard(issue));
	}
	sheet.show();
    }

    /** The day's totals, on one wrapping line so five figures fit a phone. */
    private View bucketStats(JSONObject bucket) {
	List<String> bits = new ArrayList<>();
	bits.add(I18n.text("m_8533a9ba995e") + bucket.optInt("total") + I18n.text("m_f81526efce19"));
	if (bucket.optInt("warn") > 0) bits.add(I18n.text("m_aafb70b7aab2") + bucket.optInt("warn"));
	if (bucket.optInt("down") > 0) bits.add(I18n.text("m_edc10d4f3ecf") + bucket.optInt("down"));
	if (bucket.optLong("peak_ms") > 0) bits.add(I18n.text("m_cd4fd6677c07") + bucket.optLong("peak_ms") + "ms");
	double avg = bucket.optDouble("avg_ms", 0);
	if (avg > 0) bits.add(I18n.text("m_acf531be9aa9") + Math.round(avg) + "ms");
	TextView line = UI.muted(requireContext(), TextUtils.join(" · ", bits));
	UI.margin(line, UI.MD, UI.MD, UI.MD, 0);
	return line;
    }

    /** One target's day: what failed, and each stretch it was failing. */
    private View issueCard(JSONObject issue) {
	Palette p = Theme.p();
	LinearLayout card = UI.column(requireContext());
	card.setBackground(UI.roundedStroke(p.canvasRaised, UI.RADIUS_LG, p.hairline, 1));
	card.setPadding(UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD));

	LinearLayout head = UI.row(requireContext());
	String name = issue.optString("monitor_name", "") + " · " + issue.optString("target_name", "");
	head.addView(UI.strong(requireContext(), name),
		new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
	ImageView chevron = new ImageView(requireContext());
	chevron.setImageResource(R.drawable.ic_nav_chevron);
	UI.tint(chevron, p.mute);
	head.addView(chevron, new LinearLayout.LayoutParams(UI.dp(15), UI.dp(15)));
	card.addView(head);

	List<String> meta = new ArrayList<>();
	if (issue.optInt("down") > 0) meta.add(I18n.text("m_edc10d4f3ecf") + issue.optInt("down"));
	if (issue.optInt("warn") > 0) meta.add(I18n.text("m_aafb70b7aab2") + issue.optInt("warn"));
	if (issue.optLong("peak_ms") > 0) meta.add(I18n.text("m_cd4fd6677c07") + issue.optLong("peak_ms") + "ms");
	if (!meta.isEmpty()) {
		TextView line = UI.muted(requireContext(), TextUtils.join(" · ", meta));
		UI.margin(line, 0, 2, 0, 0);
		card.addView(line);
	}

	JSONArray incidents = issue.optJSONArray("incidents");
	int shown = incidents == null ? 0 : incidents.length();
	if (shown > 0) card.addView(UI.divider(requireContext()));
	for (int i = 0; i < shown; i++) {
		JSONObject inc = incidents.optJSONObject(i);
		if (inc != null) card.addView(incidentRow(inc));
	}
	int windows = issue.optInt("incident_count", shown);
	if (windows > shown) {
		TextView more = UI.muted(requireContext(), I18n.text("m_dff74742a6f0") + (windows - shown) + I18n.text("m_0ca688cba31e"));
		UI.margin(more, 0, UI.SM, 0, 0);
		card.addView(more);
	}

	// The whole card is the tap target: the phone sheet is narrow enough
	// that a chevron-only affordance would be a miss.
	Drawable resting = card.getBackground();
	card.setBackground(UI.pressable(resting, p.btnGhostHover));
	card.setClickable(true);
	card.setOnClickListener(v -> {
		Sheet.dismissVisible();
		openRoute("/monitors/" + issue.optString("monitor_id", ""));
	});
	return card;
    }

    /** One outage window: the clock range, the worst state, and the cause. */
    private View incidentRow(JSONObject inc) {
	Palette p = Theme.p();
	boolean down = "down".equals(inc.optString("state", ""));

	LinearLayout row = UI.row(requireContext());
	row.setGravity(Gravity.TOP);
	row.addView(UI.text(requireContext(), incidentRange(inc), 12, p.ink, Typeface.NORMAL));

	LinearLayout.LayoutParams stateLp = new LinearLayout.LayoutParams(
		ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);
	stateLp.leftMargin = UI.dp(UI.SM);
	row.addView(UI.text(requireContext(), down ? I18n.text("m_8d7c03019f1d") : I18n.text("m_9f22b41cf5d7"), 12,
		down ? p.error : p.warning, Typeface.BOLD), stateLp);

	String cause = incidentCause(inc);
	if (!cause.isEmpty()) {
		TextView line = UI.muted(requireContext(), cause);
		line.setMaxLines(2);
		line.setEllipsize(TextUtils.TruncateAt.END);
		LinearLayout.LayoutParams causeLp = new LinearLayout.LayoutParams(
			0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
		causeLp.leftMargin = UI.dp(UI.SM);
		row.addView(line, causeLp);
	}
	UI.margin(row, 0, UI.SM, 0, 0);
	return row;
    }

    /** "11:53–12:26", or a single clock time when the window is one sample. */
    private static String incidentRange(JSONObject inc) {
	String from = clock(inc.optLong("from"));
	String to = clock(inc.optLong("to"));
	return from.equals(to) ? from : from + "–" + to;
    }

    private static String clock(long seconds) {
	Calendar c = Calendar.getInstance();
	c.setTimeInMillis(seconds * 1000L);
	return String.format(Locale.US, "%02d:%02d", c.get(Calendar.HOUR_OF_DAY), c.get(Calendar.MINUTE));
    }

    /** The probe's own message beats the bare status code. */
    private static String incidentCause(JSONObject inc) {
	List<String> bits = new ArrayList<>();
	int count = inc.optInt("count");
	if (count > 1) bits.add(I18n.text("m_3c54ef9ee95e") + count + I18n.text("m_f81526efce19"));
	String error = inc.optString("error", "");
	if (!error.isEmpty()) bits.add(error);
	else if (inc.optInt("code") > 0) bits.add("HTTP " + inc.optInt("code"));
	return TextUtils.join(" · ", bits);
    }

    /** A date for the day-wide buckets the overview sends. */
    private static String dayLabel(long seconds) {
	Calendar c = Calendar.getInstance();
	c.setTimeInMillis(seconds * 1000L);
	return (c.get(Calendar.MONTH) + 1) + "/" + c.get(Calendar.DAY_OF_MONTH);
    }

    private static String percent(double value) {
        return String.format(Locale.US, "%.2f%%", value);
    }
}
