package com.tunnelmanager.app;

import android.graphics.Typeface;
import android.os.Bundle;
import android.text.InputType;
import android.view.Gravity;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.EditText;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout;

import org.json.JSONArray;
import org.json.JSONObject;

/**
 * 监控详情 — one project's targets and its detection settings.
 *
 * Everything here is a call the web console already makes, in the same order:
 * the page loads the monitor, edits go out one field at a time, and every
 * mutation ends by re-reading the monitor so the screen never guesses what the
 * server did.
 */
public class MonitorDetailFragment extends PageFragment {

    private static final String ARG_ID = "id";

    static MonitorDetailFragment forId(String id) {
        MonitorDetailFragment fragment = new MonitorDetailFragment();
        Bundle args = new Bundle();
        args.putString(ARG_ID, id);
        fragment.setArguments(args);
        return fragment;
    }

    private String monitorId = "";
    private SwipeRefreshLayout refresh;
    private LinearLayout body;
    /** HTTP / TCP / ICMP and GET / POST, held until the form is submitted. */
    private int newTargetType = 0;
    private int newTargetMethod = 0;

    @Override
    public void onCreate(@Nullable Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        if (getArguments() != null) monitorId = getArguments().getString(ARG_ID, "");
    }

    @Override
    String route() {
        return "/monitors/" + monitorId;
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
        Api.async(() -> Api.get("/api/monitors/" + monitorId),
                monitor -> {
                    refresh.setRefreshing(false);
                    render(monitor);
                },
                failure -> {
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

    private void render(JSONObject monitor) {
        if (body == null || !alive()) return;
        body.removeAllViews();

        String name = monitor.optString("name", "监控项目");
        body.addView(UI.pageTitle(requireContext(), name));

        TextView subtitle = UI.muted(requireContext(), "每 " + monitor.optInt("interval_sec") + "s 检测 · "
                + (monitor.optBoolean("publish_enabled") ? "公开页已开启" : "未公开"));
        UI.margin(subtitle, 0, UI.XS, 0, UI.MD);
        body.addView(subtitle);

        LinearLayout actions = UI.row(requireContext());
        actions.addView(actionButton("立即检测", UI.BTN_PRIMARY, v -> checkNow()));
        actions.addView(actionButton(monitor.optBoolean("publish_enabled") ? "取消公开" : "开启公开",
                UI.BTN_SECONDARY, v -> setPublish(!monitor.optBoolean("publish_enabled"))));
        actions.addView(actionButton("删除", UI.BTN_DANGER, v -> askDelete(name)));
        body.addView(actions);

        body.addView(UI.spacer(requireContext(), UI.LG));
        body.addView(targetsCard(monitor));
        body.addView(UI.spacer(requireContext(), UI.LG));
        body.addView(addTargetCard());
        body.addView(UI.spacer(requireContext(), UI.LG));
        body.addView(intervalCard(monitor));

        if (monitor.optBoolean("publish_enabled")) {
            String url = Session.server() + "/status/" + monitor.optString("public_token");
            body.addView(UI.spacer(requireContext(), UI.LG));
            body.addView(publicCard(url));
        }
    }

    private TextView actionButton(String label, int kind, View.OnClickListener onClick) {
        TextView button = UI.button(requireContext(), label, kind);
        button.setOnClickListener(onClick);
        LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);
        lp.rightMargin = UI.dp(UI.SM);
        button.setLayoutParams(lp);
        return button;
    }

    private View targetsCard(JSONObject monitor) {
        Palette p = Theme.p();
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), "服务"));

        JSONArray targets = monitor.optJSONArray("targets");
        if (targets == null || targets.length() == 0) {
            TextView empty = UI.muted(requireContext(), "还没有服务，在下面添加一个要探测的地址");
            UI.margin(empty, 0, UI.SM, 0, 0);
            card.addView(empty);
            return card;
        }
        for (int i = 0; i < targets.length(); i++) {
            JSONObject target = targets.optJSONObject(i);
            if (i > 0) card.addView(UI.divider(requireContext()));
            card.addView(targetRow(target));
        }
        return card;
    }

    private View targetRow(JSONObject target) {
        Palette p = Theme.p();
        LinearLayout row = UI.row(requireContext());
        row.setGravity(Gravity.TOP);

        LinearLayout.LayoutParams pillLp = new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);
        pillLp.topMargin = UI.dp(1);
        row.addView(UI.statusPill(requireContext(), target.optString("state", "")), pillLp);

        LinearLayout text = UI.column(requireContext());
        LinearLayout.LayoutParams textLp = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
        textLp.leftMargin = UI.dp(UI.SM);
        textLp.rightMargin = UI.dp(UI.SM);
        row.addView(text, textLp);

        LinearLayout title = UI.row(requireContext());
        title.addView(UI.text(requireContext(), target.optString("name", ""), 14, p.ink, Typeface.NORMAL),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        long latency = target.optLong("latency_ms");
        if (latency > 0) title.addView(UI.mono(requireContext(), latency + " ms", p.mute));
        text.addView(title);

        TextView url = UI.mono(requireContext(), target.optString("url", ""), p.mute);
        url.setMaxLines(2);
        url.setEllipsize(android.text.TextUtils.TruncateAt.END);
        text.addView(url);

        String error = target.optString("error", "");
        if (!error.isEmpty()) {
            TextView line = UI.text(requireContext(), error, 12, p.error, Typeface.NORMAL);
            line.setMaxLines(2);
            line.setEllipsize(android.text.TextUtils.TruncateAt.END);
            text.addView(line);
        }
        text.addView(UI.text(requireContext(), "24 小时可用率 "
                + String.format(java.util.Locale.US, "%.2f%%", target.optDouble("uptime_24h")),
                11, p.mute, Typeface.NORMAL));

        row.setClickable(true);
        row.setOnClickListener(v -> editTarget(target));
        return row;
    }

    private View addTargetCard() {
        Palette p = Theme.p();
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), "添加服务"));

        EditText name = UI.input(requireContext(), "名称，例如：官网");
        card.addView(UI.field(requireContext(), "名称", name));

        EditText url = UI.input(requireContext(), "https://example.com");
        url.setInputType(InputType.TYPE_TEXT_VARIATION_URI);
        card.addView(UI.field(requireContext(), "地址", url, UI.MD));

        LinearLayout type = UI.segmented(requireContext(),
                new String[]{"HTTP", "TCP", "ICMP"}, newTargetType, value -> newTargetType = value);
        card.addView(UI.field(requireContext(), "探测方式", type, UI.MD));

        LinearLayout method = UI.segmented(requireContext(),
                new String[]{"GET", "POST"}, newTargetMethod, value -> newTargetMethod = value);
        card.addView(UI.field(requireContext(), "请求方法", method, UI.MD));

        TextView add = UI.button(requireContext(), "添加服务", UI.BTN_PRIMARY);
        UI.fill(add);
        UI.margin(add, 0, UI.MD, 0, 0);
        add.setOnClickListener(v -> addTarget(name.getText().toString().trim(), url.getText().toString().trim()));
        card.addView(add);
        return card;
    }

    private View intervalCard(JSONObject monitor) {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), "检测设置"));

        EditText interval = UI.input(requireContext(), "60");
        interval.setInputType(InputType.TYPE_CLASS_NUMBER);
        interval.setText(String.valueOf(monitor.optInt("interval_sec")));
        card.addView(UI.field(requireContext(), "检测间隔（秒）", interval, UI.MD));

        TextView save = UI.button(requireContext(), "保存设置", UI.BTN_SECONDARY);
        UI.fill(save);
        UI.margin(save, 0, UI.MD, 0, 0);
        save.setOnClickListener(v -> setInterval(interval.getText().toString().trim()));
        card.addView(save);
        return card;
    }

    private View publicCard(String url) {
        Palette p = Theme.p();
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), "公开状态页"));
        TextView link = UI.mono(requireContext(), url, p.link);
        link.setMaxLines(2);
        UI.margin(link, 0, UI.SM, 0, UI.MD);
        card.addView(link);

        TextView copy = UI.button(requireContext(), "复制链接", UI.BTN_SECONDARY);
        copy.setOnClickListener(v -> {
            android.content.ClipboardManager clipboard =
                    (android.content.ClipboardManager) requireContext().getSystemService(android.content.Context.CLIPBOARD_SERVICE);
            if (clipboard != null) {
                clipboard.setPrimaryClip(android.content.ClipData.newPlainText("status url", url));
            }
            toast("已复制");
        });
        card.addView(copy);
        return card;
    }

    // ------------------------------------------------------------------ actions

    private void checkNow() {
        Api.async(() -> Api.post("/api/monitors/" + monitorId + "/check", new JSONObject()),
                result -> {
                    JSONArray outcomes = result.optJSONArray("outcomes");
                    int ok = 0;
                    if (outcomes != null) {
                        for (int i = 0; i < outcomes.length(); i++) {
                            if ("ok".equals(outcomes.optJSONObject(i).optString("state"))) ok++;
                        }
                    }
                    toast("检测完成：" + (outcomes == null ? 0 : outcomes.length()) + " 个服务，正常 " + ok);
                    load();
                },
                failure -> toast(failure.getMessage()));
    }

    private void setPublish(boolean enabled) {
        patch("公开设置已更新", () -> {
            JSONObject payload = new JSONObject();
            payload.put("publish_enabled", enabled);
            return payload;
        });
    }

    private void setInterval(String raw) {
        int seconds;
        try {
            seconds = Integer.parseInt(raw);
        } catch (NumberFormatException e) {
            toast("检测间隔要填数字");
            return;
        }
        patch("设置已保存", () -> {
            JSONObject payload = new JSONObject();
            payload.put("interval_sec", seconds);
            return payload;
        });
    }

    /** One field at a time over PUT, the same shape the settings form sends. */
    private void patch(String success, java.util.concurrent.Callable<JSONObject> payload) {
        Api.async(() -> Api.put("/api/monitors/" + monitorId, payload.call()),
                updated -> {
                    toast(success);
                    render(updated);
                },
                failure -> toast(failure.getMessage()));
    }

    private void addTarget(String name, String url) {
        if (name.isEmpty() || url.isEmpty()) {
            toast("名称和地址都要填");
            return;
        }
        String[] types = {"http", "tcp", "icmp"};
        String[] methods = {"GET", "POST"};
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("name", name);
            payload.put("url", url);
            payload.put("type", types[newTargetType]);
            if (newTargetType == 0) payload.put("method", methods[newTargetMethod]);
            return Api.post("/api/monitors/" + monitorId + "/targets", payload);
        }, updated -> {
            toast("已添加");
            render(updated);
        }, failure -> toast(failure.getMessage()));
    }

    /** Tap a target: rename it, retarget it, or drop it. */
    private void editTarget(JSONObject target) {
        String targetId = target.optString("id");
        EditText name = UI.input(requireContext(), "名称");
        name.setText(target.optString("name", ""));

        EditText url = UI.input(requireContext(), "地址");
        url.setInputType(InputType.TYPE_TEXT_VARIATION_URI);
        url.setText(target.optString("url", ""));

        LinearLayout form = UI.column(requireContext());
        form.addView(UI.field(requireContext(), "名称", name));
        form.addView(UI.field(requireContext(), "地址", url, UI.MD));

        TextView save = UI.button(requireContext(), "保存修改", UI.BTN_PRIMARY);
        UI.fill(save);
        UI.margin(save, 0, UI.MD, 0, 0);
        save.setOnClickListener(v -> {
            String nextName = name.getText().toString().trim();
            String nextUrl = url.getText().toString().trim();
            if (nextName.isEmpty() || nextUrl.isEmpty()) {
                toast("名称和地址都要填");
                return;
            }
            Api.async(() -> {
                JSONObject payload = new JSONObject();
                payload.put("name", nextName);
                payload.put("url", nextUrl);
                return Api.put("/api/monitors/" + monitorId + "/targets/" + targetId, payload);
            }, updated -> render(updated), failure -> toast(failure.getMessage()));
        });
        form.addView(save);

        TextView remove = UI.button(requireContext(), "移除这个服务", UI.BTN_DANGER);
        UI.fill(remove);
        UI.margin(remove, 0, UI.SM, 0, 0);
        form.addView(remove);

        Sheet.Builder builder = Sheet.of(requireContext(), target.optString("name", "服务"));
        remove.setOnClickListener(v -> {
            Api.async(() -> Api.delete("/api/monitors/" + monitorId + "/targets/" + targetId),
                    updated -> render(updated), failure -> toast(failure.getMessage()));
            Sheet.dismissVisible();
        });
        builder.content(form).show();
    }

    private void askDelete(String name) {
        Modal.of(requireContext(), "删除监控项目")
                .message("将删除「" + name + "」及其全部检测历史，公开链接同步失效。")
                .cancel("取消")
                .confirm("确认删除", true, () -> Api.async(
                        () -> Api.delete("/api/monitors/" + monitorId),
                        ok -> openRoute("/monitors"),
                        failure -> toast(failure.getMessage())))
                .show();
    }

    private void toast(String message) {
        if (message == null || message.isEmpty() || !alive()) return;
        android.widget.Toast.makeText(requireContext(), message, android.widget.Toast.LENGTH_LONG).show();
    }
}
