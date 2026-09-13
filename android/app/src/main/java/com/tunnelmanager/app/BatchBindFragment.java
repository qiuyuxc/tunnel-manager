package com.tunnelmanager.app;

import android.graphics.Typeface;
import android.text.InputType;
import android.text.TextUtils;
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

import java.util.ArrayList;
import java.util.List;

/**
 * 批量绑定 — several domains, each with its own forwarding target.
 *
 * The web console keeps an array of groups and re-renders the whole list on
 * every change. The same list lives here, but each group also keeps the widgets
 * it is drawn with: switching a group's mode should only show or hide two
 * fields, not tear down every input on the page.
 */
public class BatchBindFragment extends PageFragment {

    /** One 绑定组: the values, its own last result, and its current widgets. */
    private static final class Group {
        int mode;
        String serviceUrl = "";
        String cname = "";
        String main = "";
        String aux = "";
        String result = "";
        boolean ok;

        EditText serviceView;
        EditText cnameView;
        EditText mainView;
        EditText auxView;
        LinearLayout cnameField;
        LinearLayout auxField;
    }

    private SwipeRefreshLayout refresh;
    private LinearLayout body;
    private JSONObject config = new JSONObject();
    private final List<Group> groups = new ArrayList<>();
    private String summary = "";

    @Override
    String route() {
        return "/domain/batch";
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
        Api.async(() -> Api.get("/api/config"), loaded -> {
            refresh.setRefreshing(false);
            config = loaded;
            if (groups.isEmpty()) groups.add(newGroup());
            render();
        }, failure -> {
            refresh.setRefreshing(false);
            renderFailure(failure.getMessage());
        });
    }

    private Group newGroup() {
        Group group = new Group();
        group.serviceUrl = config.optString("service_url", "");
        return group;
    }

    // --------------------------------------------------------------- rendering

    private void renderFailure(String message) {
        if (body == null || !alive()) return;
        body.removeAllViews();
        body.addView(UI.banner(requireContext(), message == null ? "加载失败" : message, true));
    }

    private void render() {
        if (body == null || !alive()) return;
        body.removeAllViews();

        body.addView(UI.pageTitle(requireContext(), "批量绑定"));

        TextView subtitle = UI.muted(requireContext(), "每组独立设置转发地址，按顺序绑定主域名和辅助域名");
        UI.margin(subtitle, 0, UI.XS, 0, UI.MD);
        body.addView(subtitle);

        String tunnelId = config.optString("tunnel_id", "");
        if (tunnelId.isEmpty()) {
            body.addView(UI.banner(requireContext(), "前置条件未满足：请先在「隧道管理」锁定隧道。", true));
            body.addView(UI.spacer(requireContext(), UI.LG));
        } else {
            LinearLayout contextCard = UI.card(requireContext());
            contextCard.addView(UI.label(requireContext(), "当前隧道"));
            TextView name = UI.text(requireContext(), config.optString("tunnel_name", "已选隧道"),
                    14, Theme.p().ink, Typeface.BOLD);
            UI.margin(name, 0, UI.XS, 0, 0);
            contextCard.addView(name);
            body.addView(contextCard);
            body.addView(UI.spacer(requireContext(), UI.MD));
        }

        for (int i = 0; i < groups.size(); i++) {
            if (i > 0) body.addView(UI.spacer(requireContext(), UI.MD));
            body.addView(groupCard(groups.get(i), i));
        }

        if (!summary.isEmpty()) {
            TextView banner = UI.banner(requireContext(), summary, false);
            UI.margin(banner, 0, UI.MD, 0, 0);
            body.addView(banner);
        }

        TextView submit = UI.button(requireContext(), "批量绑定", UI.BTN_PRIMARY);
        UI.fill(submit);
        UI.margin(submit, 0, UI.MD, 0, 0);
        submit.setOnClickListener(v -> submit());
        body.addView(submit);
    }

    private View groupCard(final Group group, final int index) {
        LinearLayout card = UI.card(requireContext());

        LinearLayout head = UI.row(requireContext());
        head.setGravity(Gravity.CENTER_VERTICAL);
        head.addView(UI.label(requireContext(), "绑定组 " + (index + 1)),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        head.addView(UI.iconButton(requireContext(), R.drawable.ic_nav_plus, Theme.p().body, v -> {
            capture();
            groups.add(newGroup());
            render();
        }));
        if (groups.size() > 1) {
            head.addView(UI.iconButton(requireContext(), R.drawable.ic_nav_close, Theme.p().error, v -> {
                capture();
                groups.remove(group);
                render();
            }));
        }
        card.addView(head);

        LinearLayout modes = UI.segmented(requireContext(), new String[]{"简单模式", "优选模式"}, group.mode,
                selected -> {
                    group.mode = selected;
                    group.cnameField.setVisibility(selected == 1 ? View.VISIBLE : View.GONE);
                    group.auxField.setVisibility(selected == 1 ? View.VISIBLE : View.GONE);
                });
        UI.margin(modes, 0, UI.MD, 0, 0);
        card.addView(modes);

        group.serviceView = UI.input(requireContext(), "http://localhost:3000");
        group.serviceView.setInputType(InputType.TYPE_TEXT_VARIATION_URI);
        group.serviceView.setText(group.serviceUrl);
        card.addView(UI.field(requireContext(), "转发地址", group.serviceView, UI.MD));

        group.cnameView = UI.input(requireContext(), "留空使用全局配置");
        group.cnameView.setText(group.cname);
        TextView pick = UI.button(requireContext(), "常用线路", UI.BTN_SECONDARY);
        pick.setOnClickListener(v -> openCnamePicker(group));
        LinearLayout cnameRow = UI.row(requireContext());
        cnameRow.addView(group.cnameView, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        LinearLayout.LayoutParams pickLp = new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);
        pickLp.leftMargin = UI.dp(UI.SM);
        cnameRow.addView(pick, pickLp);
        group.cnameField = UI.field(requireContext(), "优选 CNAME · 选填", cnameRow, UI.MD);
        card.addView(group.cnameField);

        group.mainView = UI.input(requireContext(), "例如：kukie.cn");
        group.mainView.setInputType(InputType.TYPE_TEXT_VARIATION_URI);
        group.mainView.setText(group.main);
        card.addView(UI.field(requireContext(), "主域名", group.mainView, UI.MD));

        group.auxView = UI.input(requireContext(), "例如：fallback.169977.xyz");
        group.auxView.setInputType(InputType.TYPE_TEXT_VARIATION_URI);
        group.auxView.setText(group.aux);
        group.auxField = UI.field(requireContext(), "辅助域名 · 用作回源", group.auxView, UI.MD);
        card.addView(group.auxField);

        group.cnameField.setVisibility(group.mode == 1 ? View.VISIBLE : View.GONE);
        group.auxField.setVisibility(group.mode == 1 ? View.VISIBLE : View.GONE);

        if (!group.result.isEmpty()) {
            TextView banner = UI.banner(requireContext(), group.result, !group.ok);
            UI.margin(banner, 0, UI.MD, 0, 0);
            card.addView(banner);
        }
        return card;
    }

    // ------------------------------------------------------------------ actions

    private void openCnamePicker(final Group group) {
        String fallback = config.optString("preferred_cname", "");
        Sheet.Builder sheet = Sheet.of(requireContext(), "优选 CNAME").label("常用线路");
        sheet.item(R.drawable.ic_nav_check,
                fallback.isEmpty() ? "使用全局配置" : "使用全局配置（" + fallback + "）",
                () -> group.cnameView.setText(""));

        JSONArray presets = config.optJSONArray("cname_presets");
        if (presets != null) {
            for (int i = 0; i < presets.length(); i++) {
                JSONObject preset = presets.optJSONObject(i);
                if (preset == null) continue;
                String value = preset.optString("value", "");
                if (value.isEmpty()) continue;
                sheet.item(R.drawable.ic_nav_domain,
                        preset.optString("name", value) + " · " + value,
                        () -> group.cnameView.setText(value));
            }
        }
        sheet.show();
    }

    /** Pulls every field back into the model before the list is rebuilt. */
    private void capture() {
        for (Group group : groups) {
            if (group.serviceView == null) continue;
            group.serviceUrl = group.serviceView.getText().toString();
            group.cname = group.cnameView.getText().toString();
            group.main = group.mainView.getText().toString();
            group.aux = group.auxView.getText().toString();
        }
    }

    private void submit() {
        capture();
        boolean valid = true;
        for (Group group : groups) {
            group.result = "";
            boolean needsAux = group.mode == 1;
            boolean complete = !group.serviceUrl.trim().isEmpty()
                    && !group.main.trim().isEmpty()
                    && (!needsAux || !group.aux.trim().isEmpty());
            if (!complete) {
                valid = false;
                group.ok = false;
                group.result = needsAux
                        ? "转发地址、主域名、辅助域名都要填"
                        : "转发地址和主域名都要填";
            }
        }
        if (!valid) {
            summary = "";
            render();
            toast("请补全每组的转发地址和域名");
            return;
        }

        final JSONArray items = new JSONArray();
        for (Group group : groups) {
            JSONObject item = new JSONObject();
            try {
                item.put("mode", group.mode == 1 ? "preferred" : "simple");
                item.put("service_url", group.serviceUrl.trim());
                item.put("preferred_cname", group.cname.trim());
                item.put("main_domain", group.main.trim());
                item.put("aux_domain", group.aux.trim());
            } catch (Exception ignored) {
            }
            items.put(item);
        }

        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("items", items);
            return Api.post("/api/domain/bind-batch", payload);
        }, response -> {
            JSONArray results = response.optJSONArray("results");
            int succeeded = 0;
            for (int i = 0; i < groups.size(); i++) {
                Group group = groups.get(i);
                JSONObject result = results == null ? null : results.optJSONObject(i);
                if (result == null) {
                    group.ok = false;
                    group.result = "未收到该组的执行结果";
                    continue;
                }
                group.ok = result.optBoolean("success");
                group.result = result.optString("message", "");
                if (group.ok) succeeded++;
            }
            summary = "批量绑定完成：" + succeeded + "/" + groups.size() + " 成功";
            render();
            toast(summary);
        }, failure -> toast(failure.getMessage()));
    }

    private void toast(String message) {
        if (message == null || message.isEmpty() || !alive()) return;
        android.widget.Toast.makeText(requireContext(), message, android.widget.Toast.LENGTH_LONG).show();
    }
}
