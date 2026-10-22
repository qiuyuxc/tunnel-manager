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

    private static final String[] TYPES = DNSValues.SIMPLE_TYPES;
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

        body.addView(UI.pageTitle(requireContext(), I18n.text("m_474fb6002594")));
        TextView subtitle = UI.muted(requireContext(), I18n.text("m_7157e9363786"));
        UI.margin(subtitle, 0, UI.XS, 0, UI.MD);
        body.addView(subtitle);

        LinearLayout actions = UI.row(requireContext());
        TextView add = UI.button(requireContext(), I18n.text("m_42845d86b969"), UI.BTN_PRIMARY);
        add.setEnabled(!zoneId.isEmpty());
        add.setOnClickListener(v -> openEditor(null));
        UI.weight(add, 1f);
        actions.addView(add);
        TextView bulk = UI.button(requireContext(), I18n.text("m_8cb8203cbf30"), UI.BTN_SECONDARY);
        bulk.setEnabled(!zoneId.isEmpty());
        bulk.setOnClickListener(view -> openBatchCreate());
        UI.weight(bulk, 1f);
        UI.margin(bulk, UI.SM, 0, 0, 0);
        actions.addView(bulk);
        TextView reload = UI.button(requireContext(), I18n.text("m_aee887434131"), UI.BTN_SECONDARY);
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

        card.addView(UI.label(requireContext(), I18n.text("m_3382870bc4ad")));
        TextView pick = UI.button(requireContext(),
                zoneId.isEmpty() ? I18n.text("m_8b34c0fd32be") : zoneName, UI.BTN_SECONDARY);
        pick.setSingleLine(true);
        pick.setEllipsize(TextUtils.TruncateAt.MIDDLE);
        UI.fill(pick);
        UI.margin(pick, 0, UI.XS, 0, UI.MD);
        pick.setOnClickListener(v -> openZonePicker());
        card.addView(pick);

        card.addView(UI.label(requireContext(), I18n.text("m_72c1c6e22586")));
        EditText query = UI.input(requireContext(), I18n.text("m_c7fb3c34cfcb"));
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

        TextView count = UI.strong(requireContext(), I18n.text("m_52c963128eab") + selected.size() + I18n.text("m_6fb2840d568a"));
        bar.addView(count, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));

        TextView edit = UI.button(requireContext(), I18n.text("m_8f98ede913e1"), UI.BTN_SECONDARY);
        edit.setOnClickListener(v -> openBatchEditor());
        bar.addView(edit);

        TextView remove = UI.button(requireContext(), I18n.text("m_ddae7a0fc554"), UI.BTN_DANGER);
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
            list.addView(UI.muted(requireContext(), I18n.text("m_9cc04745d685")));
            return list;
        }
        if (!loadError.isEmpty()) {
            list.addView(UI.banner(requireContext(), loadError, true));
            return list;
        }
        if (zoneId.isEmpty()) {
            list.addView(emptyCard(I18n.text("m_341902462c1f"), I18n.text("m_e712c11d0c29")));
            return list;
        }

        List<JSONObject> visible = filtered();
        if (visible.isEmpty()) {
            list.addView(emptyCard(search.isEmpty() ? I18n.text("m_e87dcce7a9db") : I18n.text("m_e455f075d925"),
                    search.isEmpty() ? I18n.text("m_52112034c082") : I18n.text("m_55ace8c95341")));
            return list;
        }

        TextView count = UI.muted(requireContext(), I18n.text("m_2166772cf952") + visible.size() + I18n.text("m_53aa7f3019bb")
                + (search.isEmpty() ? "" : I18n.text("m_5fc0078b78d8") + records.length() + I18n.text("m_6fb2840d568a"))
                + (selected.isEmpty() ? "" : I18n.text("m_c0d45ffee400") + selected.size() + I18n.text("m_6fb2840d568a")));
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
        dot.setEnabled(DNSJson.editable(record));
        dot.setAlpha(DNSJson.editable(record) ? 1f : 0.35f);
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

        if (DNSJson.editable(record)) {
            head.addView(UI.iconButton(requireContext(), R.drawable.ic_nav_edit, p.body, v -> openEditor(record)));
            head.addView(UI.iconButton(requireContext(), R.drawable.ic_nav_trash, p.error, v -> confirmDelete(record)));
        } else {
            head.addView(UI.muted(requireContext(), I18n.text("m_3b5ec3533b0e")));
        }
        card.addView(head);

        String content = record.optString("content");
        if (content.isEmpty() && record.has("data")) content = record.optString("data");
        if ("MX".equals(type)) content = I18n.text("m_810f949e9ec1") + record.optInt("priority") + " · " + content;
        TextView value = UI.mono(requireContext(), content, p.body);
        value.setTextSize(android.util.TypedValue.COMPLEX_UNIT_SP, 12);
        value.setPadding(UI.dp(20 + UI.MD), UI.dp(UI.SM), 0, 0);
        card.addView(value);

        StringBuilder meta = new StringBuilder();
        if (proxyEligible(type)) {
            meta.append(record.optBoolean("proxied") ? I18n.text("m_5c7bbf98286f") : I18n.text("m_80848ec9f9a4"));
        } else {
            meta.append(I18n.text("m_0b389df53cd0"));
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
        if (ttl == 1) return I18n.text("m_7eb336e42cb5");
        if (ttl < 3600) return (ttl / 60) + I18n.text("m_40de54344d22");
        if (ttl < 86400) return (ttl / 3600) + I18n.text("m_8f82ece04f3a");
        return (ttl / 86400) + I18n.text("m_4ce3054e7c4d");
    }

    private static String contentLabel(String type) {
        if ("MX".equals(type)) return I18n.text("m_bc41a064e66b");
        if ("TXT".equals(type)) return I18n.text("m_479bac57975e");
        return I18n.text("m_99ec095a258f");
    }

    private static String validateContent(String type, String raw) {
        return DNSValues.contentError(type, raw);
    }

    // ------------------------------------------------------------------ sheets

    private void openZonePicker() {
        Sheet.Builder sheet = Sheet.of(requireContext(), I18n.text("m_8b34c0fd32be")).label(I18n.text("m_4fab6a8be07d"));
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
            sheet.item(R.drawable.ic_nav_dns, I18n.text("m_4d069966fe94"), () -> {
            });
        }
        sheet.show();
    }

    private void openEditor(@Nullable JSONObject record) {
        if (zoneId.isEmpty() || (record != null && !DNSJson.editable(record))) return;
        final String targetZone = zoneId;
        new DNSRecordEditor(requireContext(), targetZone, record, () -> {
            if (alive() && zoneId.equals(targetZone)) loadRecords();
        }).show();
    }

    private void openBatchCreate() {
        if (zoneId.isEmpty()) return;
        final String targetZone = zoneId;
        new DNSBatchCreate(requireContext(), targetZone, zoneName, records, () -> {
            if (alive() && zoneId.equals(targetZone)) loadRecords();
        }).show();
    }

    private static String placeholderFor(String type) {
        switch (type) {
            case "AAAA":
                return I18n.text("m_7fcbf1d0e8d2");
            case "NS":
            case "PTR":
            case "CNAME":
                return I18n.text("m_01c2dcecc710");
            case "TXT":
                return I18n.text("m_aa51c4d276e4");
            case "MX":
                return I18n.text("m_7b17349f7925");
            default:
                return I18n.text("m_1e661077b17d");
        }
    }

    // ------------------------------------------------------------------ batch

    private void openBatchEditor() {
        final int[] typeIndex = {0};
        final int[] ttl = {-1};
        /** 0 keeps the current state, 1 forces proxied, 2 forces DNS only. */
        final int[] proxyChoice = {0};

        LinearLayout form = UI.column(requireContext());
        form.addView(UI.muted(requireContext(), I18n.text("m_3b2ee82dd97c") + selected.size()
                + I18n.text("m_bfa6a79d97ea")));

        final String[] typeLabels = new String[TYPES.length + 1];
        typeLabels[0] = I18n.text("m_dc919123840f");
        System.arraycopy(TYPES, 0, typeLabels, 1, TYPES.length);
        final TextView typeButton = UI.button(requireContext(), typeLabels[0], UI.BTN_SECONDARY);
        typeButton.setSingleLine(true);
        typeButton.setEllipsize(TextUtils.TruncateAt.END);
        UI.fill(typeButton);

        final LinearLayout contentField = UI.column(requireContext());
        final EditText content = UI.input(requireContext(), I18n.text("m_5632a6a4e542"));

        final TextView ttlButton = UI.button(requireContext(), I18n.text("m_b531bbc08116"), UI.BTN_SECONDARY);
        UI.fill(ttlButton);

        final LinearLayout proxyField = UI.column(requireContext());
        final TextView proxyButton = UI.button(requireContext(), I18n.text("m_b531bbc08116"), UI.BTN_SECONDARY);
        UI.fill(proxyButton);

        typeButton.setOnClickListener(v -> {
            Sheet.Builder sheet = Sheet.of(requireContext(), I18n.text("m_d45e6a43ac72")).label(I18n.text("m_8f98ede913e1"));
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
            Sheet.Builder sheet = Sheet.of(requireContext(), "TTL").label(I18n.text("m_8f98ede913e1"));
            sheet.item(R.drawable.ic_nav_check, I18n.text("m_b531bbc08116"), () -> {
                ttl[0] = -1;
                ttlButton.setText(I18n.text("m_b531bbc08116"));
            });
            for (final int value : TTLS) {
                sheet.item(R.drawable.ic_nav_check, ttlLabel(value), () -> {
                    ttl[0] = value;
                    ttlButton.setText(ttlLabel(value));
                });
            }
            sheet.show();
        });

        final String[] proxyLabels = {I18n.text("m_b531bbc08116"), I18n.text("m_5c7bbf98286f"), I18n.text("m_80848ec9f9a4")};
        proxyButton.setOnClickListener(v -> {
            Sheet.Builder sheet = Sheet.of(requireContext(), I18n.text("m_b90665e74c01")).label(I18n.text("m_8f98ede913e1"));
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
        form.addView(UI.field(requireContext(), I18n.text("m_d45e6a43ac72"), typeButton, UI.MD));
        UI.margin(contentField, 0, UI.MD, 0, 0);
        UI.fill(contentField);
        contentField.addView(UI.label(requireContext(), I18n.text("m_5632a6a4e542")));
        UI.margin(content, 0, UI.dp(6), 0, 0);
        contentField.addView(content);
        form.addView(contentField);
        form.addView(UI.field(requireContext(), "TTL", ttlButton, UI.MD));
        proxyField.addView(UI.label(requireContext(), I18n.text("m_b90665e74c01")));
        UI.margin(proxyButton, 0, UI.dp(6), 0, 0);
        proxyField.addView(proxyButton);
        UI.margin(proxyField, 0, UI.MD, 0, 0);
        UI.fill(proxyField);
        form.addView(proxyField);

        TextView apply = UI.button(requireContext(), I18n.text("m_d6c0b9b6c6fb") + selected.size() + I18n.text("m_53aa7f3019bb"), UI.BTN_PRIMARY);
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

        Sheet.of(requireContext(), I18n.text("m_542fce600639")).content(form).show();
    }

    /** The console PUTs each selected record in turn; so does this. */
    private void applyBatch(String type, String content, int ttl, int proxyChoice) {
        List<JSONObject> targets = new ArrayList<>();
        for (int i = 0; i < records.length(); i++) {
            JSONObject record = records.optJSONObject(i);
            if (record != null && selected.contains(record.optString("id"))) targets.add(record);
        }
        for (JSONObject record : targets) {
            if (!DNSJson.editable(record) || (!type.isEmpty() && DNSValues.structured(record.optString("type")))) {
                toast(I18n.text("m_98c1648c9086"));
                return;
            }
        }
        statusMessage = "";
        final String targetZone = zoneId;
        Api.async(() -> {
            int done = 0;
            for (JSONObject record : targets) {
                JSONObject payload = new JSONObject();
                String recordType = type.isEmpty() ? record.optString("type") : type;
                payload.put("type", recordType);
                payload.put("name", record.optString("name"));
                if (DNSValues.structured(recordType)) payload.put("data", record.getJSONObject("data"));
                else payload.put("content", type.isEmpty() ? record.optString("content") : content);
                payload.put("ttl", ttl == -1 ? record.optInt("ttl") : ttl);
                if (proxyEligible(recordType)) {
                    if (proxyChoice == 0) {
                        payload.put("proxied", record.optBoolean("proxied"));
                    } else {
                        payload.put("proxied", proxyChoice == 1);
                    }
                }
                if ("MX".equals(recordType)) payload.put("priority", record.optInt("priority"));
                Api.put("/api/zones/" + targetZone + "/dns-records/" + record.optString("id"), payload);
                done++;
            }
            return done;
        }, count -> {
            toast(I18n.text("m_2e770b687629") + count + I18n.text("m_53aa7f3019bb"));
            loadRecords();
        }, failure -> toast(failure.getMessage()));
    }

    private void confirmBatchDelete() {
        Modal.of(requireContext(), I18n.text("m_6f0b4dde3678"))
                .message(I18n.text("m_f065dedfd77a") + selected.size() + I18n.text("m_1009f8ca3b6c"))
                .cancel(I18n.text("m_2cd0f3be8738"))
                .confirm(I18n.text("m_a3ea3c17b401"), true, this::runBatchDelete)
                .show();
    }

    private void runBatchDelete() {
        List<String> ids = new ArrayList<>(selected);
        statusMessage = I18n.text("m_911f3aea5f86") + ids.size() + " …";
        render();
        Api.async(() -> {
            int done = 0;
            for (String id : ids) {
                Api.delete("/api/zones/" + zoneId + "/dns-records/" + id);
                done++;
            }
            return done;
        }, count -> {
            statusMessage = I18n.text("m_4f616c9f960e") + count + I18n.text("m_53aa7f3019bb");
            toast(statusMessage);
            loadRecords();
        }, failure -> {
            statusMessage = "";
            toast(failure.getMessage());
            loadRecords();
        });
    }

    private void confirmDelete(JSONObject record) {
        if (!DNSJson.editable(record)) return;
        final String targetZone = zoneId;
        Modal.of(requireContext(), I18n.text("m_ab190f1faa64"))
                .message(I18n.text("m_6fa91af954cd") + record.optString("type") + I18n.text("m_5cd9d0528965") + record.optString("name")
                        + I18n.text("m_a79a0def3e5e"))
                .cancel(I18n.text("m_2cd0f3be8738"))
                .confirm(I18n.text("m_a3ea3c17b401"), true, () -> Api.async(
                        () -> Api.delete("/api/zones/" + targetZone + "/dns-records/" + record.optString("id")),
                        ok -> {
                            toast(I18n.text("m_36bdc2fdb515"));
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
