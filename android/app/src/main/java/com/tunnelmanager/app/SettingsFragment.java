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

    // ---- Relocated system & security settings (formerly the 管理后台 page) ----
    // These cards moved here when the multi-user admin console was removed for
    // the single-user build. They call the same backend endpoints as before
    // (/api/admin/settings, /smtp, /oauth, /encryption-key), which stay put.
    private JSONObject sysSettings = new JSONObject();

    private boolean turnstileEnabled = false;
    private String turnstileSiteKey = "";
    private String turnstileSecret = "";
    private boolean turnstileHasSecret = false;
    private EditText siteKeyInput;
    private EditText secretKeyInput;

    private boolean experimentalEnabled = false;

    private String passkeyRPID = "";
    private String passkeyOrigins = "";
    private String androidPackage = "";
    private String androidFingerprints = "";
    private String androidFingerprintsEffective = "";
    private boolean passwordLoginDisabled = false;
    private boolean passkeyAdminReady = false;
    private EditText passkeyRPIDInput;
    private EditText passkeyOriginsInput;
    private EditText androidPackageInput;
    private EditText androidFingerprintsInput;

    private boolean rateLimitEnabled = false;
    private boolean rateLimitNotify = false;
    private String rateLimitPerAccount = "0";
    private String rateLimitPerIP = "0";
    private String rateLimitWindow = "0";
    private String rateLimitFamiliar = "0";
    private EditText rateLimitPerAccountInput;
    private EditText rateLimitPerIPInput;
    private EditText rateLimitWindowInput;
    private EditText rateLimitFamiliarInput;

    private String oauthClientId = "";
    private String oauthClientSecret = "";
    private String oauthRedirect = "";
    private String oauthScopes = "";
    private boolean oauthHasSecret = false;
    private EditText oauthIdInput;
    private EditText oauthSecretInput;
    private EditText oauthRedirectInput;
    private EditText oauthScopesInput;

    private String encKeyInput = "";
    private String encKeySource = "none";
    private EditText encKeyField;

    private String smtpHost = "";
    private String smtpPort = "587";
    private String smtpUsername = "";
    private String smtpPassword = "";
    private String smtpFrom = "";
    private int smtpTls = 0;
    private boolean smtpReady = false;
    private String testMailTo = "";
    private EditText smtpHostInput;
    private EditText smtpPortInput;
    private EditText smtpUserInput;
    private EditText smtpPasswordInput;
    private EditText smtpFromInput;
    private EditText testMailInput;

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
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("config", Api.get("/api/config"));
            payload.put("settings", Api.get("/api/admin/settings"));
            payload.put("smtp", Api.get("/api/admin/smtp"));
            payload.put("oauth", Api.get("/api/admin/oauth"));
            payload.put("key", Api.get("/api/admin/encryption-key"));
            return payload;
        }, payload -> {
            loading = false;
            applyConfig(orEmpty(payload.optJSONObject("config")));
            applySysPayload(payload);
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

    private static JSONObject orEmpty(JSONObject value) {
        return value == null ? new JSONObject() : value;
    }

    /** Reloads only the relocated system/security settings, so a save on one of
     *  those cards does not discard unsaved site/CNAME/fallback drafts. */
    private void loadSys() {
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("settings", Api.get("/api/admin/settings"));
            payload.put("smtp", Api.get("/api/admin/smtp"));
            payload.put("oauth", Api.get("/api/admin/oauth"));
            payload.put("key", Api.get("/api/admin/encryption-key"));
            return payload;
        }, payload -> {
            applySysPayload(payload);
            render();
        }, failure -> render());
    }

    private void applySysPayload(JSONObject payload) {
        sysSettings = orEmpty(payload.optJSONObject("settings"));
        applySmtp(payload.optJSONObject("smtp"));
        applyOauth(payload.optJSONObject("oauth"));
        JSONObject key = payload.optJSONObject("key");
        encKeySource = key == null ? "none" : key.optString("source", "none");
        applySysDrafts();
    }

    private void applySmtp(JSONObject payload) {
        JSONObject data = orEmpty(payload);
        smtpHost = data.optString("host", "");
        smtpPort = String.valueOf(data.optInt("port", 587));
        smtpUsername = data.optString("username", "");
        smtpFrom = data.optString("from", "");
        smtpTls = "plain".equals(data.optString("tls_mode", "")) ? 1 : 0;
        smtpReady = data.optBoolean("configured", false);
        smtpPassword = "";
    }

    private void applyOauth(JSONObject payload) {
        JSONObject data = orEmpty(payload);
        oauthClientId = data.optString("client_id", "");
        oauthRedirect = data.optString("redirect_uri", "");
        oauthScopes = data.optString("scopes", "");
        oauthHasSecret = data.optBoolean("has_client_secret", false);
        oauthClientSecret = "";
    }

    private void applySysDrafts() {
        turnstileEnabled = sysSettings.optBoolean("turnstile_enabled", false);
        turnstileSiteKey = sysSettings.optString("turnstile_site_key", "");
        turnstileHasSecret = sysSettings.optBoolean("turnstile_has_secret", false);
        turnstileSecret = "";
        experimentalEnabled = sysSettings.optBoolean("experimental_features_enabled", false);
        passkeyRPID = sysSettings.optString("passkey_rp_id", "");
        passkeyOrigins = sysSettings.optString("passkey_origins", "");
        androidPackage = sysSettings.optString("passkey_android_package", "");
        androidFingerprints = sysSettings.optString("passkey_android_fingerprints", "");
        JSONArray effective = sysSettings.optJSONArray("passkey_android_fingerprints_effective");
        androidFingerprintsEffective = effective == null ? "" : effective.toString();
        passwordLoginDisabled = sysSettings.optBoolean("password_login_disabled", false);
        passkeyAdminReady = sysSettings.optBoolean("passkey_admin_ready", false);
        rateLimitEnabled = sysSettings.optBoolean("rate_limit_enabled", false);
        rateLimitNotify = sysSettings.optBoolean("rate_limit_notify", false);
        rateLimitPerAccount = String.valueOf(sysSettings.optInt("rate_limit_per_account", 0));
        rateLimitPerIP = String.valueOf(sysSettings.optInt("rate_limit_per_ip", 0));
        rateLimitWindow = String.valueOf(sysSettings.optInt("rate_limit_window_minutes", 0));
        rateLimitFamiliar = String.valueOf(sysSettings.optInt("rate_limit_familiar_multiplier", 0));
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
        // Relocated system/security inputs.
        if (siteKeyInput != null) turnstileSiteKey = siteKeyInput.getText().toString().trim();
        if (secretKeyInput != null) turnstileSecret = secretKeyInput.getText().toString();
        if (oauthIdInput != null) oauthClientId = oauthIdInput.getText().toString().trim();
        if (oauthSecretInput != null) oauthClientSecret = oauthSecretInput.getText().toString();
        if (oauthRedirectInput != null) oauthRedirect = oauthRedirectInput.getText().toString().trim();
        if (oauthScopesInput != null) oauthScopes = oauthScopesInput.getText().toString().trim();
        if (encKeyField != null) encKeyInput = encKeyField.getText().toString().trim();
        if (passkeyRPIDInput != null) passkeyRPID = passkeyRPIDInput.getText().toString().trim();
        if (passkeyOriginsInput != null) passkeyOrigins = passkeyOriginsInput.getText().toString().trim();
        if (androidPackageInput != null) androidPackage = androidPackageInput.getText().toString().trim();
        if (androidFingerprintsInput != null) androidFingerprints = androidFingerprintsInput.getText().toString().trim();
        if (rateLimitPerAccountInput != null) rateLimitPerAccount = rateLimitPerAccountInput.getText().toString().trim();
        if (rateLimitPerIPInput != null) rateLimitPerIP = rateLimitPerIPInput.getText().toString().trim();
        if (rateLimitWindowInput != null) rateLimitWindow = rateLimitWindowInput.getText().toString().trim();
        if (rateLimitFamiliarInput != null) rateLimitFamiliar = rateLimitFamiliarInput.getText().toString().trim();
        if (smtpHostInput != null) smtpHost = smtpHostInput.getText().toString().trim();
        if (smtpPortInput != null) smtpPort = smtpPortInput.getText().toString().trim();
        if (smtpUserInput != null) smtpUsername = smtpUserInput.getText().toString().trim();
        if (smtpPasswordInput != null) smtpPassword = smtpPasswordInput.getText().toString();
        if (smtpFromInput != null) smtpFrom = smtpFromInput.getText().toString().trim();
        if (testMailInput != null) testMailTo = testMailInput.getText().toString().trim();
    }

    private void render() {
        if (body == null || !alive()) return;
        int keepScroll = scroll.getScrollY();
        body.removeAllViews();

        body.addView(UI.pageTitle(requireContext(), "全局设置"));
        TextView subtitle = UI.muted(requireContext(), "站点品牌、域名绑定偏好、回退源、隧道，以及人机验证、OAuth、加密密钥、通行密钥、登录保护与邮件服务");
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

        // Relocated system & security section.
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(turnstileCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(oauthCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(encryptionCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(passkeySettingsCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(rateLimitCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(experimentalCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(smtpCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(smtpTestCard());

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

    // ================ Relocated system & security cards ==================
    // Moved from the removed 管理后台 (AdminFragment). Same endpoints, same
    // behaviour; user/group/invite/audit management is gone with single-user.

    /** A card that opens with a title and an optional description line. */
    private LinearLayout titledCard(String title, String desc) {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), title));
        if (desc != null && !desc.isEmpty()) {
            TextView hint = UI.muted(requireContext(), desc);
            UI.margin(hint, 0, UI.XS, 0, UI.MD);
            card.addView(hint);
        }
        return card;
    }

    private interface ToggleListener {
        void onChange(boolean next);
    }

    private View toggleRow(String label, boolean on, boolean enabled, ToggleListener listener) {
        LinearLayout row = UI.row(requireContext());
        UI.Toggle toggle = new UI.Toggle(requireContext(), on);
        toggle.setEnabled(enabled);
        toggle.setAlpha(enabled ? 1f : 0.5f);
        if (enabled) toggle.setOnClickListener(v -> listener.onChange(!toggle.isOn()));
        row.addView(toggle, new LinearLayout.LayoutParams(UI.dp(34), UI.dp(20)));
        TextView text = UI.body(requireContext(), label);
        if (!enabled) text.setAlpha(0.5f);
        UI.margin(text, UI.MD, 0, 0, 0);
        row.addView(text);
        return row;
    }

    private TextView tag(String label, boolean ok) {
        return UI.tag(requireContext(), label, ok);
    }

    private EditText passwordInput(String hint) {
        EditText input = UI.input(requireContext(), hint);
        input.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_PASSWORD);
        return input;
    }

    private EditText numberInput(String hint, String value) {
        EditText input = UI.input(requireContext(), hint);
        input.setInputType(InputType.TYPE_CLASS_NUMBER);
        input.setText(value);
        return input;
    }

    private EditText signedInput(String hint, String value) {
        EditText input = UI.input(requireContext(), hint);
        input.setInputType(InputType.TYPE_CLASS_NUMBER | InputType.TYPE_NUMBER_FLAG_SIGNED);
        input.setText(value);
        return input;
    }

    private static int draftInt(String value) {
        try {
            return Integer.parseInt(value.trim());
        } catch (NumberFormatException e) {
            return 0;
        }
    }

    private static int parseInt(String raw) {
        try {
            return Integer.parseInt(raw.trim());
        } catch (NumberFormatException e) {
            return 0;
        }
    }

    private interface Callable {
        JSONObject call() throws Exception;
    }

    /** Runs a write, reloads the system settings, and reports the outcome. */
    private void run(Callable call, String success) {
        busy = true;
        statusMessage = "";
        render();
        Api.async(call::call, result -> {
            busy = false;
            statusMessage = success;
            statusOk = true;
            loadSys();
        }, failure -> {
            busy = false;
            statusMessage = "保存失败：" + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    // ---- the single PUT /api/admin/settings document -----------------------
    // The endpoint replaces the whole document, so every writable field travels
    // on every save. Registration/audit fields were dropped server-side and are
    // deliberately never sent (the decoder rejects unknown fields).
    private void saveAppSettings(String success) {
        capture();
        JSONObject payload = new JSONObject();
        try {
            payload.put("turnstile_enabled", turnstileEnabled);
            payload.put("turnstile_site_key", turnstileSiteKey);
            if (!turnstileSecret.isEmpty()) payload.put("turnstile_secret", turnstileSecret);
            payload.put("experimental_features_enabled", experimentalEnabled);
            payload.put("password_login_disabled", passwordLoginDisabled);
            payload.put("passkey_rp_id", passkeyRPID);
            payload.put("passkey_origins", passkeyOrigins);
            payload.put("passkey_android_package", androidPackage);
            payload.put("passkey_android_fingerprints", androidFingerprints);
            payload.put("rate_limit_enabled", rateLimitEnabled);
            payload.put("rate_limit_notify", rateLimitNotify);
            payload.put("rate_limit_per_account", draftInt(rateLimitPerAccount));
            payload.put("rate_limit_per_ip", draftInt(rateLimitPerIP));
            payload.put("rate_limit_window_minutes", draftInt(rateLimitWindow));
            payload.put("rate_limit_familiar_multiplier", draftInt(rateLimitFamiliar));
        } catch (org.json.JSONException ignored) {
            // JSONObject.put only throws on NaN/Infinity.
        }
        turnstileSecret = "";
        final JSONObject body = payload;
        run(() -> {
            // Keeps the nav experimental flag in step without a full reload.
            Session.applyConfig(body);
            return Api.put("/api/admin/settings", body);
        }, success);
    }

    private View turnstileCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), "人机验证（Cloudflare Turnstile）"));
        TextView hint = UI.muted(requireContext(),
                "可选防护：开启后登录、找回密码需要完成 Cloudflare 人机验证。"
                        + "先在 Cloudflare 控制台创建 Turnstile widget（把本站域名加入允许域名），再填写 Site Key 与 Secret Key。");
        UI.margin(hint, 0, UI.XS, 0, UI.MD);
        card.addView(hint);

        UI.addRow(card, toggleRow("启用人机验证", turnstileEnabled, true, next -> {
            capture();
            turnstileEnabled = next;
            saveAppSettings("人机验证设置已保存");
        }), 0);

        siteKeyInput = UI.input(requireContext(), "Site Key（0x4A…）");
        siteKeyInput.setText(turnstileSiteKey);
        card.addView(UI.field(requireContext(), "Site Key", siteKeyInput, UI.MD));

        secretKeyInput = passwordInput("Secret Key（留空保持不变）");
        secretKeyInput.setText(turnstileSecret);
        card.addView(UI.field(requireContext(), "Secret Key", secretKeyInput, UI.MD));

        TextView state = tag(turnstileHasSecret ? "密钥：已设置" : "密钥：未设置", turnstileHasSecret);
        UI.margin(state, 0, UI.SM, 0, UI.MD);
        card.addView(state);

        TextView save = UI.button(requireContext(), busy ? "保存中…" : "保存", UI.BTN_PRIMARY);
        save.setEnabled(!busy);
        save.setOnClickListener(v -> saveAppSettings("人机验证设置已保存"));
        UI.addRow(card, save, 0);
        return card;
    }

    private View oauthCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), "Cloudflare OAuth 客户端"));
        TextView hint = UI.muted(requireContext(),
                "留空时使用环境变量 CF_OAUTH_CLIENT_ID / CF_OAUTH_CLIENT_SECRET；此处填写后优先于环境变量，密钥留空表示保持不变。\n"
                        + "OAuth 客户端需勾选权限：Account Settings:Read、Cloudflare Tunnel:Edit、Zone:Read、DNS:Edit、SSL and Certificates:Edit。");
        UI.margin(hint, 0, UI.XS, 0, UI.MD);
        card.addView(hint);

        oauthIdInput = UI.input(requireContext(), "Client ID");
        oauthIdInput.setText(oauthClientId);
        card.addView(UI.field(requireContext(), "Client ID", oauthIdInput));

        oauthSecretInput = passwordInput("Client Secret（留空保持不变）");
        oauthSecretInput.setText(oauthClientSecret);
        card.addView(UI.field(requireContext(), "Client Secret", oauthSecretInput, UI.MD));

        oauthRedirectInput = UI.input(requireContext(), "回调地址（留空自动按访问域名推导）");
        oauthRedirectInput.setText(oauthRedirect);
        card.addView(UI.field(requireContext(), "回调地址", oauthRedirectInput, UI.MD));

        oauthScopesInput = UI.input(requireContext(), "Scopes（留空用客户端已配置的）");
        oauthScopesInput.setText(oauthScopes);
        card.addView(UI.field(requireContext(), "Scopes", oauthScopesInput, UI.MD));

        TextView state = tag(oauthHasSecret ? "密钥：已设置" : "密钥：未设置", oauthHasSecret);
        UI.margin(state, 0, UI.SM, 0, UI.MD);
        card.addView(state);

        TextView save = UI.button(requireContext(), busy ? "保存中…" : "保存", UI.BTN_PRIMARY);
        save.setEnabled(!busy);
        save.setOnClickListener(v -> saveOauth());
        UI.addRow(card, save, 0);
        return card;
    }

    private void saveOauth() {
        capture();
        final String secret = oauthClientSecret;
        oauthClientSecret = "";
        run(() -> {
            JSONObject payload = new JSONObject();
            payload.put("client_id", oauthClientId);
            payload.put("redirect_uri", oauthRedirect);
            payload.put("scopes", oauthScopes);
            if (!secret.isEmpty()) payload.put("client_secret", secret);
            return Api.put("/api/admin/oauth", payload);
        }, "OAuth 客户端配置已保存");
    }

    private View encryptionCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), "应用加密密钥"));
        TextView hint = UI.muted(requireContext(),
                "用于加密 SMTP 密码、Cloudflare 授权令牌与 2FA 密文。未设置时自动生成并保存在数据库；"
                        + "环境变量 APP_ENCRYPTION_KEY 优先。更换后需重启服务，且会使已保存的密文失效。");
        UI.margin(hint, 0, UI.XS, 0, UI.MD);
        card.addView(hint);

        encKeyField = UI.input(requireContext(), "64 位十六进制密钥（留空保持不变）");
        encKeyField.setText(encKeyInput);
        card.addView(encKeyField);

        String source = "env".equals(encKeySource) ? "环境变量" : "stored".equals(encKeySource) ? "数据库" : "未设置";
        TextView state = tag("当前来源：" + source, !"none".equals(encKeySource));
        UI.margin(state, 0, UI.SM, 0, UI.MD);
        card.addView(state);

        TextView save = UI.button(requireContext(), busy ? "保存中…" : "保存密钥", UI.BTN_PRIMARY);
        save.setEnabled(!busy && !encKeyInput.trim().isEmpty());
        save.setOnClickListener(v -> saveEncKey());
        UI.addRow(card, save, 0);
        return card;
    }

    private void saveEncKey() {
        capture();
        if (encKeyInput.isEmpty()) {
            statusMessage = "请输入 64 位十六进制密钥";
            statusOk = false;
            render();
            return;
        }
        String key = encKeyInput;
        encKeyInput = "";
        run(() -> Api.put("/api/admin/encryption-key", new JSONObject().put("key", key)), "加密密钥已保存");
    }

    private View passkeySettingsCard() {
        LinearLayout card = titledCard("通行密钥",
                "通行密钥要求 HTTPS，依赖方 ID 只能是访问域名本身或其父域。两个字段都留空时按访问域名自动推导，同域下的子域都能用。「允许的来源」只在需要限定同一依赖方下放开哪几个子域时才填，须与依赖方 ID 同域或为其子域；一个依赖方 ID 覆盖不了两个不同的域名。");

        passkeyRPIDInput = UI.input(requireContext(), "依赖方 ID，如 panel.example.com");
        passkeyRPIDInput.setText(passkeyRPID);
        card.addView(UI.field(requireContext(), "依赖方 ID", passkeyRPIDInput, UI.SM));

        passkeyOriginsInput = UI.input(requireContext(), "允许的来源，逗号分隔");
        passkeyOriginsInput.setText(passkeyOrigins);
        card.addView(UI.field(requireContext(), "允许的来源", passkeyOriginsInput, UI.SM));

        androidPackageInput = UI.input(requireContext(), "com.tunnelmanager.app");
        androidPackageInput.setText(androidPackage);
        card.addView(UI.field(requireContext(), "Android 包名", androidPackageInput, UI.SM));

        androidFingerprintsInput = UI.textArea(requireContext(), "SHA-256 指纹，逗号或换行分隔");
        androidFingerprintsInput.setText(androidFingerprints);
        card.addView(UI.field(requireContext(), "Android 签名指纹", androidFingerprintsInput, UI.SM));
        card.addView(UI.muted(requireContext(), androidFingerprintsEffective.isEmpty()
                ? "当前没有生效的签名指纹，App 无法使用通行密钥。"
                : "当前生效：" + androidFingerprintsEffective));
        card.addView(UI.muted(requireContext(), "App 通过 /.well-known/assetlinks.json 校验域名归属，指纹取自签名证书（keytool 或 apksigner 的 SHA-256）。"));

        TextView save = UI.button(requireContext(), busy ? "保存中…" : "保存通行密钥设置", UI.BTN_PRIMARY);
        save.setEnabled(!busy);
        save.setOnClickListener(v -> saveAppSettings("通行密钥设置已保存"));
        UI.margin(save, 0, UI.MD, 0, 0);
        card.addView(save);

        card.addView(UI.divider(requireContext()));
        UI.addRow(card, toggleRow("禁用密码登录（全站）", passwordLoginDisabled, passkeyAdminReady, next -> {
            passwordLoginDisabled = next;
            saveAppSettings(next ? "已禁用密码登录（全站）" : "已恢复密码登录（全站）");
        }), UI.MD);
        card.addView(UI.muted(requireContext(), passkeyAdminReady
                ? "开启后所有账户都只能用通行密钥登录；账户丢失通行密钥时，可用 CLI 的 --allow-password-login 恢复。"
                : "需管理员已绑定通行密钥，避免面板被锁死。"));
        return card;
    }

    private View rateLimitCard() {
        LinearLayout card = titledCard("登录保护",
                "登录、二次验证、找回与重置密码都受此限制。账号额度是真正的防线，换 IP 绕不过去；IP 额度用来挡住一台机器横扫多个账号。任一触发即返回 429 并附带解锁时间。");

        UI.addRow(card, toggleRow("启用登录限流", rateLimitEnabled, true, next -> {
            rateLimitEnabled = next;
            saveAppSettings("登录保护设置已保存");
        }), UI.SM);

        rateLimitPerAccountInput = signedInput("0 = 默认 5 次，负数 = 不限", rateLimitPerAccount);
        card.addView(UI.field(requireContext(), "每账号失败次数", rateLimitPerAccountInput, UI.SM));

        rateLimitPerIPInput = signedInput("0 = 默认 20 次，负数 = 不限", rateLimitPerIP);
        card.addView(UI.field(requireContext(), "每 IP 失败次数", rateLimitPerIPInput, UI.SM));

        rateLimitWindowInput = signedInput("0 = 默认 15 分钟", rateLimitWindow);
        card.addView(UI.field(requireContext(), "统计窗口（分钟）", rateLimitWindowInput, UI.SM));

        rateLimitFamiliarInput = signedInput("0 = 默认 3 倍，1 = 不放宽", rateLimitFamiliar);
        card.addView(UI.field(requireContext(), "熟悉来源的额度倍数", rateLimitFamiliarInput, UI.SM));
        card.addView(UI.muted(requireContext(), "90 天内登录过的网段（IPv4 记 /24，IPv6 记 /64）会给更宽的账号额度。放宽是倍数而不是放行，所以熟悉的网段也会用完。"));

        TextView save = UI.button(requireContext(), busy ? "保存中…" : "保存限流设置", UI.BTN_PRIMARY);
        save.setEnabled(!busy);
        save.setOnClickListener(v -> saveAppSettings("登录保护设置已保存"));
        UI.margin(save, 0, UI.MD, 0, 0);
        card.addView(save);

        UI.addRow(card, toggleRow("锁定时通知管理员", rateLimitNotify, true, next -> {
            rateLimitNotify = next;
            saveAppSettings("登录保护设置已保存");
        }), UI.MD);
        card.addView(UI.muted(requireContext(), "走管理员配置的通知渠道（TG / 邮件），并沿用「登录通知」开关。"));
        return card;
    }

    private View experimentalCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), "实验性功能"));
        TextView hint = UI.muted(requireContext(),
                "开启后，侧边栏会显示「IP 优选实验室」。关闭后入口与 API 均不暴露。");
        UI.margin(hint, 0, UI.XS, 0, UI.MD);
        card.addView(hint);
        UI.addRow(card, toggleRow("开启实验性功能", experimentalEnabled, true, next -> {
            experimentalEnabled = next;
            saveAppSettings("实验性功能设置已保存");
        }), 0);
        return card;
    }

    private View smtpCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), "SMTP 邮件服务"));
        TextView hint = UI.muted(requireContext(),
                "用于监控告警邮件与找回密码验证码。密码留空表示保持原值不变。"
                        + (smtpReady ? "" : " 当前尚未完整配置。"));
        UI.margin(hint, 0, UI.XS, 0, UI.MD);
        card.addView(hint);

        smtpHostInput = UI.input(requireContext(), "smtp.example.com");
        smtpHostInput.setText(smtpHost);
        card.addView(UI.field(requireContext(), "SMTP 主机", smtpHostInput));

        smtpPortInput = numberInput("465 或 587", smtpPort);
        card.addView(UI.field(requireContext(), "端口", smtpPortInput, UI.MD));

        smtpUserInput = UI.input(requireContext(), "用户名（通常为邮箱）");
        smtpUserInput.setText(smtpUsername);
        card.addView(UI.field(requireContext(), "用户名", smtpUserInput, UI.MD));

        smtpPasswordInput = passwordInput("密码 / 授权码（留空保持不变）");
        smtpPasswordInput.setText(smtpPassword);
        card.addView(UI.field(requireContext(), "密码 / 授权码", smtpPasswordInput, UI.MD));

        smtpFromInput = UI.input(requireContext(), "panel@example.com");
        smtpFromInput.setText(smtpFrom);
        card.addView(UI.field(requireContext(), "发件人", smtpFromInput, UI.MD));

        TextView tlsLabel = UI.label(requireContext(), "传输加密");
        UI.margin(tlsLabel, 0, UI.MD, 0, UI.SM);
        card.addView(tlsLabel);
        UI.addRow(card, UI.segmented(requireContext(), new String[]{"加密", "不加密"}, smtpTls, index -> {
            capture();
            smtpTls = index;
            render();
        }), 0);

        TextView save = UI.button(requireContext(), busy ? "保存中…" : "保存设置", UI.BTN_PRIMARY);
        save.setEnabled(!busy);
        save.setOnClickListener(v -> saveSmtp());
        UI.addRow(card, save, UI.MD);
        return card;
    }

    private View smtpTestCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), "发送测试邮件"));
        testMailInput = UI.input(requireContext(), "你的邮箱");
        testMailInput.setInputType(InputType.TYPE_TEXT_VARIATION_EMAIL_ADDRESS);
        testMailInput.setText(testMailTo);
        card.addView(UI.field(requireContext(), "收件地址", testMailInput));
        TextView send = UI.button(requireContext(), busy ? "发送中…" : "发送测试邮件", UI.BTN_SECONDARY);
        send.setEnabled(!busy);
        send.setOnClickListener(v -> {
            capture();
            if (testMailTo.isEmpty()) {
                statusMessage = "请输入收件邮箱";
                statusOk = false;
                render();
                return;
            }
            String to = testMailTo;
            run(() -> Api.post("/api/admin/smtp/test", new JSONObject().put("to", to)), "测试邮件已发送");
        });
        UI.addRow(card, send, UI.MD);
        return card;
    }

    private void saveSmtp() {
        capture();
        final String password = smtpPassword;
        smtpPassword = "";
        run(() -> {
            JSONObject payload = new JSONObject();
            payload.put("host", smtpHost);
            payload.put("port", parseInt(smtpPort));
            payload.put("username", smtpUsername);
            payload.put("from", smtpFrom);
            payload.put("tls_mode", smtpTls == 1 ? "plain" : "ssl");
            if (!password.isEmpty()) payload.put("password", password);
            return Api.put("/api/admin/smtp", payload);
        }, "SMTP 设置已保存");
    }
}
