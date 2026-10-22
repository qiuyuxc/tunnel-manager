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

        body.addView(UI.muted(requireContext(), I18n.text("m_4927a53bcc88")));
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
        body.addView(UI.banner(requireContext(), message == null ? I18n.text("m_d1d044826a45") : message, true));
    }

    private void render(JSONObject detail) {
        if (body == null || !alive()) return;
        body.removeAllViews();
        if (detail == null) {
            renderFailure(I18n.text("m_70dbcc69a01d"));
            return;
        }

        Palette p = Theme.p();
        String name = detail.optString("name", "");
        if (name.isEmpty()) name = I18n.text("m_91315b982b04");
        final String label = name;
        String status = detail.optString("status");
        boolean locked = tunnelId.equals(currentId);

        body.addView(UI.pageTitle(requireContext(), name));

        LinearLayout meta = UI.row(requireContext());
        meta.setGravity(Gravity.CENTER_VERTICAL);
        UI.margin(meta, 0, UI.XS, 0, UI.MD);
        meta.addView(UI.muted(requireContext(), I18n.text("m_094f104f556a")));
        TextView pill = UI.tunnelStatusPill(requireContext(), status);
        UI.margin(pill, UI.SM, 0, 0, 0);
        meta.addView(pill);
        body.addView(meta);

        body.addView(infoCard(detail, locked));

        body.addView(UI.spacer(requireContext(), UI.LG));
        LinearLayout actions = UI.row(requireContext());
        if (locked) {
            TextView unlock = UI.button(requireContext(), I18n.text("m_2446e5438f16"), UI.BTN_SECONDARY);
            unlock.setOnClickListener(v -> select("", ""));
            UI.weight(unlock, 1f);
            actions.addView(unlock);
        } else {
            TextView lock = UI.button(requireContext(), I18n.text("m_017c754f6e8b"), UI.BTN_PRIMARY);
            lock.setOnClickListener(v -> select(tunnelId, label));
            UI.weight(lock, 1f);
            actions.addView(lock);
        }
        body.addView(actions);

        body.addView(UI.spacer(requireContext(), UI.XL));
        body.addView(routesHeader(detail.optJSONArray("ingress")));
        body.addView(UI.spacer(requireContext(), UI.SM));
        body.addView(routesList(detail.optJSONArray("ingress")));
        body.addView(UI.spacer(requireContext(), UI.XL));
        body.addView(UI.label(requireContext(), I18n.text("m_cd84121acb17")));
        TextView warning = UI.muted(requireContext(), I18n.text("m_09fabe1bd838"));
        UI.addRow(body, warning, UI.SM);
        TextView remove = UI.button(requireContext(), I18n.text("m_cf5e530b1ad3"), UI.BTN_DANGER);
        remove.setOnClickListener(view -> askDelete(label));
        UI.fill(remove);
        UI.addRow(body, remove, UI.MD);
    }

    private View infoCard(JSONObject detail, boolean locked) {
        Palette p = Theme.p();
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.label(requireContext(), I18n.text("m_2a8dc33c7749")));

        TextView idLabel = UI.label(requireContext(), I18n.text("m_17f261a9385c"));
        UI.margin(idLabel, 0, UI.MD, 0, 0);
        card.addView(idLabel);

        TextView id = UI.mono(requireContext(), detail.optString("id"), p.ink);
        id.setTextIsSelectable(true);
        UI.margin(id, 0, UI.XS, 0, 0);
        card.addView(id);

        LinearLayout lockRow = UI.row(requireContext());
        lockRow.setGravity(Gravity.CENTER_VERTICAL);
        UI.margin(lockRow, 0, UI.MD, 0, 0);
        lockRow.addView(UI.tag(requireContext(), locked ? I18n.text("m_5ddff4afab27") : I18n.text("m_8436399d1ec2"), locked));
        card.addView(lockRow);
        return card;
    }

    private View routesHeader(@Nullable JSONArray ingress) {
        int count = ingress == null ? 0 : ingress.length();
        LinearLayout head = UI.row(requireContext());
        head.setGravity(Gravity.CENTER_VERTICAL);
        head.addView(UI.label(requireContext(), I18n.text("m_27d793992500") + count),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));

        TextView add = UI.button(requireContext(), I18n.text("m_a9ea26f64358"), UI.BTN_SECONDARY);
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
            UI.tint(icon, p.mute);
            card.addView(icon, new LinearLayout.LayoutParams(UI.dp(28), UI.dp(28)));
            TextView empty = UI.muted(requireContext(), I18n.text("m_e26984ddf5b0"));
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

        TextView host = UI.text(requireContext(), hostname.isEmpty() ? I18n.text("m_44e8472cf913") : hostname,
                14, p.ink, Typeface.BOLD);
        host.setTextIsSelectable(true);
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
        service.setTextIsSelectable(true);
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
            toast(id.isEmpty() ? I18n.text("m_af906a2476a2") : I18n.text("m_a9ca30fdc300") + name + "」");
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
        form.addView(UI.field(requireContext(), I18n.text("m_0224c3f6672e"), hostname));
        form.addView(UI.field(requireContext(), I18n.text("m_02739d1bae8b"), service, UI.MD));

        TextView save = UI.button(requireContext(), editing ? I18n.text("m_991bb7cfe5a8") : I18n.text("m_a9ea26f64358"), UI.BTN_PRIMARY);
        UI.fill(save);
        UI.margin(save, 0, UI.MD, 0, 0);
        save.setOnClickListener(v -> submitForm(editing, rule,
                hostname.getText().toString().trim(), service.getText().toString().trim()));
        form.addView(save);

        Sheet.of(requireContext(), editing ? I18n.text("m_daa951cf34e2") : I18n.text("m_a9ea26f64358")).content(form).show();
    }

    private void submitForm(boolean editing, @Nullable JSONObject rule, String hostname, String service) {
        if (service.isEmpty()) {
            toast(I18n.text("m_45cffd20e889"));
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
            toast(editing ? I18n.text("m_f8f23707c6e5") : I18n.text("m_3612a0b57866"));
            load();
        }, failure -> toast(failure.getMessage()));
    }

    private void askDeleteRoute(String hostname) {
        Palette p = Theme.p();
        UI.Toggle toggle = new UI.Toggle(requireContext(), true);

        LinearLayout option = UI.row(requireContext());
        option.setGravity(Gravity.CENTER_VERTICAL);
        LinearLayout text = UI.column(requireContext());
        text.addView(UI.strong(requireContext(), I18n.text("m_ccd191ba0044")));
        TextView hint = UI.muted(requireContext(), I18n.text("m_0b3577e4baaf"));
        UI.margin(hint, 0, UI.XS, 0, 0);
        text.addView(hint);
        option.addView(text, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        LinearLayout.LayoutParams toggleLp = new LinearLayout.LayoutParams(UI.dp(34), UI.dp(20));
        toggleLp.leftMargin = UI.dp(UI.MD);
        option.addView(toggle, toggleLp);
        option.setClickable(true);
        option.setOnClickListener(v -> toggle.setOn(!toggle.isOn()));

        Modal.of(requireContext(), I18n.text("m_81a8aceaea20"))
                .message(I18n.text("m_371298053c5f") + hostname + I18n.text("m_a79a0def3e5e"))
                .content(option)
                .cancel(I18n.text("m_2cd0f3be8738"))
                .confirm(I18n.text("m_a3ea3c17b401"), true, () -> deleteRoute(hostname, toggle.isOn()))
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
                toast(I18n.text("m_14588e04342a") + warning);
            } else if (deleteDns) {
                toast(I18n.text("m_37de6204815f") + result.optInt("dns_deleted") + I18n.text("m_cc53a1da4656"));
            } else {
                toast(I18n.text("m_0265de89db75"));
            }
            load();
        }, failure -> toast(failure.getMessage()));
    }

    private void askDelete(String name) {
        Modal.of(requireContext(), I18n.text("m_cf5e530b1ad3"))
                .message(I18n.text("m_72f49a525798") + name + I18n.text("m_a17e8f4371a0"))
                .cancel(I18n.text("m_2cd0f3be8738"))
                .confirm(I18n.text("m_a3ea3c17b401"), true, () -> Api.async(
                        () -> Api.delete("/api/tunnels/" + tunnelId),
                        ok -> {
                            toast(I18n.text("m_72d83d119709"));
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
