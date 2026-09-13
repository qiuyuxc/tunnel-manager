package com.tunnelmanager.app;

import android.graphics.Typeface;
import android.os.Bundle;
import android.text.InputType;
import android.text.TextUtils;
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

/**
 * 隧道详情 — one tunnel's ingress rules, and the lock/delete actions.
 *
 * The web console lays the rules out as a two column grid of cards; on a phone
 * one rule per card reads better, and editing opens a sheet instead of a modal
 * because the form is two fields and a keyboard.
 */
public class TunnelDetailFragment extends PageFragment {

    private static final String ARG_ID = "id";

    static TunnelDetailFragment forId(String id) {
        TunnelDetailFragment fragment = new TunnelDetailFragment();
        Bundle args = new Bundle();
        args.putString(ARG_ID, id);
        fragment.setArguments(args);
        return fragment;
    }

    private String tunnelId = "";
    private SwipeRefreshLayout refresh;
    private LinearLayout body;
    private String currentId = "";

    @Override
    public void onCreate(@Nullable Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        if (getArguments() != null) tunnelId = getArguments().getString(ARG_ID, "");
    }

    @Override
    String route() {
        return "/tunnels/" + tunnelId;
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

        body.addView(UI.muted(requireContext(), "加载中…"));
        load();
        return refresh;
    }

    private void load() {
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("detail", Api.get("/api/tunnels/" + tunnelId));
            payload.put("config", Api.get("/api/config"));
            return payload;
        }, payload -> {
            refresh.setRefreshing(false);
            JSONObject config = payload.optJSONObject("config");
            currentId = config == null ? "" : config.optString("tunnel_id", "");
            render(payload.optJSONObject("detail"));
        }, failure -> {
            refresh.setRefreshing(false);
            renderFailure(failure.getMessage());
        });
    }

    // --------------------------------------------------------------- rendering

    private void renderFailure(String message) {
        if (body == null || !alive()) return;
        body.removeAllViews();
        body.addView(UI.banner(requireContext(), message == null ? "加载失败" : message, true));
    }

    private void render(JSONObject detail) {
        if (body == null || !alive()) return;
        body.removeAllViews();
        if (detail == null) {
            renderFailure("无法加载隧道详情");
            return;
        }

        Palette p = Theme.p();
        String name = detail.optString("name", "");
        if (name.isEmpty()) name = "(未命名)";
        final String label = name;
        String status = detail.optString("status");
        boolean locked = tunnelId.equals(currentId);

        body.addView(UI.pageTitle(requireContext(), name));

        LinearLayout meta = UI.row(requireContext());
        meta.setGravity(Gravity.CENTER_VERTICAL);
        UI.margin(meta, 0, UI.XS, 0, UI.MD);
        meta.addView(UI.muted(requireContext(), "隧道详情"));
        TextView pill = UI.statusPill(requireContext(),
                "healthy".equals(status) ? "ok" : "degraded".equals(status) ? "warn" : "down");
        UI.margin(pill, UI.SM, 0, 0, 0);
        meta.addView(pill);
        body.addView(meta);

        body.addView(infoCard(detail, locked));

        body.addView(UI.spacer(requireContext(), UI.LG));
        LinearLayout actions = UI.row(requireContext());
        if (locked) {
            TextView unlock = UI.button(requireContext(), "取消锁定", UI.BTN_SECONDARY);
            unlock.setOnClickListener(v -> select("", ""));
            UI.weight(unlock, 1f);
            actions.addView(unlock);
        } else {
            TextView lock = UI.button(requireContext(), "锁定这条隧道", UI.BTN_PRIMARY);
            lock.setOnClickListener(v -> select(tunnelId, label));
            UI.weight(lock, 1f);
            actions.addView(lock);
        }
        TextView remove = UI.button(requireContext(), "删除隧道", UI.BTN_DANGER);
        remove.setOnClickListener(v -> askDelete(label));
        UI.weight(remove, 1f);
        UI.margin(remove, UI.SM, 0, 0, 0);
        actions.addView(remove);
        body.addView(actions);

        body.addView(UI.spacer(requireContext(), UI.XL));
        body.addView(routesHeader(detail.optJSONArray("ingress")));
        body.addView(UI.spacer(requireContext(), UI.SM));
        body.addView(routesList(detail.optJSONArray("ingress")));
    }

    private View infoCard(JSONObject detail, boolean locked) {
        Palette p = Theme.p();
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.label(requireContext(), "隧道信息"));

        TextView idLabel = UI.label(requireContext(), "隧道 ID");
        UI.margin(idLabel, 0, UI.MD, 0, 0);
        card.addView(idLabel);

        TextView id = UI.mono(requireContext(), detail.optString("id"), p.ink);
        id.setTextIsSelectable(true);
        UI.margin(id, 0, UI.XS, 0, 0);
        card.addView(id);

        LinearLayout lockRow = UI.row(requireContext());
        lockRow.setGravity(Gravity.CENTER_VERTICAL);
        UI.margin(lockRow, 0, UI.MD, 0, 0);
        lockRow.addView(UI.tag(requireContext(), locked ? "当前使用" : "未锁定", locked));
        card.addView(lockRow);
        return card;
    }

    private View routesHeader(@Nullable JSONArray ingress) {
        int count = ingress == null ? 0 : ingress.length();
        LinearLayout head = UI.row(requireContext());
        head.setGravity(Gravity.CENTER_VERTICAL);
        head.addView(UI.label(requireContext(), "已发布应用程序路由 · " + count),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));

        TextView add = UI.button(requireContext(), "添加路由", UI.BTN_SECONDARY);
        add.setOnClickListener(v -> openForm(null));
        head.addView(add);
        return head;
    }

    private View routesList(@Nullable JSONArray ingress) {
        Palette p = Theme.p();
        if (ingress == null || ingress.length() == 0) {
            LinearLayout card = UI.card(requireContext());
            card.setGravity(Gravity.CENTER_HORIZONTAL);
            ImageView icon = new ImageView(requireContext());
            icon.setImageResource(R.drawable.ic_nav_domain);
            icon.setColorFilter(p.mute);
            card.addView(icon, new LinearLayout.LayoutParams(UI.dp(28), UI.dp(28)));
            TextView empty = UI.muted(requireContext(), "这条隧道还没有路由规则");
            UI.margin(empty, 0, UI.SM, 0, 0);
            card.addView(empty);
            return card;
        }

        LinearLayout list = UI.column(requireContext());
        for (int i = 0; i < ingress.length(); i++) {
            JSONObject rule = ingress.optJSONObject(i);
            if (rule == null) continue;
            if (i > 0) list.addView(UI.spacer(requireContext(), UI.SM));
            list.addView(routeCard(rule));
        }
        return list;
    }

    private View routeCard(JSONObject rule) {
        Palette p = Theme.p();
        String hostname = rule.optString("hostname", "");
        LinearLayout card = UI.card(requireContext());

        LinearLayout head = UI.row(requireContext());
        head.setGravity(Gravity.CENTER_VERTICAL);

        TextView host = UI.text(requireContext(), hostname.isEmpty() ? "Catch-all 兜底规则" : hostname,
                14, p.ink, Typeface.BOLD);
        host.setSingleLine(true);
        host.setEllipsize(TextUtils.TruncateAt.END);
        if (hostname.isEmpty()) host.setTextColor(p.mute);
        head.addView(host, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));

        // The catch-all rule is Cloudflare's, not the operator's: it cannot be
        // edited or removed, which is why the console hides those buttons too.
        if (!hostname.isEmpty()) {
            head.addView(UI.iconButton(requireContext(), R.drawable.ic_nav_edit, p.body, v -> openForm(rule)));
            head.addView(UI.iconButton(requireContext(), R.drawable.ic_nav_trash, p.error, v -> askDeleteRoute(hostname)));
        }
        card.addView(head);

        TextView service = UI.mono(requireContext(), rule.optString("service", ""), p.body);
        service.setSingleLine(true);
        service.setEllipsize(TextUtils.TruncateAt.MIDDLE);
        UI.margin(service, 0, UI.XS, 0, 0);
        card.addView(service);
        return card;
    }

    // ------------------------------------------------------------------ actions

    private void select(String id, String name) {
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("id", id);
            payload.put("name", name);
            return Api.post("/api/config/tunnel", payload);
        }, ok -> {
            currentId = id;
            toast(id.isEmpty() ? "已取消锁定" : "已锁定「" + name + "」");
            load();
        }, failure -> toast(failure.getMessage()));
    }

    /** {@code rule} null means add; otherwise the hostname is the key being edited. */
    private void openForm(@Nullable JSONObject rule) {
        boolean editing = rule != null;
        EditText hostname = UI.input(requireContext(), "app.example.com");
        hostname.setInputType(InputType.TYPE_TEXT_VARIATION_URI);
        EditText service = UI.input(requireContext(), "http://localhost:8080");
        service.setInputType(InputType.TYPE_TEXT_VARIATION_URI);
        if (editing) {
            hostname.setText(rule.optString("hostname", ""));
            service.setText(rule.optString("service", ""));
        }

        LinearLayout form = UI.column(requireContext());
        form.addView(UI.field(requireContext(), "主机名", hostname));
        form.addView(UI.field(requireContext(), "服务地址", service, UI.MD));

        TextView save = UI.button(requireContext(), editing ? "保存修改" : "添加路由", UI.BTN_PRIMARY);
        UI.fill(save);
        UI.margin(save, 0, UI.MD, 0, 0);
        save.setOnClickListener(v -> submitForm(editing, rule,
                hostname.getText().toString().trim(), service.getText().toString().trim()));
        form.addView(save);

        Sheet.of(requireContext(), editing ? "编辑路由" : "添加路由").content(form).show();
    }

    private void submitForm(boolean editing, @Nullable JSONObject rule, String hostname, String service) {
        if (service.isEmpty()) {
            toast("服务地址要填");
            return;
        }
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("hostname", hostname);
            payload.put("service", service);
            if (editing) {
                payload.put("old_hostname", rule.optString("hostname", ""));
                return Api.put("/api/tunnels/" + tunnelId + "/ingress", payload);
            }
            return Api.post("/api/tunnels/" + tunnelId + "/ingress", payload);
        }, ok -> {
            Sheet.dismissVisible();
            toast(editing ? "路由已更新" : "路由已添加");
            load();
        }, failure -> toast(failure.getMessage()));
    }

    private void askDeleteRoute(String hostname) {
        Palette p = Theme.p();
        UI.Toggle toggle = new UI.Toggle(requireContext(), true);

        LinearLayout option = UI.row(requireContext());
        option.setGravity(Gravity.CENTER_VERTICAL);
        LinearLayout text = UI.column(requireContext());
        text.addView(UI.strong(requireContext(), "同时删除 DNS 记录"));
        TextView hint = UI.muted(requireContext(), "清理指向该主机名的解析，避免留下悬空记录");
        UI.margin(hint, 0, UI.XS, 0, 0);
        text.addView(hint);
        option.addView(text, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        LinearLayout.LayoutParams toggleLp = new LinearLayout.LayoutParams(UI.dp(34), UI.dp(20));
        toggleLp.leftMargin = UI.dp(UI.MD);
        option.addView(toggle, toggleLp);
        option.setClickable(true);
        option.setOnClickListener(v -> toggle.setOn(!toggle.isOn()));

        Modal.of(requireContext(), "删除路由")
                .message("确定删除路由 " + hostname + " 吗？此操作无法撤销。")
                .content(option)
                .cancel("取消")
                .confirm("确认删除", true, () -> deleteRoute(hostname, toggle.isOn()))
                .show();
    }

    private void deleteRoute(String hostname, boolean deleteDns) {
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("hostname", hostname);
            payload.put("delete_dns", deleteDns);
            return Api.delete("/api/tunnels/" + tunnelId + "/ingress", payload);
        }, result -> {
            String warning = result.optString("dns_warning", "");
            if (!warning.isEmpty()) {
                toast("路由已删除，但 DNS 未清理：" + warning);
            } else if (deleteDns) {
                toast("路由已删除，同时清理了 " + result.optInt("dns_deleted") + " 条 DNS 记录");
            } else {
                toast("路由已删除");
            }
            load();
        }, failure -> toast(failure.getMessage()));
    }

    private void askDelete(String name) {
        Modal.of(requireContext(), "删除隧道")
                .message("确定删除「" + name + "」吗？此操作无法撤销，隧道仍有活动连接时 Cloudflare 会拒绝删除。")
                .cancel("取消")
                .confirm("确认删除", true, () -> Api.async(
                        () -> Api.delete("/api/tunnels/" + tunnelId),
                        ok -> {
                            toast("隧道已删除");
                            openRoute("/tunnels");
                        },
                        failure -> toast(failure.getMessage())))
                .show();
    }

    private void toast(String message) {
        if (message == null || message.isEmpty() || !alive()) return;
        android.widget.Toast.makeText(requireContext(), message, android.widget.Toast.LENGTH_LONG).show();
    }
}
