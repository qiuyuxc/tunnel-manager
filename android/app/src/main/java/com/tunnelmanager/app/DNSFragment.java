package com.tunnelmanager.app;

import android.graphics.Color;
import android.graphics.Typeface;
import android.text.Editable;
import android.text.InputType;
import android.text.TextUtils;
import android.text.TextWatcher;
import android.view.Gravity;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.EditText;
import android.widget.FrameLayout;
import android.widget.ImageView;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout;

import org.json.JSONArray;
import org.json.JSONObject;

import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Set;

/**
 * DNS 管理 — one zone's records.
 *
 * This is the page the phone layout fights hardest: the console renders it as a
 * seven column table with checkboxes, which on a 384dp screen scrolls sideways
 * and clips every value. Here a record is a card — type, name, value, proxy and
 * TTL stacked — and selection is a circle you tap rather than a column you drag
 * across.
 */
public class DNSFragment extends PageFragment {

    /** A / AAAA / CNAME / TXT / MX, in the order the console lists them. */
    private static final String[] TYPES = {"A", "AAAA", "CNAME", "TXT", "MX"};
    private static final int[] TTLS = {1, 60, 120, 300, 600, 1800, 3600, 7200, 18000, 43200, 86400};

    private SwipeRefreshLayout refresh;
    private LinearLayout body;
    private FrameLayout batchBarHolder;

    private JSONArray zones = new JSONArray();
    private String zoneId = "";
    private String zoneName = "";
    private JSONArray records = new JSONArray();
    private final Set<String> selected = new LinkedHashSet<>();

    private String search = "";
    private String loadError = "";
    private boolean loading = false;
    private String statusMessage = "";

    @Override
    String route() {
        return "/dns";
    }

    @Override
    protected View build(@NonNull LayoutInflater inflater, @Nullable ViewGroup container) {
        refresh = new SwipeRefreshLayout(requireContext());
        refresh.setColorSchemeColors(Theme.p().ink);
        refresh.setProgressBackgroundColorSchemeColor(Theme.p().canvasRaised);
        // Pull-to-refresh means "the record list is stale" far more often than
        // it means "the zone list is"; only go back for zones when there is
        // nothing selected to reload.
        refresh.setOnRefreshListener(() -> {
            if (zoneId.isEmpty()) {
                loadZones();
            } else {
                loadRecords();
            }
        });

        ScrollView scroll = new ScrollView(requireContext());
        body = UI.column(requireContext());
        UI.pagePadding(body);
        // Room for the selection bar that slides in over the last card.
        boolean wide = getResources().getConfiguration().screenWidthDp >= UI.WIDE_DP;
        body.setPadding(body.getPaddingLeft(), body.getPaddingTop(), body.getPaddingRight(), UI.dp(wide ? 96 : UI.TABBAR_H + 120));
        scroll.addView(body);
        refresh.addView(scroll);

        batchBarHolder = new FrameLayout(requireContext());
        FrameLayout.LayoutParams barLp = new FrameLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT);
        barLp.gravity = Gravity.BOTTOM;
        barLp.bottomMargin = wide ? 0 : UI.dp(UI.TABBAR_H + 24);

        FrameLayout root = new FrameLayout(requireContext());
        root.addView(refresh, new FrameLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT));
        root.addView(batchBarHolder, barLp);

        render();
        loadZones();
        return root;
    }

    private void loadZones() {
        Api.async(() -> sortByName(Api.getArray("/api/zones")), loaded -> {
            refresh.setRefreshing(false);
            zones = loaded;
            // The console lands on the first zone rather than an empty table,
            // so the page does too — the picker is one tap away either way.
            if (zoneId.isEmpty() && zones.length() > 0) {
                JSONObject zone = zones.optJSONObject(0);
                if (zone != null) {
                    zoneId = zone.optString("id");
                    zoneName = zone.optString("name");
                    loadRecords();
                    return;
                }
            }
            render();
        }, failure -> {
            refresh.setRefreshing(false);
            loadError = failure.getMessage();
            render();
        });
    }

    /** Cloudflare hands the zones back unsorted; the picker reads better sorted. */
    private static JSONArray sortByName(JSONArray input) {
        List<JSONObject> sorted = new ArrayList<>();
        for (int i = 0; i < input.length(); i++) {
            JSONObject zone = input.optJSONObject(i);
            if (zone != null) sorted.add(zone);
        }
        sorted.sort((a, b) -> a.optString("name").compareToIgnoreCase(b.optString("name")));
        JSONArray out = new JSONArray();
        for (JSONObject zone : sorted) out.put(zone);
        return out;
    }

    private void loadRecords() {
        if (zoneId.isEmpty()) return;
        loading = true;
        loadError = "";
        render();
        Api.async(() -> Api.getArray("/api/zones/" + zoneId + "/dns-records"), loaded -> {
            refresh.setRefreshing(false);
            loading = false;
            records = loaded;
            selected.clear();
            render();
        }, failure -> {
            refresh.setRefreshing(false);
            loading = false;
            loadError = failure.getMessage();
            render();
        });
    }

    // --------------------------------------------------------------- rendering

    private void render() {
        if (body == null || !alive()) return;
        body.removeAllViews();

        body.addView(UI.pageTitle(requireContext(), "DNS 管理"));
        TextView subtitle = UI.muted(requireContext(), "集中查看和维护 Cloudflare 区域中的 DNS 记录");
        UI.margin(subtitle, 0, UI.XS, 0, UI.MD);
        body.addView(subtitle);

        LinearLayout actions = UI.row(requireContext());
        TextView add = UI.button(requireContext(), "添加记录", UI.BTN_PRIMARY);
        add.setOnClickListener(v -> openEditor(null));
        UI.weight(add, 1f);
        actions.addView(add);
        TextView reload = UI.button(requireContext(), "刷新", UI.BTN_SECONDARY);
        reload.setOnClickListener(v -> loadRecords());
        UI.weight(reload, 1f);
        UI.margin(reload, UI.SM, 0, 0, 0);
        actions.addView(reload);
        body.addView(actions);

        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(zoneCard());

        if (!statusMessage.isEmpty()) {
            TextView banner = UI.banner(requireContext(), statusMessage, false);
            UI.margin(banner, 0, UI.MD, 0, 0);
            body.addView(banner);
        }

        body.addView(UI.spacer(requireContext(), UI.MD));
        renderList();
        renderBatchBar();
    }

    private View zoneCard() {
        LinearLayout card = UI.card(requireContext());

        card.addView(UI.label(requireContext(), "区域"));
        TextView pick = UI.button(requireContext(),
                zoneId.isEmpty() ? "选择区域" : zoneName, UI.BTN_SECONDARY);
        pick.setSingleLine(true);
        pick.setEllipsize(TextUtils.TruncateAt.MIDDLE);
        UI.fill(pick);
        UI.margin(pick, 0, UI.XS, 0, UI.MD);
        pick.setOnClickListener(v -> openZonePicker());
        card.addView(pick);

        card.addView(UI.label(requireContext(), "搜索记录"));
        EditText query = UI.input(requireContext(), "名称、内容或类型");
        query.setText(search);
        query.setEnabled(!zoneId.isEmpty());
        query.addTextChangedListener(new TextWatcher() {
            @Override
            public void beforeTextChanged(CharSequence s, int a, int b, int c) {
            }

            @Override
            public void onTextChanged(CharSequence s, int a, int b, int c) {
            }

            @Override
            public void afterTextChanged(Editable s) {
                search = s.toString().trim().toLowerCase();
                renderList();
            }
        });
        UI.margin(query, 0, UI.XS, 0, 0);
        card.addView(query);
        return card;
    }

    /** Only the list is redrawn when the query changes, so the keyboard stays up. */
    private void renderList() {
        if (body == null || !alive()) return;
        View old = body.findViewWithTag("dns-list");
        if (old != null) body.removeView(old);
        body.addView(listViews());
    }

    /** The bar itself; {@link #renderBatchBar} decides whether it is on screen. */
    private View batchBar() {
        LinearLayout bar = UI.row(requireContext());
        bar.setGravity(Gravity.CENTER_VERTICAL);

        TextView count = UI.strong(requireContext(), "已选 " + selected.size() + " 条");
        bar.addView(count, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));

        TextView edit = UI.button(requireContext(), "批量修改", UI.BTN_SECONDARY);
        edit.setOnClickListener(v -> openBatchEditor());
        bar.addView(edit);

        TextView remove = UI.button(requireContext(), "批量删除", UI.BTN_DANGER);
        remove.setOnClickListener(v -> confirmBatchDelete());
        UI.margin(remove, UI.SM, 0, 0, 0);
        bar.addView(remove);
        return bar;
    }

    private void renderBatchBar() {
        if (batchBarHolder == null) return;
        batchBarHolder.removeAllViews();
        if (selected.isEmpty()) return;

        LinearLayout panel = UI.column(requireContext());
        panel.setBackgroundColor(Theme.p().canvas);
        View seam = new View(requireContext());
        seam.setBackgroundColor(Theme.p().hairline);
        panel.addView(seam, new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, Math.max(1, UI.dp(1))));
        panel.setPadding(UI.dp(UI.LG), UI.dp(UI.MD), UI.dp(UI.LG), UI.dp(UI.LG));
        panel.addView(batchBar());
        batchBarHolder.addView(panel, new FrameLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));
    }

    private View listViews() {
        LinearLayout list = UI.column(requireContext());
        list.setTag("dns-list");

        if (loading) {
            list.addView(UI.muted(requireContext(), "正在读取 DNS 记录…"));
            return list;
        }
        if (!loadError.isEmpty()) {
            list.addView(UI.banner(requireContext(), loadError, true));
            return list;
        }
        if (zoneId.isEmpty()) {
            list.addView(emptyCard("先选择一个区域", "选择后即可查看和管理该区域的记录。"));
            return list;
        }

        List<JSONObject> visible = filtered();
        if (visible.isEmpty()) {
            list.addView(emptyCard(search.isEmpty() ? "尚无 DNS 记录" : "没有匹配的记录",
                    search.isEmpty() ? "添加第一条记录以开始解析流量。" : "尝试更换搜索关键词。"));
            return list;
        }

        TextView count = UI.muted(requireContext(), "显示 " + visible.size() + " 条记录"
                + (search.isEmpty() ? "" : "，共 " + records.length() + " 条")
                + (selected.isEmpty() ? "" : "，已选 " + selected.size() + " 条"));
        UI.margin(count, 0, 0, 0, UI.SM);
        list.addView(count);

        for (JSONObject record : visible) {
            list.addView(recordCard(record));
            list.addView(UI.spacer(requireContext(), UI.SM));
        }
        return list;
    }

    private List<JSONObject> filtered() {
        List<JSONObject> out = new ArrayList<>();
        for (int i = 0; i < records.length(); i++) {
            JSONObject record = records.optJSONObject(i);
            if (record == null) continue;
            if (search.isEmpty()
                    || record.optString("name").toLowerCase().contains(search)
                    || record.optString("content").toLowerCase().contains(search)
                    || record.optString("type").toLowerCase().contains(search)) {
                out.add(record);
            }
        }
        return out;
    }

    private View emptyCard(String title, String detail) {
        Palette p = Theme.p();
        LinearLayout card = UI.card(requireContext());
        card.setGravity(Gravity.CENTER_HORIZONTAL);
        ImageView icon = new ImageView(requireContext());
        icon.setImageResource(R.drawable.ic_nav_dns);
        UI.tint(icon, p.mute);
        card.addView(icon, new LinearLayout.LayoutParams(UI.dp(28), UI.dp(28)));
        TextView heading = UI.strong(requireContext(), title);
        UI.margin(heading, 0, UI.SM, 0, 0);
        card.addView(heading);
        TextView caption = UI.muted(requireContext(), detail);
        caption.setGravity(Gravity.CENTER);
        UI.margin(caption, 0, UI.XS, 0, 0);
        card.addView(caption);
        return card;
    }

    private View recordCard(JSONObject record) {
        Palette p = Theme.p();
        String id = record.optString("id");
        String type = record.optString("type");
        final boolean isSelected = selected.contains(id);

        LinearLayout card = UI.card(requireContext());
        LinearLayout head = UI.row(requireContext());
        head.setGravity(Gravity.CENTER_VERTICAL);

        FrameLayout dot = new FrameLayout(requireContext());
        dot.setBackground(UI.circle(isSelected ? p.ink : Color.TRANSPARENT,
                isSelected ? 0 : p.hairlineStrong, 1.5f));
        if (isSelected) {
            ImageView check = new ImageView(requireContext());
            check.setImageResource(R.drawable.ic_nav_check);
            UI.tint(check, p.btnPrimaryText);
            dot.addView(check, new FrameLayout.LayoutParams(UI.dp(12), UI.dp(12), Gravity.CENTER));
        }
        dot.setClickable(true);
        dot.setOnClickListener(v -> toggle(id));
        LinearLayout.LayoutParams dotLp = new LinearLayout.LayoutParams(UI.dp(20), UI.dp(20));
        dotLp.rightMargin = UI.dp(UI.MD);
        head.addView(dot, dotLp);

        TextView typeTag = UI.text(requireContext(), type, 11, p.ink, Typeface.BOLD);
        typeTag.setGravity(Gravity.CENTER);
        typeTag.setTypeface(Typeface.MONOSPACE, Typeface.BOLD);
        typeTag.setPadding(UI.dp(UI.SM), UI.dp(2), UI.dp(UI.SM), UI.dp(2));
        typeTag.setBackground(UI.roundedStroke(p.canvasSoft2, UI.RADIUS_MD, p.hairline, 1));
        LinearLayout.LayoutParams tagLp = new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);
        tagLp.rightMargin = UI.dp(UI.SM);
        head.addView(typeTag, tagLp);

        TextView name = UI.mono(requireContext(), record.optString("name"), p.ink);
        name.setSingleLine(true);
        name.setEllipsize(TextUtils.TruncateAt.MIDDLE);
        head.addView(name, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));

        head.addView(UI.iconButton(requireContext(), R.drawable.ic_nav_edit, p.body, v -> openEditor(record)));
        head.addView(UI.iconButton(requireContext(), R.drawable.ic_nav_trash, p.error,
                v -> confirmDelete(record)));
        card.addView(head);

        String content = record.optString("content");
        if ("MX".equals(type)) content = "优先级 " + record.optInt("priority") + " · " + content;
        TextView value = UI.mono(requireContext(), content, p.body);
        value.setTextSize(android.util.TypedValue.COMPLEX_UNIT_SP, 12);
        value.setPadding(UI.dp(20 + UI.MD), UI.dp(UI.SM), 0, 0);
        card.addView(value);

        StringBuilder meta = new StringBuilder();
        if (proxyEligible(type)) {
            meta.append(record.optBoolean("proxied") ? "已代理" : "仅 DNS");
        } else {
            meta.append("代理不适用");
        }
        meta.append(" · TTL ").append(ttlLabel(record.optInt("ttl")));
        TextView caption = UI.muted(requireContext(), meta.toString());
        caption.setPadding(UI.dp(20 + UI.MD), UI.dp(UI.XS), 0, 0);
        card.addView(caption);
        return card;
    }

    private void toggle(String id) {
        if (!selected.remove(id)) selected.add(id);
        render();
    }

    // -------------------------------------------------------------- selection

    private static boolean proxyEligible(String type) {
        return "A".equals(type) || "AAAA".equals(type) || "CNAME".equals(type);
    }

    private static String ttlLabel(int ttl) {
        if (ttl == 1) return "自动";
        if (ttl < 3600) return (ttl / 60) + " 分钟";
        if (ttl < 86400) return (ttl / 3600) + " 小时";
        return (ttl / 86400) + " 天";
    }

    private static String contentLabel(String type) {
        if ("MX".equals(type)) return "邮件服务器";
        if ("TXT".equals(type)) return "文本内容";
        return "目标内容";
    }

    /** Mirrors frontend/src/utils/dnsValidation.ts, message for message. */
    private static String validateContent(String type, String raw) {
        String content = raw.trim();
        if (content.isEmpty()) return "解析值不能为空";
        switch (type) {
            case "A":
                return isIPv4(content) ? null : "A 记录必须填写有效的 IPv4 地址";
            case "AAAA":
                return isIPv6(content) ? null : "AAAA 记录必须填写有效的 IPv6 地址";
            case "CNAME":
                return isHostname(content) ? null : "CNAME 记录必须填写域名目标，不能填写 IP 地址";
            case "MX":
                return isHostname(content) ? null : "MX 记录必须填写邮件服务器域名";
            default:
                return null;
        }
    }

    private static boolean isIPv4(String value) {
        String[] parts = value.split("\\.", -1);
        if (parts.length != 4) return false;
        for (String part : parts) {
            if (part.isEmpty() || part.length() > 3) return false;
            for (int i = 0; i < part.length(); i++) {
                if (!Character.isDigit(part.charAt(i))) return false;
            }
            if (Integer.parseInt(part) > 255) return false;
        }
        return true;
    }

    private static boolean isIPv6(String value) {
        if (!value.contains(":")) return false;
        for (int i = 0; i < value.length(); i++) {
            char c = value.charAt(i);
            if (Character.digit(c, 16) < 0 && c != ':' && c != '.') return false;
        }
        // One "::" at most; a trailing single colon is never valid.
        int doubles = value.split("::", -1).length - 1;
        return doubles <= 1 && !value.endsWith(":");
    }

    private static boolean isHostname(String value) {
        if (value.isEmpty() || value.length() > 253) return false;
        if (isIPv4(value) || value.contains(":")) return false;
        String trimmed = value.endsWith(".") ? value.substring(0, value.length() - 1) : value;
        if (trimmed.isEmpty()) return false;
        for (String label : trimmed.split("\\.", -1)) {
            if (label.isEmpty() || label.length() > 63) return false;
            for (int i = 0; i < label.length(); i++) {
                char c = label.charAt(i);
                boolean ok = Character.isLetterOrDigit(c) || c == '-' || c == '_' || c > 127;
                if (!ok) return false;
            }
            if (label.charAt(0) == '-' || label.charAt(label.length() - 1) == '-') return false;
        }
        return true;
    }

    // ------------------------------------------------------------------ sheets

    private void openZonePicker() {
        Sheet.Builder sheet = Sheet.of(requireContext(), "选择区域").label("Cloudflare 区域");
        for (int i = 0; i < zones.length(); i++) {
            JSONObject zone = zones.optJSONObject(i);
            if (zone == null) continue;
            String id = zone.optString("id");
            String name = zone.optString("name");
            sheet.item(R.drawable.ic_nav_check, name, () -> {
                zoneId = id;
                zoneName = name;
                selected.clear();
                statusMessage = "";
                loadRecords();
            });
        }
        if (zones.length() == 0) {
            sheet.item(R.drawable.ic_nav_dns, "没有可用的区域", () -> {
            });
        }
        sheet.show();
    }

    /**
     * Add or edit one record.
     *
     * {@code record} null means 添加; the sheet holds every field, and the ones
     * that do not apply to the chosen type are hidden rather than disabled.
     */
    private void openEditor(@Nullable JSONObject record) {
        final boolean editing = record != null;
        final String[] type = {editing ? record.optString("type") : "A"};
        final int[] ttl = {editing ? record.optInt("ttl") : 1};
        final boolean[] proxied = {editing && record.optBoolean("proxied")};

        LinearLayout form = UI.column(requireContext());

        final LinearLayout contentField = UI.column(requireContext());
        final EditText content = UI.textArea(requireContext(), "解析值");
        content.setMinLines(2);
        content.setHint(placeholderFor(type[0]));

        final LinearLayout priorityField = UI.field(requireContext(), "MX 优先级",
                UI.input(requireContext(), "0 - 65535"), UI.MD);
        EditText priority = (EditText) priorityField.getChildAt(1);
        priority.setInputType(InputType.TYPE_CLASS_NUMBER);
        priority.setText(String.valueOf(editing ? record.optInt("priority") : 0));

        final TextView ttlButton = UI.button(requireContext(), ttlLabel(ttl[0]), UI.BTN_SECONDARY);
        UI.fill(ttlButton);
        ttlButton.setOnClickListener(v -> {
            String[] labels = new String[TTLS.length];
            for (int i = 0; i < TTLS.length; i++) labels[i] = ttlLabel(TTLS[i]);
            Sheet.Builder sheet = Sheet.of(requireContext(), "TTL").label("缓存时间");
            for (int i = 0; i < TTLS.length; i++) {
                final int value = TTLS[i];
                sheet.item(R.drawable.ic_nav_check, labels[i], () -> {
                    ttl[0] = value;
                    ttlButton.setText(ttlLabel(value));
                });
            }
            sheet.show();
        });

        final LinearLayout proxyRow = UI.row(requireContext());
        proxyRow.setGravity(Gravity.CENTER_VERTICAL);
        proxyRow.addView(UI.strong(requireContext(), "代理状态"),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        UI.Toggle proxyToggle = new UI.Toggle(requireContext(), proxied[0]);
        LinearLayout.LayoutParams toggleLp = new LinearLayout.LayoutParams(UI.dp(34), UI.dp(20));
        toggleLp.leftMargin = UI.dp(UI.MD);
        proxyRow.addView(proxyToggle, toggleLp);
        proxyRow.setClickable(true);
        proxyRow.setOnClickListener(v -> {
            proxied[0] = !proxyToggle.isOn();
            proxyToggle.setOn(proxied[0]);
        });

        final EditText name = UI.input(requireContext(), "例如 www 或 example.com");
        if (editing) name.setText(record.optString("name"));
        if (editing) content.setText(record.optString("content", ""));

        final Runnable[] sync = new Runnable[1];
        LinearLayout modes = UI.segmented(requireContext(), TYPES, indexOf(type[0]), index -> {
            type[0] = TYPES[index];
            sync[0].run();
        });

        sync[0] = () -> {
            content.setHint(placeholderFor(type[0]));
            contentField.removeAllViews();
            contentField.addView(UI.label(requireContext(), contentLabel(type[0])));
            UI.margin(content, 0, UI.dp(6), 0, 0);
            contentField.addView(content);
            priorityField.setVisibility("MX".equals(type[0]) ? View.VISIBLE : View.GONE);
            proxyRow.setVisibility(proxyEligible(type[0]) ? View.VISIBLE : View.GONE);
        };

        form.addView(modes);
        form.addView(UI.field(requireContext(), "名称", name, UI.MD));
        UI.margin(contentField, 0, UI.MD, 0, 0);
        UI.fill(contentField);
        form.addView(contentField);
        form.addView(priorityField);
        form.addView(UI.field(requireContext(), "TTL", ttlButton, UI.MD));
        UI.margin(proxyRow, 0, UI.MD, 0, 0);
        form.addView(proxyRow);
        sync[0].run();

        TextView save = UI.button(requireContext(), editing ? "保存更改" : "添加记录", UI.BTN_PRIMARY);
        UI.fill(save);
        UI.margin(save, 0, UI.LG, 0, 0);
        save.setOnClickListener(v -> {
            String recordName = name.getText().toString().trim();
            String value = content.getText().toString().trim();
            String problem = recordName.isEmpty() ? "请输入记录名称"
                    : validateContent(type[0], value);
            if (problem == null && "MX".equals(type[0]))
                problem = priority.getText().toString().trim().isEmpty() ? "请输入 MX 优先级" : null;
            if (problem != null) {
                toast(problem);
                return;
            }
            int priorityValue = 0;
            try {
                priorityValue = Integer.parseInt(priority.getText().toString().trim());
            } catch (NumberFormatException ignored) {
            }
            saveRecord(editing ? record : null, type[0], recordName, value, ttl[0], proxied[0], priorityValue);
        });
        form.addView(save);

        Sheet.of(requireContext(), editing ? "编辑 DNS 记录" : "添加 DNS 记录").content(form).show();
    }

    private static int indexOf(String type) {
        for (int i = 0; i < TYPES.length; i++) {
            if (TYPES[i].equals(type)) return i;
        }
        return 0;
    }

    private static String placeholderFor(String type) {
        switch (type) {
            case "AAAA":
                return "例如 2001:db8::1";
            case "CNAME":
                return "例如 target.example.com";
            case "TXT":
                return "输入 TXT 记录内容";
            case "MX":
                return "例如 mail.example.com";
            default:
                return "例如 192.0.2.1";
        }
    }

    private void saveRecord(@Nullable JSONObject record, String type, String name, String content,
                            int ttl, boolean proxied, int priority) {
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("type", type);
            payload.put("name", name);
            payload.put("content", content);
            payload.put("ttl", ttl);
            if (proxyEligible(type)) payload.put("proxied", proxied);
            if ("MX".equals(type)) payload.put("priority", priority);
            if (record != null) {
                return Api.put("/api/zones/" + zoneId + "/dns-records/" + record.optString("id"), payload);
            }
            return Api.post("/api/zones/" + zoneId + "/dns-records", payload);
        }, ok -> {
            Sheet.dismissVisible();
            toast(record != null ? "记录已更新" : "记录已添加");
            loadRecords();
        }, failure -> toast(failure.getMessage()));
    }

    // ------------------------------------------------------------------ batch

    private void openBatchEditor() {
        final int[] typeIndex = {0};
        final int[] ttl = {-1};
        /** 0 keeps the current state, 1 forces proxied, 2 forces DNS only. */
        final int[] proxyChoice = {0};

        LinearLayout form = UI.column(requireContext());
        form.addView(UI.muted(requireContext(), "已选择 " + selected.size()
                + " 条记录。留空或选择「不修改」的字段保持原值。"));

        final String[] typeLabels = new String[TYPES.length + 1];
        typeLabels[0] = "保持原类型";
        System.arraycopy(TYPES, 0, typeLabels, 1, TYPES.length);
        final TextView typeButton = UI.button(requireContext(), typeLabels[0], UI.BTN_SECONDARY);
        typeButton.setSingleLine(true);
        typeButton.setEllipsize(TextUtils.TruncateAt.END);
        UI.fill(typeButton);

        final LinearLayout contentField = UI.column(requireContext());
        final EditText content = UI.input(requireContext(), "解析值");

        final TextView ttlButton = UI.button(requireContext(), "不修改", UI.BTN_SECONDARY);
        UI.fill(ttlButton);

        final LinearLayout proxyField = UI.column(requireContext());
        final TextView proxyButton = UI.button(requireContext(), "不修改", UI.BTN_SECONDARY);
        UI.fill(proxyButton);

        typeButton.setOnClickListener(v -> {
            Sheet.Builder sheet = Sheet.of(requireContext(), "解析类型").label("批量修改");
            for (int i = 0; i < typeLabels.length; i++) {
                final int index = i;
                sheet.item(R.drawable.ic_nav_check, typeLabels[i], () -> {
                    typeIndex[0] = index;
                    typeButton.setText(typeLabels[index]);
                    String chosen = index == 0 ? "" : TYPES[index - 1];
                    content.setHint(contentLabel(chosen.isEmpty() ? "A" : chosen));
                    contentField.setVisibility(index == 0 ? View.GONE : View.VISIBLE);
                    proxyField.setVisibility(index == 0 || proxyEligible(chosen) ? View.VISIBLE : View.GONE);
                });
            }
            sheet.show();
        });

        ttlButton.setOnClickListener(v -> {
            Sheet.Builder sheet = Sheet.of(requireContext(), "TTL").label("批量修改");
            sheet.item(R.drawable.ic_nav_check, "不修改", () -> {
                ttl[0] = -1;
                ttlButton.setText("不修改");
            });
            for (final int value : TTLS) {
                sheet.item(R.drawable.ic_nav_check, ttlLabel(value), () -> {
                    ttl[0] = value;
                    ttlButton.setText(ttlLabel(value));
                });
            }
            sheet.show();
        });

        final String[] proxyLabels = {"不修改", "已代理", "仅 DNS"};
        proxyButton.setOnClickListener(v -> {
            Sheet.Builder sheet = Sheet.of(requireContext(), "代理状态").label("批量修改");
            for (int i = 0; i < proxyLabels.length; i++) {
                final int index = i;
                sheet.item(R.drawable.ic_nav_check, proxyLabels[i], () -> {
                    proxyChoice[0] = index;
                    proxyButton.setText(proxyLabels[index]);
                });
            }
            sheet.show();
        });

        contentField.setVisibility(View.GONE);
        form.addView(UI.field(requireContext(), "解析类型", typeButton, UI.MD));
        UI.margin(contentField, 0, UI.MD, 0, 0);
        UI.fill(contentField);
        contentField.addView(UI.label(requireContext(), "解析值"));
        UI.margin(content, 0, UI.dp(6), 0, 0);
        contentField.addView(content);
        form.addView(contentField);
        form.addView(UI.field(requireContext(), "TTL", ttlButton, UI.MD));
        proxyField.addView(UI.label(requireContext(), "代理状态"));
        UI.margin(proxyButton, 0, UI.dp(6), 0, 0);
        proxyField.addView(proxyButton);
        UI.margin(proxyField, 0, UI.MD, 0, 0);
        UI.fill(proxyField);
        form.addView(proxyField);

        TextView apply = UI.button(requireContext(), "应用到 " + selected.size() + " 条记录", UI.BTN_PRIMARY);
        UI.fill(apply);
        UI.margin(apply, 0, UI.LG, 0, 0);
        apply.setOnClickListener(v -> {
            String chosen = typeIndex[0] == 0 ? "" : TYPES[typeIndex[0] - 1];
            String value = content.getText().toString().trim();
            if (!chosen.isEmpty() && validateContent(chosen, value) != null) {
                toast(validateContent(chosen, value));
                return;
            }
            Sheet.dismissVisible();
            applyBatch(chosen, value, ttl[0], proxyChoice[0]);
        });
        form.addView(apply);

        Sheet.of(requireContext(), "批量修改 DNS 记录").content(form).show();
    }

    /** The console PUTs each selected record in turn; so does this. */
    private void applyBatch(String type, String content, int ttl, int proxyChoice) {
        List<JSONObject> targets = new ArrayList<>();
        for (int i = 0; i < records.length(); i++) {
            JSONObject record = records.optJSONObject(i);
            if (record != null && selected.contains(record.optString("id"))) targets.add(record);
        }
        statusMessage = "";
        Api.async(() -> {
            int done = 0;
            for (JSONObject record : targets) {
                JSONObject payload = new JSONObject();
                String recordType = type.isEmpty() ? record.optString("type") : type;
                payload.put("type", recordType);
                payload.put("name", record.optString("name"));
                payload.put("content", type.isEmpty() ? record.optString("content") : content);
                payload.put("ttl", ttl == -1 ? record.optInt("ttl") : ttl);
                if (proxyEligible(recordType)) {
                    if (proxyChoice == 0) {
                        payload.put("proxied", record.optBoolean("proxied"));
                    } else {
                        payload.put("proxied", proxyChoice == 1);
                    }
                }
                if ("MX".equals(recordType)) payload.put("priority", record.optInt("priority"));
                Api.put("/api/zones/" + zoneId + "/dns-records/" + record.optString("id"), payload);
                done++;
            }
            return done;
        }, count -> {
            toast("已更新 " + count + " 条记录");
            loadRecords();
        }, failure -> toast(failure.getMessage()));
    }

    private void confirmBatchDelete() {
        Modal.of(requireContext(), "批量删除 DNS 记录")
                .message("将删除已选的 " + selected.size() + " 条记录，此操作无法撤销。")
                .cancel("取消")
                .confirm("确认删除", true, this::runBatchDelete)
                .show();
    }

    private void runBatchDelete() {
        List<String> ids = new ArrayList<>(selected);
        statusMessage = "正在删除 0/" + ids.size() + " …";
        render();
        Api.async(() -> {
            int done = 0;
            for (String id : ids) {
                Api.delete("/api/zones/" + zoneId + "/dns-records/" + id);
                done++;
            }
            return done;
        }, count -> {
            statusMessage = "已删除 " + count + " 条记录";
            toast(statusMessage);
            loadRecords();
        }, failure -> {
            statusMessage = "";
            toast(failure.getMessage());
            loadRecords();
        });
    }

    private void confirmDelete(JSONObject record) {
        Modal.of(requireContext(), "删除 DNS 记录")
                .message("确定删除 " + record.optString("type") + " 记录 " + record.optString("name")
                        + " 吗？此操作无法撤销。")
                .cancel("取消")
                .confirm("确认删除", true, () -> Api.async(
                        () -> Api.delete("/api/zones/" + zoneId + "/dns-records/" + record.optString("id")),
                        ok -> {
                            toast("记录已删除");
                            loadRecords();
                        },
                        failure -> toast(failure.getMessage())))
                .show();
    }

    private void toast(String message) {
        if (message == null || message.isEmpty() || !alive()) return;
        android.widget.Toast.makeText(requireContext(), message, android.widget.Toast.LENGTH_LONG).show();
    }
}
