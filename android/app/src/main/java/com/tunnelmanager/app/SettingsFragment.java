package com.tunnelmanager.app;

import android.content.Intent;
import android.graphics.Bitmap;
import android.graphics.BitmapFactory;
import android.graphics.Typeface;
import android.net.Uri;
import android.os.Handler;
import android.os.Looper;
import android.text.InputType;
import android.util.Base64;
import android.view.Gravity;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.EditText;
import android.widget.ImageView;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;

import androidx.activity.result.ActivityResultLauncher;
import androidx.activity.result.contract.ActivityResultContracts;
import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import org.json.JSONArray;
import org.json.JSONObject;

import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.util.ArrayList;
import java.util.List;

/**
 * 全局设置 — branding, CNAME presets, fallback origin and the current tunnel.
 *
 * Admin-only on the server, so the page assumes the caller can save everything
 * on it. The site icon follows the web's trick of keeping the image as a data
 * URL in the field rather than uploading it: one fewer endpoint, and the value
 * the operator sees is the value that gets stored.
 */
public class SettingsFragment extends PageFragment {

    private static final long MAX_ICON_BYTES = 512L * 1024;
    private static final int MAX_PRESETS = 20;

    private ScrollView scroll;
    private LinearLayout body;

    private JSONObject config = new JSONObject();
    private Bitmap iconBitmap;
    private String iconBitmapKey = "";
    /** Side of the preview icon; a touch larger than the brand mark it mirrors. */
    private static final int PREVIEW_ICON_DP = 40;
    private final Handler ui = new Handler(Looper.getMainLooper());
    /** Debounces icon reloads so typing a URL does not fire a request per key. */
    private final Runnable iconReload = this::loadIcon;

    private String nameDraft = "";
    private String descriptionDraft = "";
    private String iconDraft = "";
    private String panelHostDraft = "";
    private boolean landingEnabled = false;

    /** Pairs of {name, value}; edited in place and saved as one list. */
    private final List<String[]> presets = new ArrayList<>();
    private String preferredDraft = "";
    private String fallbackDraft = "";

    private EditText nameInput;
    private EditText descriptionInput;
    private EditText iconInput;
    private EditText panelHostInput;
    private EditText preferredInput;
    private EditText fallbackInput;
    private final List<EditText> presetNameInputs = new ArrayList<>();
    private final List<EditText> presetValueInputs = new ArrayList<>();

    private boolean loading = true;
    private boolean busy = false;
    private String statusMessage = "";
    private boolean statusOk = true;

    private final ActivityResultLauncher<String> iconPicker =
            registerForActivityResult(new ActivityResultContracts.GetContent(), this::onIconPicked);

    @Override
    String route() {
        return "/settings";
    }

    @Override
    protected View build(@NonNull LayoutInflater inflater, @Nullable ViewGroup container) {
        scroll = new ScrollView(requireContext());
        body = UI.column(requireContext());
        UI.pagePadding(body);
        scroll.addView(body);
        render();
        load();
        return scroll;
    }

    // ------------------------------------------------------------------- data

    private void load() {
        loading = true;
        render();
        Api.async(() -> Api.get("/api/config"), payload -> {
            loading = false;
            applyConfig(payload);
            statusMessage = "";
            render();
            loadIcon();
        }, failure -> {
            loading = false;
            statusMessage = "加载失败：" + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    private void applyConfig(JSONObject payload) {
        config = payload;
        nameDraft = payload.optString("site_name", "");
        descriptionDraft = payload.optString("site_description", "");
        iconDraft = payload.optString("site_icon", "");
        panelHostDraft = payload.optString("panel_host", "");
        landingEnabled = payload.optBoolean("landing_enabled", false);
        preferredDraft = payload.optString("preferred_cname", "");

        presets.clear();
        JSONArray list = payload.optJSONArray("cname_presets");
        if (list != null) {
            for (int i = 0; i < list.length(); i++) {
                JSONObject item = list.optJSONObject(i);
                if (item == null) continue;
                presets.add(new String[]{item.optString("name", ""), item.optString("value", "")});
            }
        }
        if (presets.isEmpty()) presets.add(new String[]{"", ""});
    }

    private void loadIcon() {
        final String key = iconDraft;
        if (key.isEmpty()) {
            boolean had = iconBitmap != null || !iconBitmapKey.isEmpty();
            iconBitmap = null;
            iconBitmapKey = "";
            if (had) refreshPreview();
            return;
        }
        if (key.equals(iconBitmapKey)) return;
        if (key.startsWith("data:image")) {
            try {
                int comma = key.indexOf(',');
                byte[] bytes = Base64.decode(key.substring(comma + 1), Base64.DEFAULT);
                iconBitmap = BitmapFactory.decodeByteArray(bytes, 0, bytes.length);
                iconBitmapKey = key;
                refreshPreview();
            } catch (Exception ignored) {
            }
            return;
        }
        Api.async(() -> Api.asset(key), bytes -> {
            if (!key.equals(iconDraft)) return;
            Bitmap bitmap = BitmapFactory.decodeByteArray(bytes, 0, bytes.length);
            if (bitmap == null) return;
            iconBitmap = bitmap;
            iconBitmapKey = key;
            refreshPreview();
        }, failure -> ignore());
    }

    private void ignore() {
        // A missing icon falls back to the drawn placeholder.
    }

    // -------------------------------------------------------------- rendering

    private void capture() {
        if (nameInput != null) nameDraft = nameInput.getText().toString();
        if (descriptionInput != null) descriptionDraft = descriptionInput.getText().toString();
        if (iconInput != null) iconDraft = iconInput.getText().toString().trim();
        if (panelHostInput != null) panelHostDraft = panelHostInput.getText().toString();
        if (preferredInput != null) preferredDraft = preferredInput.getText().toString();
        if (fallbackInput != null) fallbackDraft = fallbackInput.getText().toString();
        for (int i = 0; i < presets.size(); i++) {
            if (i < presetNameInputs.size()) presets.get(i)[0] = presetNameInputs.get(i).getText().toString();
            if (i < presetValueInputs.size()) presets.get(i)[1] = presetValueInputs.get(i).getText().toString();
        }
    }

    private void render() {
        if (body == null || !alive()) return;
        int keepScroll = scroll.getScrollY();
        body.removeAllViews();

        body.addView(UI.pageTitle(requireContext(), "全局设置"));
        TextView subtitle = UI.muted(requireContext(), "集中管理站点品牌、域名绑定偏好、回退源与隧道设置");
        UI.margin(subtitle, 0, UI.XS, 0, UI.MD);
        body.addView(subtitle);

        if (loading) {
            body.addView(UI.muted(requireContext(), "加载中…"));
            return;
        }

        presetNameInputs.clear();
        presetValueInputs.clear();

        body.addView(siteCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(presetsCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(fallbackCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(tunnelCard());

        if (!statusMessage.isEmpty()) {
            TextView banner = UI.banner(requireContext(), statusMessage, !statusOk);
            UI.margin(banner, 0, UI.MD, 0, 0);
            body.addView(banner);
        }
        UI.restoreScroll(scroll, keepScroll);
    }

    private LinearLayout cardHead(String title, String desc) {
        LinearLayout head = UI.column(requireContext());
        head.addView(UI.cardTitle(requireContext(), title));
        if (desc != null && !desc.isEmpty()) {
            TextView hint = UI.muted(requireContext(), desc);
            UI.margin(hint, 0, UI.XS, 0, 0);
            head.addView(hint);
        }
        UI.margin(head, 0, 0, 0, UI.MD);
        return head;
    }

    private void note(LinearLayout card, String text, int topDp) {
        TextView note = UI.muted(requireContext(), text);
        UI.margin(note, 0, topDp, 0, 0);
        card.addView(note);
    }

    private TextView primaryAction(String label, Runnable action) {
        TextView button = UI.button(requireContext(), busy ? "处理中…" : label, UI.BTN_PRIMARY);
        button.setEnabled(!busy);
        button.setOnClickListener(v -> action.run());
        return button;
    }

    // ------------------------------------------------------------------ cards

    private View siteCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(cardHead("站点信息", "自定义浏览器标题、导航品牌、登录页名称、描述与站点图标。"));

        TextView previewLabel = UI.label(requireContext(), "实时预览");
        UI.margin(previewLabel, 0, 0, 0, UI.SM);
        card.addView(previewLabel);

        LinearLayout preview = UI.row(requireContext());
        preview.setBackground(UI.roundedStroke(Theme.p().canvasSoft2, UI.RADIUS_MD, Theme.p().hairline, 1));
        preview.setPadding(UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD));
        preview.addView(iconPreview(), new LinearLayout.LayoutParams(
                UI.dp(PREVIEW_ICON_DP), UI.dp(PREVIEW_ICON_DP)));
        LinearLayout previewText = UI.column(requireContext());
        previewText.addView(UI.strong(requireContext(),
                nameDraft.trim().isEmpty() ? "Tunnel Manager" : nameDraft.trim()));
        TextView previewDesc = UI.muted(requireContext(),
                descriptionDraft.trim().isEmpty() ? "Cloudflare 隧道管理中心" : descriptionDraft.trim());
        UI.margin(previewDesc, 0, 1, 0, 0);
        previewText.addView(previewDesc);
        LinearLayout.LayoutParams previewTextLp = new LinearLayout.LayoutParams(
                0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
        previewTextLp.leftMargin = UI.dp(UI.MD);
        preview.addView(previewText, previewTextLp);
        UI.addRow(card, preview, 0);

        nameInput = UI.input(requireContext(), "Tunnel Manager");
        nameInput.setText(nameDraft);
        nameInput.addTextChangedListener(new SimpleWatcher(this::refreshPreview));
        card.addView(UI.field(requireContext(), "站点名称", nameInput, UI.MD));

        descriptionInput = UI.input(requireContext(), "Cloudflare 隧道管理中心");
        descriptionInput.setText(descriptionDraft);
        descriptionInput.addTextChangedListener(new SimpleWatcher(this::refreshPreview));
        card.addView(UI.field(requireContext(), "站点描述", descriptionInput, UI.MD));

        iconInput = UI.input(requireContext(), "https://example.com/icon.png，或上传本地图片");
        iconInput.setText(iconDraft);
        iconInput.addTextChangedListener(new SimpleWatcher(() -> {
            iconDraft = iconInput.getText().toString().trim();
            ui.removeCallbacks(iconReload);
            ui.postDelayed(iconReload, 450);
        }));
        card.addView(UI.field(requireContext(), "站点图标", iconInput, UI.MD));

        LinearLayout iconActions = UI.row(requireContext());
        TextView upload = UI.button(requireContext(), "上传图片", UI.BTN_SECONDARY);
        upload.setOnClickListener(v -> iconPicker.launch("image/*"));
        iconActions.addView(upload);
        if (!iconDraft.isEmpty()) {
            TextView clear = UI.button(requireContext(), "清除图标", UI.BTN_GHOST);
            clear.setOnClickListener(v -> {
                capture();
                iconDraft = "";
                iconBitmap = null;
                iconBitmapKey = "";
                render();
            });
            UI.margin(clear, UI.SM, 0, 0, 0);
            iconActions.addView(clear);
        }
        UI.addRow(card, iconActions, UI.SM);
        note(card, "建议使用 1:1 图片，文件不超过 512 KB。", UI.SM);

        panelHostInput = UI.input(requireContext(), "panel.example.com");
        panelHostInput.setText(panelHostDraft);
        card.addView(UI.field(requireContext(), "面板域名", panelHostInput, UI.MD));
        note(card, "状态页自定义域名会复制该域名 ingress 的服务与源站参数。留空时自动使用管理员首次登录时的访问域名。", UI.SM);

        UI.Toggle landing = new UI.Toggle(requireContext(), landingEnabled);
        LinearLayout landingRow = UI.row(requireContext());
        LinearLayout landingText = UI.column(requireContext());
        landingText.addView(UI.strong(requireContext(), "启用首页（落地页）"));
        TextView landingHint = UI.muted(requireContext(),
                "开启后，未登录用户访问站点将先看到首页，而不是直接进入登录页。");
        UI.margin(landingHint, 0, UI.XS, 0, 0);
        landingText.addView(landingHint);
        landingRow.addView(landingText, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        LinearLayout.LayoutParams toggleLp = new LinearLayout.LayoutParams(UI.dp(34), UI.dp(20));
        toggleLp.leftMargin = UI.dp(UI.MD);
        landingRow.addView(landing, toggleLp);
        landingRow.setClickable(true);
        landingRow.setOnClickListener(v -> {
            landingEnabled = !landing.isOn();
            landing.setOn(landingEnabled);
        });
        UI.addRow(card, landingRow, UI.MD);

        card.addView(UI.spacer(requireContext(), UI.MD));
        card.addView(primaryAction("保存站点信息", this::saveSite));
        return card;
    }

    /** Repaints only the preview line, so typing never rebuilds the page. */
    private void refreshPreview() {
        LinearLayout row = findPreviewRow();
        if (row == null) return;
        row.removeViewAt(0);
        row.addView(iconPreview(), 0, new LinearLayout.LayoutParams(
                UI.dp(PREVIEW_ICON_DP), UI.dp(PREVIEW_ICON_DP)));
        LinearLayout text = (LinearLayout) row.getChildAt(1);
        String name = nameInput == null ? "" : nameInput.getText().toString().trim();
        String desc = descriptionInput == null ? "" : descriptionInput.getText().toString().trim();
        ((TextView) text.getChildAt(0)).setText(name.isEmpty() ? "Tunnel Manager" : name);
        ((TextView) text.getChildAt(1)).setText(desc.isEmpty() ? "Cloudflare 隧道管理中心" : desc);
    }

    /**
     * Finds the preview row by its shape rather than by index.
     *
     * The card grew a "实时预览" label above it, and anything else added later
     * would shift the index again — matching on structure keeps the lookup from
     * silently pointing at the wrong child.
     */
    private LinearLayout findPreviewRow() {
        if (body == null) return null;
        for (int i = 0; i < body.getChildCount(); i++) {
            View child = body.getChildAt(i);
            if (!(child instanceof LinearLayout)) continue;
            LinearLayout card = (LinearLayout) child;
            for (int j = 0; j < card.getChildCount(); j++) {
                View candidate = card.getChildAt(j);
                if (!(candidate instanceof LinearLayout)) continue;
                LinearLayout row = (LinearLayout) candidate;
                if (row.getChildCount() != 2) continue;
                if (!(row.getChildAt(1) instanceof LinearLayout)) continue;
                LinearLayout text = (LinearLayout) row.getChildAt(1);
                if (text.getChildCount() != 2) continue;
                if (!(text.getChildAt(0) instanceof TextView) || !(text.getChildAt(1) instanceof TextView)) continue;
                return row;
            }
        }
        return null;
    }

    /** A no-op-editable TextWatcher: only the "after" edge matters here. */
    private static final class SimpleWatcher implements android.text.TextWatcher {
        private final Runnable onChanged;

        SimpleWatcher(Runnable onChanged) {
            this.onChanged = onChanged;
        }

        @Override
        public void beforeTextChanged(CharSequence s, int start, int count, int after) {
        }

        @Override
        public void onTextChanged(CharSequence s, int start, int before, int count) {
        }

        @Override
        public void afterTextChanged(android.text.Editable s) {
            onChanged.run();
        }
    }

    private View iconPreview() {
        Palette p = Theme.p();
        if (iconBitmap != null) {
            ImageView view = new ImageView(requireContext());
            view.setImageBitmap(iconBitmap);
            view.setScaleType(ImageView.ScaleType.CENTER_CROP);
            view.setBackground(UI.roundedStroke(p.canvasSoft, UI.RADIUS_MD, p.hairline, 1));
            // The outline comes from the rounded background, which is what keeps
            // a square upload from spilling past the preview's corners.
            view.setClipToOutline(true);
            return view;
        }
        TextView placeholder = UI.text(requireContext(), "◆", 16, p.ink, Typeface.BOLD);
        placeholder.setGravity(Gravity.CENTER);
        placeholder.setBackground(UI.roundedStroke(p.canvasSoft, UI.RADIUS_MD, p.hairline, 1));
        return placeholder;
    }

    private View presetsCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(cardHead("常用 CNAME 组", "维护常用优选线路。域名绑定时可以直接选择，也可以继续手动输入。"));

        preferredInput = UI.input(requireContext(), "cf.090227.xyz");
        preferredInput.setText(preferredDraft);
        card.addView(UI.field(requireContext(), "默认优选 CNAME", preferredInput));
        note(card, "未指定线路时自动使用此值。", UI.SM);

        TextView savePreferred = UI.button(requireContext(), "保存默认值", UI.BTN_SECONDARY);
        savePreferred.setOnClickListener(v -> savePreferredCNAME(preferredInput.getText().toString().trim()));
        UI.addRow(card, savePreferred, UI.SM);

        card.addView(UI.divider(requireContext()));

        for (int i = 0; i < presets.size(); i++) {
            card.addView(presetRow(i));
        }

        TextView add = UI.button(requireContext(), "添加常用 CNAME", UI.BTN_SECONDARY);
        add.setEnabled(presets.size() < MAX_PRESETS);
        add.setOnClickListener(v -> {
            capture();
            if (presets.size() >= MAX_PRESETS) return;
            presets.add(new String[]{"", ""});
            render();
        });
        UI.addRow(card, add, UI.MD);

        card.addView(UI.spacer(requireContext(), UI.MD));
        card.addView(primaryAction("保存 CNAME 组", this::savePresets));
        return card;
    }

    private View presetRow(int index) {
        LinearLayout box = UI.column(requireContext());
        box.setBackground(UI.roundedStroke(Theme.p().canvasSoft2, UI.RADIUS_MD, Theme.p().hairline, 1));
        box.setPadding(UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD));

        LinearLayout head = UI.row(requireContext());
        TextView number = UI.text(requireContext(), String.valueOf(index + 1), 13, Theme.p().mute, Typeface.BOLD);
        head.addView(number, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        ImageView remove = UI.iconButton(requireContext(), R.drawable.ic_nav_trash, Theme.p().mute, v -> {
            capture();
            if (presets.size() <= 1) return;
            presets.remove(index);
            render();
        });
        remove.setEnabled(presets.size() > 1);
        head.addView(remove);
        box.addView(head);

        EditText name = UI.input(requireContext(), "例如：移动优选");
        name.setText(presets.get(index)[0]);
        presetNameInputs.add(name);
        box.addView(UI.field(requireContext(), "线路名称", name, UI.SM));

        EditText value = UI.input(requireContext(), "例如：cdn.example.com");
        value.setText(presets.get(index)[1]);
        presetValueInputs.add(value);
        box.addView(UI.field(requireContext(), "CNAME 地址", value, UI.SM));

        LinearLayout wrap = UI.column(requireContext());
        wrap.addView(box);
        LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT);
        lp.topMargin = UI.dp(index == 0 ? 0 : UI.SM);
        wrap.setLayoutParams(lp);
        return wrap;
    }

    private View fallbackCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(cardHead("回退源设置", "设置 Custom Hostnames 的 Fallback Origin。"));

        fallbackInput = UI.input(requireContext(), "例如: fallback.example.com");
        fallbackInput.setText(fallbackDraft);
        card.addView(fallbackInput);

        TextView save = UI.button(requireContext(), "设置回退源", UI.BTN_SECONDARY);
        save.setOnClickListener(v -> saveFallback(fallbackInput.getText().toString().trim()));
        UI.addRow(card, save, UI.MD);
        return card;
    }

    private View tunnelCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(cardHead("当前隧道", "隧道选择统一在隧道管理页完成，避免误填 ID。"));

        String tunnelId = config.optString("tunnel_id", "");
        String tunnelName = config.optString("tunnel_name", "");
        LinearLayout row = UI.row(requireContext());
        LinearLayout text = UI.column(requireContext());
        text.addView(UI.strong(requireContext(), tunnelId.isEmpty() ? "尚未选择隧道"
                : (tunnelName.isEmpty() ? "已选隧道" : tunnelName)));
        if (!tunnelId.isEmpty()) {
            text.addView(UI.mono(requireContext(), tunnelId, Theme.p().mute));
        }
        row.addView(text, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        TextView go = UI.button(requireContext(), tunnelId.isEmpty() ? "选择隧道" : "切换隧道",
                tunnelId.isEmpty() ? UI.BTN_PRIMARY : UI.BTN_SECONDARY);
        go.setOnClickListener(v -> openRoute("/tunnels"));
        row.addView(go);
        card.addView(row);
        return card;
    }

    // ---------------------------------------------------------------- actions

    private void saveSite() {
        capture();
        if (nameDraft.trim().isEmpty()) {
            statusMessage = "请输入站点名称";
            statusOk = false;
            render();
            return;
        }
        busy = true;
        statusMessage = "";
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("name", nameDraft.trim());
            payload.put("description", descriptionDraft.trim());
            payload.put("icon", iconDraft.trim());
            payload.put("panel_host", panelHostDraft.trim());
            payload.put("landing_enabled", landingEnabled);
            return Api.put("/api/config/site", payload);
        }, result -> {
            busy = false;
            nameDraft = result.optString("name", nameDraft);
            descriptionDraft = result.optString("description", descriptionDraft);
            iconDraft = result.optString("icon", iconDraft);
            landingEnabled = result.optBoolean("landing_enabled", landingEnabled);
            Session.applyConfig(siteIdentity(result));
            statusMessage = "站点信息已更新";
            statusOk = true;
            render();
            loadIcon();
        }, failure -> {
            busy = false;
            statusMessage = "保存失败：" + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    /** Re-shapes the site response into the keys {@link Session} understands. */
    private static JSONObject siteIdentity(JSONObject site) {
        JSONObject identity = new JSONObject();
        try {
            identity.put("site_name", site.optString("name", ""));
        } catch (Exception ignored) {
        }
        return identity;
    }

    private void savePreferredCNAME(String value) {
        if (value.isEmpty()) {
            statusMessage = "请输入默认优选 CNAME";
            statusOk = false;
            render();
            return;
        }
        busy = true;
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("value", value);
            return Api.post("/api/config/preferred-cname", payload);
        }, result -> {
            busy = false;
            preferredDraft = value;
            statusMessage = "默认优选 CNAME 已更新";
            statusOk = true;
            render();
        }, failure -> {
            busy = false;
            statusMessage = "保存失败：" + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    private void savePresets() {
        capture();
        for (String[] preset : presets) {
            if (preset[0].trim().isEmpty() || preset[1].trim().isEmpty()) {
                statusMessage = "请补全所有线路名称和 CNAME 地址";
                statusOk = false;
                render();
                return;
            }
        }
        busy = true;
        render();
        Api.async(() -> {
            JSONArray items = new JSONArray();
            for (String[] preset : presets) {
                JSONObject item = new JSONObject();
                item.put("name", preset[0].trim());
                item.put("value", preset[1].trim());
                items.put(item);
            }
            JSONObject payload = new JSONObject();
            payload.put("items", items);
            return Api.put("/api/config/cname-presets", payload);
        }, result -> {
            busy = false;
            JSONArray saved = result.optJSONArray("cname_presets");
            if (saved != null) {
                presets.clear();
                for (int i = 0; i < saved.length(); i++) {
                    JSONObject item = saved.optJSONObject(i);
                    if (item == null) continue;
                    presets.add(new String[]{item.optString("name", ""), item.optString("value", "")});
                }
            }
            statusMessage = "常用 CNAME 组已更新";
            statusOk = true;
            render();
        }, failure -> {
            busy = false;
            statusMessage = "保存失败：" + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    private void saveFallback(String domain) {
        if (domain.isEmpty()) {
            statusMessage = "请输入回退源域名";
            statusOk = false;
            render();
            return;
        }
        busy = true;
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("domain", domain);
            return Api.post("/api/domain/fallback", payload);
        }, result -> {
            busy = false;
            fallbackDraft = domain;
            statusMessage = result.optString("message", "回退源已设置");
            statusOk = true;
            render();
        }, failure -> {
            busy = false;
            statusMessage = "设置失败：" + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    // --------------------------------------------------------------- icon IO

    private void onIconPicked(Uri uri) {
        if (uri == null) return;
        try {
            String mime = requireContext().getContentResolver().getType(uri);
            if (mime == null || !mime.startsWith("image/")) {
                statusMessage = "请选择图片文件";
                statusOk = false;
                render();
                return;
            }
            byte[] bytes = readAll(uri);
            if (bytes.length > MAX_ICON_BYTES) {
                statusMessage = "图片不能超过 512 KB";
                statusOk = false;
                render();
                return;
            }
            capture();
            iconDraft = "data:" + mime + ";base64," + Base64.encodeToString(bytes, Base64.NO_WRAP);
            iconBitmap = BitmapFactory.decodeByteArray(bytes, 0, bytes.length);
            iconBitmapKey = iconDraft;
            statusMessage = "图片已载入，保存站点信息后生效";
            statusOk = true;
            render();
        } catch (Exception e) {
            statusMessage = "读取图片失败：" + e.getMessage();
            statusOk = false;
            render();
        }
    }

    private byte[] readAll(Uri uri) throws Exception {
        try (InputStream in = requireContext().getContentResolver().openInputStream(uri)) {
            if (in == null) return new byte[0];
            ByteArrayOutputStream buffer = new ByteArrayOutputStream();
            byte[] chunk = new byte[8192];
            int read;
            while ((read = in.read(chunk)) != -1) buffer.write(chunk, 0, read);
            return buffer.toByteArray();
        }
    }
}
