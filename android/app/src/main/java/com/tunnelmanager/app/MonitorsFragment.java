package com.tunnelmanager.app;

import android.graphics.Typeface;
import android.os.Bundle;
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
 * 服务监控 — the list of monitor projects.
 *
 * One request, one card per project, and the two things you actually do from
 * here: create one, or open one. Editing and probing live in the detail page,
 * which is where the same split sits on the web console.
 */
public class MonitorsFragment extends PageFragment {

    private SwipeRefreshLayout refresh;
    private LinearLayout body;

    @Override
    String route() {
        return "/monitors";
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

        renderHeader(null);
        load();
        return refresh;
    }

    private void load() {
        Api.async(() -> Api.getArray("/api/monitors"),
                monitors -> {
                    refresh.setRefreshing(false);
                    renderHeader(monitors);
                },
                failure -> {
                    refresh.setRefreshing(false);
                    renderHeader(null);
                });
    }

    // --------------------------------------------------------------- rendering

    private void renderHeader(@Nullable JSONArray monitors) {
        if (body == null || !alive()) return;
        body.removeAllViews();

        LinearLayout head = UI.row(requireContext());
        LinearLayout titles = UI.column(requireContext());
        titles.addView(UI.pageTitle(requireContext(), "服务监控"));

        TextView subtitle = UI.muted(requireContext(), "创建监控项目，持续探测服务可用性");
        UI.margin(subtitle, 0, UI.XS, 0, 0);
        titles.addView(subtitle);
        head.addView(titles, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));

        TextView create = UI.button(requireContext(), "创建监控", UI.BTN_PRIMARY);
        create.setOnClickListener(v -> askCreate());
        head.addView(create);
        body.addView(head);
        body.addView(UI.spacer(requireContext(), UI.LG));

        if (monitors == null) {
            body.addView(UI.muted(requireContext(), "加载失败，下拉重试"));
            return;
        }
        if (monitors.length() == 0) {
            body.addView(emptyCard());
            return;
        }
        for (int i = 0; i < monitors.length(); i++) {
            if (i > 0) body.addView(UI.spacer(requireContext(), UI.SM));
            body.addView(monitorCard(monitors.optJSONObject(i)));
        }
    }

    private View emptyCard() {
        Palette p = Theme.p();
        LinearLayout card = UI.card(requireContext());
        card.setGravity(Gravity.CENTER_HORIZONTAL);

        ImageView icon = new ImageView(requireContext());
        icon.setImageResource(R.drawable.ic_nav_monitor);
        icon.setColorFilter(p.mute);
        card.addView(icon, new LinearLayout.LayoutParams(UI.dp(30), UI.dp(30)));

        TextView title = UI.strong(requireContext(), "还没有监控项目");
        UI.margin(title, 0, UI.MD, 0, 0);
        card.addView(title);

        TextView copy = UI.muted(requireContext(), "创建一个监控项目，添加需要盯住的服务地址，系统会按固定间隔自动检测。");
        copy.setGravity(Gravity.CENTER);
        UI.margin(copy, 0, UI.XS, 0, UI.MD);
        card.addView(copy);

        TextView create = UI.button(requireContext(), "创建第一个监控", UI.BTN_PRIMARY);
        create.setOnClickListener(v -> askCreate());
        card.addView(create);
        return card;
    }

    private View monitorCard(JSONObject monitor) {
        Palette p = Theme.p();
        String id = monitor.optString("id");
        JSONArray targets = monitor.optJSONArray("targets");
        int count = targets == null ? 0 : targets.length();

        LinearLayout card = UI.card(requireContext());
        LinearLayout row = UI.row(requireContext());

        LinearLayout text = UI.column(requireContext());
        LinearLayout nameRow = UI.row(requireContext());
        nameRow.addView(UI.text(requireContext(), monitor.optString("name"), 15, p.ink, Typeface.BOLD));
        TextView number = UI.text(requireContext(), count + " 个服务", 11, p.mute, Typeface.NORMAL);
        UI.margin(number, UI.SM, 1, 0, 0);
        nameRow.addView(number);
        text.addView(nameRow);

        StringBuilder meta = new StringBuilder();
        meta.append("每 ").append(monitor.optInt("interval_sec")).append("s 检测 · ")
                .append(monitor.optBoolean("publish_enabled") ? "公开页已开启" : "未公开");
        TextView detail = UI.muted(requireContext(), meta.toString());
        UI.margin(detail, 0, UI.XS, 0, 0);
        text.addView(detail);

        if (count > 0) {
            LinearLayout summary = UI.row(requireContext());
            UI.margin(summary, 0, UI.SM, 0, 0);
            summary.addView(stateCount(targets, "ok"));
            summary.addView(stateCount(targets, "warn"));
            summary.addView(stateCount(targets, "down"));
            text.addView(summary);
        }
        row.addView(text, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));

        ImageView chevron = new ImageView(requireContext());
        chevron.setImageResource(R.drawable.ic_nav_chevron);
        chevron.setColorFilter(p.mute);
        row.addView(chevron, new LinearLayout.LayoutParams(UI.dp(16), UI.dp(16)));
        card.addView(row);

        card.setClickable(true);
        card.setBackground(UI.pressable(
                UI.roundedStroke(p.canvasRaised, UI.RADIUS_LG, p.hairline, 1), p.btnGhostHover));
        card.setOnClickListener(v -> openRoute("/monitors/" + id));
        // Long-press is the phone's stand-in for the web console's row menu.
        card.setOnLongClickListener(v -> {
            askDelete(id, monitor.optString("name"));
            return true;
        });
        return card;
    }

    private View stateCount(JSONArray targets, String state) {
        Palette p = Theme.p();
        int colour = "ok".equals(state) ? p.success : "warn".equals(state) ? p.warning : p.error;
        int total = 0;
        for (int i = 0; i < targets.length(); i++) {
            JSONObject target = targets.optJSONObject(i);
            if (state.equals(target.optString("state", ""))) total++;
        }

        LinearLayout item = UI.row(requireContext());
        View dot = new View(requireContext());
        dot.setBackground(UI.rounded(colour, UI.RADIUS_PILL));
        LinearLayout.LayoutParams dotLp = new LinearLayout.LayoutParams(UI.dp(7), UI.dp(7));
        dotLp.rightMargin = UI.dp(4);
        item.addView(dot, dotLp);
        item.addView(UI.text(requireContext(), String.valueOf(total), 11, p.mute, Typeface.NORMAL));
        LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);
        lp.rightMargin = UI.dp(UI.MD);
        item.setLayoutParams(lp);
        return item;
    }

    // ------------------------------------------------------------------ actions

    private void askCreate() {
        EditText name = UI.input(requireContext(), "例如：生产环境服务");
        name.setMaxLines(1);
        name.setFilters(new android.text.InputFilter[]{new android.text.InputFilter.LengthFilter(60)});

        Modal.of(requireContext(), "创建监控项目")
                .message("给这组服务起个名字，建好后可以继续添加要探测的地址。")
                .content(name)
                .cancel("取消")
                .confirm("创建", false, () -> create(name.getText().toString().trim()))
                .show();
    }

    private void create(String name) {
        if (name.isEmpty()) return;
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("name", name);
            return Api.post("/api/monitors", payload);
        }, created -> openRoute("/monitors/" + created.optString("id")),
                failure -> toast(failure.getMessage()));
    }

    private void askDelete(String id, String name) {
        Modal.of(requireContext(), "删除监控项目")
                .message("将删除「" + name + "」及其全部检测历史，公开链接同步失效。")
                .cancel("取消")
                .confirm("确认删除", true, () -> remove(id))
                .show();
    }

    private void remove(String id) {
        Api.async(() -> Api.delete("/api/monitors/" + id),
                ok -> load(),
                failure -> toast(failure.getMessage()));
    }

    private void toast(String message) {
        if (message == null || message.isEmpty() || !alive()) return;
        android.widget.Toast.makeText(requireContext(), message, android.widget.Toast.LENGTH_LONG).show();
    }

}
