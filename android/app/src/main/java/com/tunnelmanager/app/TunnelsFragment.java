package com.tunnelmanager.app;

import android.content.ClipData;
import android.content.ClipboardManager;
import android.content.Context;
import android.graphics.Typeface;
import android.os.Build;
import android.text.InputFilter;
import android.text.TextUtils;
import android.util.TypedValue;
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
 * 隧道管理 — every Cloudflare Tunnel on the account, plus the one the console is
 * currently pointed at.
 *
 * The web console renders this as a five column table. That does not survive a
 * 384dp screen, so each tunnel is a card carrying only what you scan for (name,
 * status, id, whether it is the active one) and the rest — 详情, 复制 ID, 删除 —
 * sits behind the "⋯" sheet, which is the phone's version of the row menu.
 */
public class TunnelsFragment extends PageFragment {

    private SwipeRefreshLayout refresh;
    private LinearLayout body;
    /** The locked tunnel, read from /api/config on every load. */
    private String currentId = "";
    private String currentName = "";

    @Override
    String route() {
        return "/tunnels";
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

        render(null, null);
        load();
        return refresh;
    }

    /**
     * Tunnels and the current selection come from two endpoints but are one
     * screen; fetching them together keeps the card from flashing "未选择"
     * before the config lands.
     */
    private void load() {
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("tunnels", Api.getArray("/api/tunnels"));
            payload.put("config", Api.get("/api/config"));
            return payload;
        }, payload -> {
            refresh.setRefreshing(false);
            JSONObject config = payload.optJSONObject("config");
            currentId = config == null ? "" : config.optString("tunnel_id", "");
            currentName = config == null ? "" : config.optString("tunnel_name", "");
            render(payload.optJSONArray("tunnels"), null);
        }, failure -> {
            refresh.setRefreshing(false);
            render(null, failure.getMessage());
        });
    }

    // --------------------------------------------------------------- rendering

    /** {@code tunnels} is null while loading and on failure; {@code error} tells the two apart. */
    private void render(@Nullable JSONArray tunnels, @Nullable String error) {
        if (body == null || !alive()) return;
        body.removeAllViews();

        body.addView(UI.pageTitle(requireContext(), "隧道管理"));

        TextView subtitle = UI.muted(requireContext(), "浏览 Cloudflare Tunnel，并锁定当前要管理的隧道");
        UI.margin(subtitle, 0, UI.XS, 0, UI.MD);
        body.addView(subtitle);

        LinearLayout actions = UI.row(requireContext());
        TextView reload = UI.button(requireContext(), "刷新", UI.BTN_SECONDARY);
        reload.setOnClickListener(v -> load());
        UI.weight(reload, 1f);
        actions.addView(reload);

        TextView create = UI.button(requireContext(), "新建隧道", UI.BTN_PRIMARY);
        create.setOnClickListener(v -> askCreate());
        UI.weight(create, 1f);
        UI.margin(create, UI.SM, 0, 0, 0);
        actions.addView(create);
        body.addView(actions);

        if (error != null) {
            body.addView(UI.spacer(requireContext(), UI.MD));
            body.addView(UI.banner(requireContext(), error, true));
            return;
        }
        if (tunnels == null) {
            body.addView(UI.spacer(requireContext(), UI.MD));
            body.addView(UI.muted(requireContext(), "加载中…"));
            return;
        }

        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(currentCard());

        body.addView(UI.spacer(requireContext(), UI.XL));
        body.addView(UI.label(requireContext(), "全部隧道 · " + tunnels.length()));
        body.addView(UI.spacer(requireContext(), UI.SM));

        if (tunnels.length() == 0) {
            body.addView(emptyCard());
            return;
        }
        for (int i = 0; i < tunnels.length(); i++) {
            JSONObject tunnel = tunnels.optJSONObject(i);
            if (tunnel == null) continue;
            if (i > 0) body.addView(UI.spacer(requireContext(), UI.SM));
            body.addView(tunnelCard(tunnel));
        }
    }

    /** Which tunnel the console is pointed at, the page's one piece of global state. */
    private View currentCard() {
        Palette p = Theme.p();
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.label(requireContext(), "当前隧道"));

        if (currentId.isEmpty()) {
            TextView hint = UI.muted(requireContext(), "尚未锁定隧道。锁定后，域名绑定、DNS 管理等页面都会以它为目标。");
            UI.margin(hint, 0, UI.SM, 0, 0);
            card.addView(hint);
            return card;
        }

        TextView name = UI.text(requireContext(), currentName.isEmpty() ? "(未命名)" : currentName,
                16, p.ink, Typeface.BOLD);
        UI.margin(name, 0, UI.SM, 0, 0);
        card.addView(name);

        TextView id = UI.mono(requireContext(), currentId, p.mute);
        id.setSingleLine(true);
        id.setEllipsize(TextUtils.TruncateAt.MIDDLE);
        UI.margin(id, 0, UI.XS, 0, 0);
        card.addView(id);
        return card;
    }

    private View emptyCard() {
        Palette p = Theme.p();
        LinearLayout card = UI.card(requireContext());
        card.setGravity(Gravity.CENTER_HORIZONTAL);

        ImageView icon = new ImageView(requireContext());
        icon.setImageResource(R.drawable.ic_nav_tunnels);
        icon.setColorFilter(p.mute);
        card.addView(icon, new LinearLayout.LayoutParams(UI.dp(30), UI.dp(30)));

        TextView title = UI.strong(requireContext(), "还没有隧道");
        UI.margin(title, 0, UI.MD, 0, 0);
        card.addView(title);

        TextView copy = UI.muted(requireContext(), "创建一条隧道，Cloudflare 会同时给出连接令牌，把它交给要暴露服务的那台机器。");
        copy.setGravity(Gravity.CENTER);
        UI.margin(copy, 0, UI.XS, 0, UI.MD);
        card.addView(copy);

        TextView create = UI.button(requireContext(), "创建第一条隧道", UI.BTN_PRIMARY);
        create.setOnClickListener(v -> askCreate());
        card.addView(create);
        return card;
    }

    private View tunnelCard(JSONObject tunnel) {
        Palette p = Theme.p();
        String id = tunnel.optString("id");
        String name = tunnel.optString("name", "");
        if (name.isEmpty()) name = "(未命名)";
        final String label = name;
        boolean current = id.equals(currentId);

        LinearLayout card = UI.card(requireContext());

        LinearLayout head = UI.row(requireContext());
        head.setGravity(Gravity.CENTER_VERTICAL);

        TextView title = UI.text(requireContext(), name, 15, p.ink, Typeface.BOLD);
        title.setSingleLine(true);
        title.setEllipsize(TextUtils.TruncateAt.END);
        head.addView(title, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));

        head.addView(statusPill(tunnel.optString("status")));

        ImageView more = new ImageView(requireContext());
        more.setImageResource(R.drawable.ic_nav_more);
        more.setColorFilter(p.mute);
        more.setClickable(true);
        more.setBackground(UI.pressable(UI.rounded(p.canvas, UI.RADIUS_MD), p.btnGhostHover));
        int inset = UI.dp(UI.SM);
        more.setPadding(inset, inset, inset, inset);
        LinearLayout.LayoutParams moreLp = new LinearLayout.LayoutParams(UI.dp(32), UI.dp(32));
        moreLp.leftMargin = UI.dp(UI.SM);
        head.addView(more, moreLp);
        more.setOnClickListener(v -> openActions(tunnel));

        card.addView(head);

        TextView idText = UI.mono(requireContext(), id, p.mute);
        idText.setSingleLine(true);
        idText.setEllipsize(TextUtils.TruncateAt.MIDDLE);
        UI.margin(idText, 0, UI.SM, 0, UI.MD);
        card.addView(idText);

        card.addView(UI.divider(requireContext()));

        LinearLayout foot = UI.row(requireContext());
        foot.setGravity(Gravity.CENTER_VERTICAL);
        if (current) {
            // Tag left, ghost action right: a weighted button would centre its
            // own label in the gap instead of hugging the edge.
            foot.addView(UI.tag(requireContext(), "当前使用", true));
            View gap = new View(requireContext());
            UI.weight(gap, 1f);
            foot.addView(gap);
            TextView clear = UI.button(requireContext(), "取消锁定", UI.BTN_GHOST);
            clear.setOnClickListener(v -> select("", ""));
            foot.addView(clear);
        } else {
            TextView pick = UI.button(requireContext(), "锁定这条隧道", UI.BTN_SECONDARY);
            pick.setOnClickListener(v -> select(id, label));
            UI.fill(pick);
            foot.addView(pick);
        }
        card.addView(foot);

        card.setClickable(true);
        card.setOnClickListener(v -> openRoute("/tunnels/" + id));
        return card;
    }

    /** Cloudflare reports healthy / degraded / down / inactive. */
    private View statusPill(String status) {
        String state = "healthy".equals(status) ? "ok" : "degraded".equals(status) ? "warn" : "down";
        return UI.statusPill(requireContext(), state);
    }

    // ------------------------------------------------------------------ actions

    private void openActions(JSONObject tunnel) {
        String id = tunnel.optString("id");
        String name = tunnel.optString("name", "");
        if (name.isEmpty()) name = "隧道";
        final String label = name;
        boolean current = id.equals(currentId);

        Sheet.Builder sheet = Sheet.of(requireContext(), name)
                .label("隧道操作")
                .action(R.drawable.ic_nav_tunnels, "查看详情", "路由规则与隧道信息",
                        () -> openRoute("/tunnels/" + id));
        if (current) {
            sheet.item(R.drawable.ic_nav_close, "取消锁定", () -> select("", ""));
        } else {
            sheet.item(R.drawable.ic_nav_check, "锁定这条隧道", () -> select(id, label));
        }
        sheet.item(R.drawable.ic_nav_copy, "复制隧道 ID", () -> copy("隧道 ID", id));
        sheet.item(R.drawable.ic_nav_trash, "删除隧道", () -> askDelete(id, label));
        sheet.show();
    }

    private void select(String id, String name) {
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("id", id);
            payload.put("name", name);
            return Api.post("/api/config/tunnel", payload);
        }, ok -> {
            currentId = id;
            currentName = name;
            toast(id.isEmpty() ? "已取消锁定" : "已锁定「" + name + "」");
            load();
        }, failure -> toast(failure.getMessage()));
    }

    private void askCreate() {
        EditText name = UI.input(requireContext(), "例如：prod-tunnel");
        name.setFilters(new InputFilter[]{new InputFilter.LengthFilter(60)});

        Modal.of(requireContext(), "新建隧道")
                .message("名称只用来在控制台里辨认这条隧道；创建后 Cloudflare 会给出连接令牌。")
                .content(name)
                .cancel("取消")
                .confirm("创建", false, () -> create(name.getText().toString().trim()))
                .show();
    }

    private void create(String name) {
        if (name.isEmpty()) {
            toast("请填写隧道名称");
            return;
        }
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("name", name);
            return Api.post("/api/tunnels", payload);
        }, created -> {
            load();
            showToken(created);
        }, failure -> toast(failure.getMessage()));
    }

    /** The token is shown once and never again, so it gets a sheet of its own. */
    private void showToken(JSONObject created) {
        Context ctx = requireContext();
        Palette p = Theme.p();
        LinearLayout box = UI.column(ctx);

        TextView hint = UI.muted(ctx, "把下面的命令粘到已安装 cloudflared 的机器上执行，隧道就会连上 Cloudflare。");
        UI.margin(hint, 0, 0, 0, UI.MD);
        box.addView(hint);

        String command = created.optString("run_command", "");
        if (command.isEmpty()) command = created.optString("token", "");
        final String payload = command;

        TextView code = UI.mono(ctx, command, p.ink);
        code.setTextSize(TypedValue.COMPLEX_UNIT_SP, 12);
        code.setTextIsSelectable(true);
        code.setPadding(UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD));
        code.setBackground(UI.roundedStroke(p.canvasSoft, UI.RADIUS_MD, p.hairline, 1));
        box.addView(code);

        TextView copy = UI.button(ctx, "复制运行命令", UI.BTN_SECONDARY);
        UI.fill(copy);
        UI.margin(copy, 0, UI.MD, 0, 0);
        copy.setOnClickListener(v -> copy("隧道运行命令", payload));
        box.addView(copy);

        if (!created.isNull("warning") && !created.optString("warning", "").isEmpty()) {
            TextView warning = UI.banner(ctx, created.optString("warning"), true);
            UI.margin(warning, 0, UI.MD, 0, 0);
            box.addView(warning);
        }
        Sheet.of(requireContext(), "隧道创建成功").content(box).show();
    }

    private void askDelete(String id, String name) {
        Modal.of(requireContext(), "删除隧道")
                .message("确定删除「" + name + "」吗？此操作无法撤销，隧道仍有活动连接时 Cloudflare 会拒绝删除。")
                .cancel("取消")
                .confirm("确认删除", true, () -> Api.async(
                        () -> Api.delete("/api/tunnels/" + id),
                        ok -> {
                            toast("隧道已删除");
                            load();
                        },
                        failure -> toast(failure.getMessage())))
                .show();
    }

    private void copy(String label, String text) {
        if (text.isEmpty()) return;
        ClipboardManager clipboard = (ClipboardManager) requireContext()
                .getSystemService(Context.CLIPBOARD_SERVICE);
        if (clipboard == null) return;
        clipboard.setPrimaryClip(ClipData.newPlainText(label, text));
        // Android 13 draws its own "copied" toast; adding one doubles it up.
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) toast("已复制");
    }

    private void toast(String message) {
        if (message == null || message.isEmpty() || !alive()) return;
        android.widget.Toast.makeText(requireContext(), message, android.widget.Toast.LENGTH_LONG).show();
    }
}
