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
                result.put("summary_error", I18n.text("m_ec17571f0591"));
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
            error = failure.getMessage() == null ? I18n.text("m_abe7d7c55e82") : failure.getMessage();
            render();
        });
    }

    private void render() {
        if (body == null || !alive()) return;
        int offset = scroll.getScrollY();
        refresh.setRefreshing(loading);
        body.removeAllViews();
        LinearLayout head = UI.row(requireContext());
        head.addView(UI.pageTitle(requireContext(), I18n.text("m_6496450c34cc")),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        ImageView create = UI.iconButton(requireContext(), R.drawable.ic_nav_plus, Theme.p().success, view -> askCreate());
        create.setContentDescription(I18n.text("m_f9313095ee64"));
        create.setEnabled(!creating);
        head.addView(create);
        body.addView(head);
        UI.addRow(body, UI.muted(requireContext(), I18n.text("m_9bc73d27682d")), UI.XS);
        if (!error.isEmpty()) {
            UI.addRow(body, UI.banner(requireContext(), error + (monitors == null ? "" : I18n.text("m_d695dc5b7076")), true), UI.MD);
            TextView retry = UI.button(requireContext(), I18n.text("m_7bdd5ce1e298"), UI.BTN_SECONDARY);
            retry.setOnClickListener(view -> load());
            UI.addRow(body, retry, UI.SM);
        }
        if (monitors == null) {
            if (error.isEmpty()) UI.addRow(body, UI.muted(requireContext(), I18n.text("m_5dec25178795")), UI.LG);
            return;
        }
        UI.addRow(body, healthCard(), UI.LG);
        if (monitors.length() == 0) {
            UI.addRow(body, emptyCard(), UI.MD);
        } else {
            UI.addRow(body, MonitorViews.heading(requireContext(), I18n.text("m_ce0d1c8bffa4"), UI.muted(requireContext(), monitors.length() + I18n.text("m_7aaf96ad52e0"))), UI.LG);
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
        String headline = counts.total == 0 ? I18n.text("m_0692146b00f4") : counts.down > 0 ? I18n.text("m_d82e21620f20")
                : counts.warn > 0 ? I18n.text("m_8f870465b0d0") : counts.unknown > 0 ? I18n.text("m_fac5c7c11d39") : I18n.text("m_7a5db8b2e966");
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
                : overview == null ? I18n.text("m_0c05d493962c") : I18n.text("m_9c7536505d91"), available ? 38 : 24, Theme.p().ink, Typeface.BOLD);
        value.setFontFeatureSettings("tnum");
        UI.addRow(card, value, UI.SM);
        UI.addRow(card, UI.muted(requireContext(), I18n.text("m_715297b3b0b9") + monitors.length() + I18n.text("m_80e59927e937") + counts.total + I18n.text("m_97827a9c3766")), UI.XS);
        if (counts.total > 0) UI.addRow(card, UI.muted(requireContext(), MonitorViews.summary(counts)), UI.SM);
        if (!summaryError.isEmpty()) UI.addRow(card, UI.muted(requireContext(), summaryError), UI.SM);
        return card;
    }

    private View emptyCard() {
        LinearLayout card = UI.card(requireContext());
        card.setGravity(Gravity.CENTER_HORIZONTAL);
        card.addView(UI.strong(requireContext(), I18n.text("m_de10338046d1")));
        TextView copy = UI.muted(requireContext(), I18n.text("m_b863658061be"));
        copy.setGravity(Gravity.CENTER);
        UI.addRow(card, copy, UI.SM);
        TextView create = UI.button(requireContext(), creating ? I18n.text("m_7c0305bc0ce6") : I18n.text("m_b466199f99b9"), UI.BTN_PRIMARY);
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
        if (counts.total == 0) status.setText(I18n.text("m_80a57e03f071"));
        card.addView(MonitorViews.heading(requireContext(), monitor.optString("name", I18n.text("m_ce0d1c8bffa4")), status));
        UI.addRow(card, UI.muted(requireContext(), counts.total + I18n.text("m_ec0d8d41c8b8") + monitor.optInt("interval_sec")
                + I18n.text("m_501a9b32374a") + (monitor.optBoolean("publish_enabled") ? I18n.text("m_8e1f5e13ff18") : I18n.text("m_abcd7f608563"))), UI.SM);
        if (counts.total > 0) UI.addRow(card, UI.muted(requireContext(), MonitorViews.summary(counts)), UI.XS);
        JSONObject target = MonitorViews.selectedTarget(targets, "");
        if (target != null) {
            MonitorHistory history = MonitorViews.history(target);
            UI.addRow(card, UI.muted(requireContext(), I18n.text("m_cecaf4fdd200") + target.optString("name", I18n.text("m_ec309ab207ef"))), UI.MD);
            UI.addRow(card, new MonitorHistoryView(requireContext(), history, false, 0), UI.XS);
            UI.addRow(card, UI.muted(requireContext(), history.samples.size() + I18n.text("m_497f6778fd27")
                    + MonitorViews.time(history.lastTime())), UI.XS);
            UI.addRow(card, UI.text(requireContext(), I18n.text("m_42bd6da6129e") + MonitorViews.latency(target) + " · "
                    + MonitorViews.stateLabel(target.optString("state")), 12, MonitorViews.color(target.optString("state")), Typeface.NORMAL), UI.XS);
        } else {
            UI.addRow(card, UI.muted(requireContext(), I18n.text("m_7b9c2ecf8593")), UI.MD);
        }
        TextView open = UI.text(requireContext(), I18n.text("m_49378cc6f396"), 13, Theme.p().success, Typeface.BOLD);
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
        EditText name = UI.input(requireContext(), I18n.text("m_f34513a8e6f4"));
        name.setSingleLine(true);
        name.setFilters(new android.text.InputFilter[]{new android.text.InputFilter.LengthFilter(60)});
        Modal.of(requireContext(), I18n.text("m_f9313095ee64"))
                .message(I18n.text("m_996ba67e9a60"))
                .content(name).cancel(I18n.text("m_2cd0f3be8738"))
                .confirm(I18n.text("m_cde2cd071d25"), false, () -> create(name.getText().toString().trim())).show();
    }

    private void create(String name) {
        if (creating) return;
        if (name.isEmpty()) {
            toast(I18n.text("m_68aec18ecc28"));
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
        Modal.of(requireContext(), I18n.text("m_b2fc749edfa3"))
                .message(I18n.text("m_fbc9178f86b7") + name + I18n.text("m_1b54e6f260d4"))
                .cancel(I18n.text("m_2cd0f3be8738")).confirm(I18n.text("m_a3ea3c17b401"), true, () -> Api.async(() -> Api.delete("/api/monitors/" + id),
                        result -> load(), failure -> toast(failure.getMessage()))).show();
    }

    private void toast(String message) {
        if (message != null && !message.isEmpty() && alive()) {
            android.widget.Toast.makeText(requireContext(), message, android.widget.Toast.LENGTH_LONG).show();
        }
    }
}
