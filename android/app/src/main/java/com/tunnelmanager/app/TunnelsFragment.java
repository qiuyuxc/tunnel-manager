package com.tunnelmanager.app;

import android.content.ClipData;
import android.content.ClipboardManager;
import android.content.Context;
import android.graphics.Typeface;
import android.os.Build;
import android.os.Bundle;
import android.text.Editable;
import android.text.TextUtils;
import android.text.TextWatcher;
import android.view.Gravity;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.EditText;
import android.widget.FrameLayout;
import android.widget.HorizontalScrollView;
import android.widget.ImageView;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout;

import org.json.JSONArray;
import org.json.JSONObject;

public class TunnelsFragment extends PageFragment {
    private SwipeRefreshLayout refresh;
    private ScrollView scroll;
    private LinearLayout currentSlot;
    private LinearLayout listSlot;
    private LinearLayout filters;
    private JSONArray tunnels;
    private String currentId = "";
    private String currentName = "";
    private String query = "";
    private String error = "";
    private int selectedFilter;
    private int generation;

    @Override String route() {
        return "/tunnels";
    }

    @Override public void onCreate(@Nullable Bundle saved) {
        super.onCreate(saved);
        if (saved != null) {
            query = saved.getString("tunnel_query", "");
            selectedFilter = Math.max(0, Math.min(TunnelFilter.State.values().length, saved.getInt("tunnel_filter")));
        }
    }

    @Override public void onSaveInstanceState(@NonNull Bundle saved) {
        super.onSaveInstanceState(saved);
        saved.putString("tunnel_query", query);
        saved.putInt("tunnel_filter", selectedFilter);
    }

    @Override protected View build(@NonNull LayoutInflater inflater, @Nullable ViewGroup container) {
        refresh = new SwipeRefreshLayout(requireContext());
        refresh.setOnRefreshListener(this::reload);
        scroll = new ScrollView(requireContext());
        LinearLayout body = UI.column(requireContext());
        UI.pagePadding(body);
        scroll.addView(body);
        refresh.addView(scroll);

        LinearLayout header = UI.row(requireContext());
        header.addView(UI.pageTitle(requireContext(), I18n.text("m_e920f47d0c05")),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        ImageView reload = UI.iconButton(requireContext(), R.drawable.ic_nav_refresh, Theme.p().body, view -> reload());
        reload.setContentDescription(I18n.text("m_5f79b57ee40f"));
        header.addView(reload);
        ImageView create = UI.iconButton(requireContext(), R.drawable.ic_nav_plus, Theme.p().success, view -> console().showCreateTunnel());
        create.setContentDescription(I18n.text("m_ee6527b22aaf"));
        header.addView(create);
        body.addView(header);
        UI.addRow(body, UI.muted(requireContext(), I18n.text("m_6f4973626b77")), UI.XS);

        EditText search = UI.input(requireContext(), I18n.text("m_111798a8646d"));
        search.setSingleLine(true);
        search.setSaveEnabled(false);
        search.setText(query);
        search.setContentDescription(I18n.text("m_111798a8646d"));
        search.addTextChangedListener(new TextWatcher() {
            @Override public void beforeTextChanged(CharSequence text, int start, int count, int after) {}
            @Override public void onTextChanged(CharSequence text, int start, int before, int count) {}
            @Override public void afterTextChanged(Editable text) {
                query = text.toString();
                renderList();
            }
        });
        UI.addRow(body, search, UI.LG);
        String[] labels = new String[TunnelFilter.State.values().length + 1];
        labels[0] = I18n.text("m_5c55a67935af");
        for (int index = 1; index < labels.length; index++) labels[index] = TunnelFilter.State.values()[index - 1].label();
        filters = UI.segmented(requireContext(), labels, selectedFilter, selected -> {
            selectedFilter = selected;
            renderList();
        });
        for (int index = 0; index < filters.getChildCount(); index++) {
            TextView chip = (TextView) filters.getChildAt(index);
            chip.setLayoutParams(new LinearLayout.LayoutParams(ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT));
            chip.setPadding(UI.dp(14), UI.dp(UI.SM), UI.dp(14), UI.dp(UI.SM));
            chip.setMinHeight(UI.dp(44));
        }
        HorizontalScrollView filterScroll = new HorizontalScrollView(requireContext());
        filterScroll.setHorizontalScrollBarEnabled(false);
        filterScroll.addView(filters, new FrameLayout.LayoutParams(ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT));
        UI.addRow(body, filterScroll, UI.MD);
        currentSlot = UI.column(requireContext());
        UI.addRow(body, currentSlot, UI.MD);
        listSlot = UI.column(requireContext());
        UI.addRow(body, listSlot, UI.LG);
        render();
        filterScroll.post(() -> {
            if (filters != null) filterScroll.smoothScrollTo(filters.getChildAt(selectedFilter).getLeft(), 0);
        });
        return refresh;
    }

    @Override public void onResume() {
        super.onResume();
        reload();
    }

    @Override public void onDestroyView() {
        generation++;
        refresh = null;
        scroll = null;
        listSlot = null;
        currentSlot = null;
        filters = null;
        super.onDestroyView();
    }

    void reload() {
        if (!alive() || refresh == null) return;
        int request = ++generation;
        refresh.setRefreshing(true);
        Api.async(() -> new JSONObject()
                .put("tunnels", Api.getArray("/api/tunnels"))
                .put("config", Api.get("/api/config")), payload -> {
            if (!alive() || refresh == null || request != generation) return;
            refresh.setRefreshing(false);
            JSONObject config = payload.optJSONObject("config");
            currentId = config == null ? "" : config.optString("tunnel_id", "");
            currentName = config == null ? "" : config.optString("tunnel_name", "");
            tunnels = payload.optJSONArray("tunnels");
            error = "";
            render();
        }, failure -> {
            if (!alive() || refresh == null || request != generation) return;
            refresh.setRefreshing(false);
            error = failure.getMessage();
            render();
        });
    }

    private void render() {
        if (!alive() || listSlot == null) return;
        int position = scroll.getScrollY();
        currentSlot.removeAllViews();
        if (tunnels != null && error.isEmpty()) currentSlot.addView(currentCard());
        int[] counts = new int[TunnelFilter.State.values().length + 1];
        if (tunnels != null) {
            for (int index = 0; index < tunnels.length(); index++) {
                JSONObject tunnel = tunnels.optJSONObject(index);
                if (tunnel == null) continue;
                counts[0]++;
                counts[TunnelFilter.state(tunnel.optString("status")).ordinal() + 1]++;
            }
        }
        for (int index = 0; index < counts.length; index++) {
            String label = index == 0 ? I18n.text("m_5c55a67935af") : TunnelFilter.State.values()[index - 1].label();
            ((TextView) filters.getChildAt(index)).setText(label + (tunnels == null ? "" : " " + counts[index]));
        }
        renderList();
        UI.restoreScroll(scroll, position);
    }

    private View currentCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.label(requireContext(), I18n.text("m_b903d203f70a")));
        TextView title = UI.strong(requireContext(), currentId.isEmpty() ? I18n.text("m_c5768ca9726c") : currentName.isEmpty() ? I18n.text("m_80fa39fefe91") : currentName);
        UI.addRow(card, title, UI.SM);
        UI.addRow(card, UI.muted(requireContext(), currentId.isEmpty()
                ? I18n.text("m_4f84f8847292")
                : I18n.text("m_efefd6cb6c71")), UI.XS);
        return card;
    }

    private void renderList() {
        if (!alive() || listSlot == null) return;
        listSlot.removeAllViews();
        if (!error.isEmpty()) {
            listSlot.addView(UI.banner(requireContext(), error, true));
            TextView retry = UI.button(requireContext(), I18n.text("m_7bdd5ce1e298"), UI.BTN_SECONDARY);
            retry.setOnClickListener(view -> reload());
            UI.addRow(listSlot, retry, UI.MD);
            return;
        }
        if (tunnels == null) {
            listSlot.addView(UI.muted(requireContext(), I18n.text("m_8d725908f6c1")));
            return;
        }
        if (tunnels.length() == 0) {
            LinearLayout empty = UI.card(requireContext());
            empty.addView(UI.strong(requireContext(), I18n.text("m_29bde9a99a93")));
            UI.addRow(empty, UI.muted(requireContext(), I18n.text("m_87cfef35ce5d")), UI.SM);
            TextView create = UI.button(requireContext(), I18n.text("m_596a9e55c9ef"), UI.BTN_PRIMARY);
            create.setOnClickListener(view -> console().showCreateTunnel());
            UI.addRow(empty, create, UI.MD);
            listSlot.addView(empty);
            return;
        }
        LinearLayout list = UI.card(requireContext());
        list.setPadding(UI.dp(UI.MD), 0, UI.dp(UI.MD), 0);
        int count = 0;
        for (int index = 0; index < tunnels.length(); index++) {
            JSONObject tunnel = tunnels.optJSONObject(index);
            if (tunnel == null || !TunnelFilter.matches(tunnel.optString("name"), tunnel.optString("id"),
                    tunnel.optString("status"), query, selectedFilter)) continue;
            if (count++ > 0) list.addView(UI.divider(requireContext()));
            list.addView(tunnelRow(tunnel));
        }
        listSlot.addView(UI.label(requireContext(), I18n.text("m_d218b0776515") + count + I18n.text("m_6fb2840d568a")));
        if (count == 0) {
            UI.addRow(listSlot, UI.muted(requireContext(), I18n.text("m_5a0e866d3bdb")), UI.MD);
        } else {
            UI.addRow(listSlot, list, UI.SM);
        }
    }

    private View tunnelRow(JSONObject tunnel) {
        String id = tunnel.optString("id");
        String name = tunnel.optString("name", "").isEmpty() ? I18n.text("m_80fa39fefe91") : tunnel.optString("name");
        LinearLayout row = UI.row(requireContext());
        row.setPadding(0, UI.dp(UI.MD), 0, UI.dp(UI.MD));
        LinearLayout text = UI.column(requireContext());
        TextView title = UI.text(requireContext(), name, 15, Theme.p().ink, Typeface.BOLD);
        text.addView(title);
        TextView identifier = UI.mono(requireContext(), id, Theme.p().mute);
        identifier.setSingleLine(true);
        identifier.setEllipsize(TextUtils.TruncateAt.MIDDLE);
        UI.addRow(text, identifier, UI.XS);
        LinearLayout meta = UI.row(requireContext());
        meta.addView(UI.tunnelStatusPill(requireContext(), tunnel.optString("status")));
        if (id.equals(currentId)) {
            TextView current = UI.text(requireContext(), I18n.text("m_5ddff4afab27"), 12, Theme.p().success, Typeface.NORMAL);
            UI.margin(current, UI.SM, 0, 0, 0);
            meta.addView(current);
        }
        UI.addRow(text, meta, UI.SM);
        row.addView(text, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        ImageView more = UI.iconButton(requireContext(), R.drawable.ic_nav_more, Theme.p().body, view -> openActions(tunnel));
        more.setContentDescription(I18n.text("m_f4f119412d35") + name);
        UI.margin(more, UI.SM, 0, 0, 0);
        row.addView(more);
        row.setBackground(UI.pressable(UI.rounded(Theme.p().canvasRaised, UI.RADIUS_MD), Theme.p().btnGhostHover));
        row.setClickable(true);
        row.setFocusable(true);
        UI.pressFeedback(row);
        row.setOnClickListener(view -> openRoute("/tunnels/" + id));
        return row;
    }

    private void openActions(JSONObject tunnel) {
        String id = tunnel.optString("id");
        String name = tunnel.optString("name", "").isEmpty() ? I18n.text("m_80fa39fefe91") : tunnel.optString("name");
        Sheet.Builder sheet = Sheet.of(requireContext(), name)
                .action(R.drawable.ic_nav_tunnels, I18n.text("m_a748cc074f78"), I18n.text("m_e14ac5b6d3e8"), () -> openRoute("/tunnels/" + id));
        if (id.equals(currentId)) sheet.item(R.drawable.ic_nav_close, I18n.text("m_2446e5438f16"), () -> select("", ""));
        else sheet.item(R.drawable.ic_nav_check, I18n.text("m_017c754f6e8b"), () -> select(id, name));
        sheet.item(R.drawable.ic_nav_copy, I18n.text("m_e4454a660640"), () -> copy(id));
        sheet.item(R.drawable.ic_nav_trash, I18n.text("m_cf5e530b1ad3"), () -> askDelete(id, name));
        sheet.show();
    }

    private void select(String id, String name) {
        Api.async(() -> Api.post("/api/config/tunnel", new JSONObject().put("id", id).put("name", name)), result -> {
            toast(id.isEmpty() ? I18n.text("m_af906a2476a2") : I18n.text("m_a9ca30fdc300") + name + "」");
            reload();
        }, failure -> toast(failure.getMessage()));
    }

    private void askDelete(String id, String name) {
        Modal.of(requireContext(), I18n.text("m_cf5e530b1ad3"))
                .message(I18n.text("m_72f49a525798") + name + I18n.text("m_a17e8f4371a0"))
                .cancel(I18n.text("m_2cd0f3be8738"))
                .confirm(I18n.text("m_a3ea3c17b401"), true, () -> Api.async(() -> Api.delete("/api/tunnels/" + id), result -> {
                    toast(I18n.text("m_72d83d119709"));
                    reload();
                }, failure -> toast(failure.getMessage())))
                .show();
    }

    private void copy(String id) {
        if (id.isEmpty()) return;
        ClipboardManager clipboard = (ClipboardManager) requireContext().getSystemService(Context.CLIPBOARD_SERVICE);
        if (clipboard == null) return;
        clipboard.setPrimaryClip(ClipData.newPlainText(I18n.text("m_17f261a9385c"), id));
        if (Build.VERSION.SDK_INT < 33) toast(I18n.text("m_8f6f8d979c98"));
    }

    private void toast(String message) {
        if (message == null || message.isEmpty() || !alive()) return;
        android.widget.Toast.makeText(requireContext(), message, android.widget.Toast.LENGTH_LONG).show();
    }
}
