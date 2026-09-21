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
        header.addView(UI.pageTitle(requireContext(), "隧道管理"),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        ImageView reload = UI.iconButton(requireContext(), R.drawable.ic_nav_refresh, Theme.p().body, view -> reload());
        reload.setContentDescription("刷新隧道");
        header.addView(reload);
        ImageView create = UI.iconButton(requireContext(), R.drawable.ic_nav_plus, Theme.p().success, view -> console().showCreateTunnel());
        create.setContentDescription("新建隧道");
        header.addView(create);
        body.addView(header);
        UI.addRow(body, UI.muted(requireContext(), "从本地服务，到世界的另一端。"), UI.XS);

        EditText search = UI.input(requireContext(), "搜索隧道名称或 ID");
        search.setSingleLine(true);
        search.setSaveEnabled(false);
        search.setText(query);
        search.setContentDescription("搜索隧道名称或 ID");
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
        labels[0] = "全部";
        for (int index = 1; index < labels.length; index++) labels[index] = TunnelFilter.State.values()[index - 1].label;
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
            String label = index == 0 ? "全部" : TunnelFilter.State.values()[index - 1].label;
            ((TextView) filters.getChildAt(index)).setText(label + (tunnels == null ? "" : " " + counts[index]));
        }
        renderList();
        UI.restoreScroll(scroll, position);
    }

    private View currentCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.label(requireContext(), "当前隧道"));
        TextView title = UI.strong(requireContext(), currentId.isEmpty() ? "尚未选择" : currentName.isEmpty() ? "未命名隧道" : currentName);
        UI.addRow(card, title, UI.SM);
        UI.addRow(card, UI.muted(requireContext(), currentId.isEmpty()
                ? "从列表的操作菜单选择隧道，用于后续域名绑定。"
                : "域名绑定将使用这条隧道，可在操作菜单中切换或取消锁定。"), UI.XS);
        return card;
    }

    private void renderList() {
        if (!alive() || listSlot == null) return;
        listSlot.removeAllViews();
        if (!error.isEmpty()) {
            listSlot.addView(UI.banner(requireContext(), error, true));
            TextView retry = UI.button(requireContext(), "重新加载", UI.BTN_SECONDARY);
            retry.setOnClickListener(view -> reload());
            UI.addRow(listSlot, retry, UI.MD);
            return;
        }
        if (tunnels == null) {
            listSlot.addView(UI.muted(requireContext(), "正在加载隧道…"));
            return;
        }
        if (tunnels.length() == 0) {
            LinearLayout empty = UI.card(requireContext());
            empty.addView(UI.strong(requireContext(), "还没有隧道"));
            UI.addRow(empty, UI.muted(requireContext(), "创建一条隧道，获取连接命令，再把它交给要接入的主机。"), UI.SM);
            TextView create = UI.button(requireContext(), "创建第一条隧道", UI.BTN_PRIMARY);
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
        listSlot.addView(UI.label(requireContext(), "筛选结果 · " + count + " 条"));
        if (count == 0) {
            UI.addRow(listSlot, UI.muted(requireContext(), "没有匹配的隧道，试试其他名称、ID 或状态。"), UI.MD);
        } else {
            UI.addRow(listSlot, list, UI.SM);
        }
    }

    private View tunnelRow(JSONObject tunnel) {
        String id = tunnel.optString("id");
        String name = tunnel.optString("name", "").isEmpty() ? "未命名隧道" : tunnel.optString("name");
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
            TextView current = UI.text(requireContext(), "当前使用", 12, Theme.p().success, Typeface.NORMAL);
            UI.margin(current, UI.SM, 0, 0, 0);
            meta.addView(current);
        }
        UI.addRow(text, meta, UI.SM);
        row.addView(text, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        ImageView more = UI.iconButton(requireContext(), R.drawable.ic_nav_more, Theme.p().body, view -> openActions(tunnel));
        more.setContentDescription("隧道操作：" + name);
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
        String name = tunnel.optString("name", "").isEmpty() ? "未命名隧道" : tunnel.optString("name");
        Sheet.Builder sheet = Sheet.of(requireContext(), name)
                .action(R.drawable.ic_nav_tunnels, "查看详情", "路由规则与隧道信息", () -> openRoute("/tunnels/" + id));
        if (id.equals(currentId)) sheet.item(R.drawable.ic_nav_close, "取消锁定", () -> select("", ""));
        else sheet.item(R.drawable.ic_nav_check, "锁定这条隧道", () -> select(id, name));
        sheet.item(R.drawable.ic_nav_copy, "复制隧道 ID", () -> copy(id));
        sheet.item(R.drawable.ic_nav_trash, "删除隧道", () -> askDelete(id, name));
        sheet.show();
    }

    private void select(String id, String name) {
        Api.async(() -> Api.post("/api/config/tunnel", new JSONObject().put("id", id).put("name", name)), result -> {
            toast(id.isEmpty() ? "已取消锁定" : "已锁定「" + name + "」");
            reload();
        }, failure -> toast(failure.getMessage()));
    }

    private void askDelete(String id, String name) {
        Modal.of(requireContext(), "删除隧道")
                .message("确定删除「" + name + "」吗？此操作无法撤销，隧道仍有活动连接时 Cloudflare 会拒绝删除。")
                .cancel("取消")
                .confirm("确认删除", true, () -> Api.async(() -> Api.delete("/api/tunnels/" + id), result -> {
                    toast("隧道已删除");
                    reload();
                }, failure -> toast(failure.getMessage())))
                .show();
    }

    private void copy(String id) {
        if (id.isEmpty()) return;
        ClipboardManager clipboard = (ClipboardManager) requireContext().getSystemService(Context.CLIPBOARD_SERVICE);
        if (clipboard == null) return;
        clipboard.setPrimaryClip(ClipData.newPlainText("隧道 ID", id));
        if (Build.VERSION.SDK_INT < 33) toast("已复制");
    }

    private void toast(String message) {
        if (message == null || message.isEmpty() || !alive()) return;
        android.widget.Toast.makeText(requireContext(), message, android.widget.Toast.LENGTH_LONG).show();
    }
}
