package com.tunnelmanager.app;

import android.content.Intent;
import android.os.Bundle;
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
import androidx.appcompat.app.AlertDialog;

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
    @Override boolean canRefresh() { return !busy && !loading && !sysLoading; }
    @Override void refreshPage() { if (canRefresh()) load(); }

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

    private boolean loading = false;
    private boolean configLoaded = false;
    private boolean sysLoaded = false;
    private boolean sysLoading = false;
    private JSONObject sysSnapshot = new JSONObject();
    private boolean busy = false;
    private int readGeneration;
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
    private int labDailyRequestLimit = 100000;
    private int labRequestsPerSecond = 10;
    private int labMaxWorkers = 32;

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
        if (!configLoaded) load();
        else {
            loadIcon();
            if (!sysLoaded) loadSys();
        }
        return scroll;
    }

    @Override public void onCreate(Bundle saved) {
        super.onCreate(saved);
        if (saved == null || !saved.containsKey("settingsConfig")) return;
        try {
            JSONObject restored = new JSONObject(saved.getString("settingsConfig"));
            if (!restored.has("site_name")) return;
            applyConfig(restored);
            if (saved.containsKey("settingsSystem")) {
                JSONObject restoredSystem = new JSONObject(saved.getString("settingsSystem"));
                if (restoredSystem.has("settings")) applySysPayload(restoredSystem);
            }
        } catch (Exception ignored) { return; }
        nameDraft = saved.getString("settingsName", nameDraft);
        descriptionDraft = saved.getString("settingsDescription", descriptionDraft);
        iconDraft = saved.getString("settingsIcon", iconDraft);
        panelHostDraft = saved.getString("settingsHost", panelHostDraft);
        preferredDraft = saved.getString("settingsPreferred", preferredDraft);
        fallbackDraft = saved.getString("settingsFallback", fallbackDraft);
        landingEnabled = saved.getBoolean("settingsLanding", landingEnabled);
        java.util.ArrayList<String> names = saved.getStringArrayList("settingsPresetNames");
        java.util.ArrayList<String> values = saved.getStringArrayList("settingsPresetValues");
        if (names != null && values != null && names.size() == values.size()) {
            presets.clear();
            for (int i = 0; i < names.size(); i++) presets.add(new String[]{names.get(i), values.get(i)});
        }
        turnstileSiteKey = saved.getString("settingsSys_turnstileSiteKey", turnstileSiteKey);
        passkeyRPID = saved.getString("settingsSys_passkeyRPID", passkeyRPID);
        passkeyOrigins = saved.getString("settingsSys_passkeyOrigins", passkeyOrigins);
        androidPackage = saved.getString("settingsSys_androidPackage", androidPackage);
        androidFingerprints = saved.getString("settingsSys_androidFingerprints", androidFingerprints);
        rateLimitPerAccount = saved.getString("settingsSys_rateLimitPerAccount", rateLimitPerAccount);
        rateLimitPerIP = saved.getString("settingsSys_rateLimitPerIP", rateLimitPerIP);
        rateLimitWindow = saved.getString("settingsSys_rateLimitWindow", rateLimitWindow);
        rateLimitFamiliar = saved.getString("settingsSys_rateLimitFamiliar", rateLimitFamiliar);
        oauthClientId = saved.getString("settingsSys_oauthClientId", oauthClientId);
        oauthRedirect = saved.getString("settingsSys_oauthRedirect", oauthRedirect);
        oauthScopes = saved.getString("settingsSys_oauthScopes", oauthScopes);
        smtpHost = saved.getString("settingsSys_smtpHost", smtpHost);
        smtpPort = saved.getString("settingsSys_smtpPort", smtpPort);
        smtpUsername = saved.getString("settingsSys_smtpUsername", smtpUsername);
        smtpFrom = saved.getString("settingsSys_smtpFrom", smtpFrom);
        testMailTo = saved.getString("settingsSys_testMailTo", testMailTo);
        turnstileEnabled = saved.getBoolean("settingsSys_turnstileEnabled", turnstileEnabled);
        experimentalEnabled = saved.getBoolean("settingsSys_experimentalEnabled", experimentalEnabled);
        passwordLoginDisabled = saved.getBoolean("settingsSys_passwordLoginDisabled", passwordLoginDisabled);
        rateLimitEnabled = saved.getBoolean("settingsSys_rateLimitEnabled", rateLimitEnabled);
        rateLimitNotify = saved.getBoolean("settingsSys_rateLimitNotify", rateLimitNotify);
        smtpTls = saved.getInt("settingsSys_smtpTls", smtpTls);
        labDailyRequestLimit = saved.getInt("settingsSys_labDailyRequestLimit", labDailyRequestLimit);
        labRequestsPerSecond = saved.getInt("settingsSys_labRequestsPerSecond", labRequestsPerSecond);
        labMaxWorkers = saved.getInt("settingsSys_labMaxWorkers", labMaxWorkers);
        loading = false;
    }

    @Override public void onSaveInstanceState(@NonNull Bundle saved) {
        capture();
        if (configLoaded) saved.putString("settingsConfig", config.toString());
        if (sysLoaded) saved.putString("settingsSystem", sysSnapshot.toString());
        saved.putString("settingsName", nameDraft);
        saved.putString("settingsDescription", descriptionDraft);
        saved.putString("settingsIcon", iconDraft);
        saved.putString("settingsHost", panelHostDraft);
        saved.putString("settingsPreferred", preferredDraft);
        saved.putString("settingsFallback", fallbackDraft);
        saved.putBoolean("settingsLanding", landingEnabled);
        java.util.ArrayList<String> names = new java.util.ArrayList<>(), values = new java.util.ArrayList<>();
        for (String[] preset : presets) { names.add(preset[0]); values.add(preset[1]); }
        saved.putStringArrayList("settingsPresetNames", names);
        saved.putStringArrayList("settingsPresetValues", values);
        saved.putString("settingsSys_turnstileSiteKey", turnstileSiteKey);
        saved.putString("settingsSys_passkeyRPID", passkeyRPID);
        saved.putString("settingsSys_passkeyOrigins", passkeyOrigins);
        saved.putString("settingsSys_androidPackage", androidPackage);
        saved.putString("settingsSys_androidFingerprints", androidFingerprints);
        saved.putString("settingsSys_rateLimitPerAccount", rateLimitPerAccount);
        saved.putString("settingsSys_rateLimitPerIP", rateLimitPerIP);
        saved.putString("settingsSys_rateLimitWindow", rateLimitWindow);
        saved.putString("settingsSys_rateLimitFamiliar", rateLimitFamiliar);
        saved.putString("settingsSys_oauthClientId", oauthClientId);
        saved.putString("settingsSys_oauthRedirect", oauthRedirect);
        saved.putString("settingsSys_oauthScopes", oauthScopes);
        saved.putString("settingsSys_smtpHost", smtpHost);
        saved.putString("settingsSys_smtpPort", smtpPort);
        saved.putString("settingsSys_smtpUsername", smtpUsername);
        saved.putString("settingsSys_smtpFrom", smtpFrom);
        saved.putString("settingsSys_testMailTo", testMailTo);
        saved.putBoolean("settingsSys_turnstileEnabled", turnstileEnabled);
        saved.putBoolean("settingsSys_experimentalEnabled", experimentalEnabled);
        saved.putBoolean("settingsSys_passwordLoginDisabled", passwordLoginDisabled);
        saved.putBoolean("settingsSys_rateLimitEnabled", rateLimitEnabled);
        saved.putBoolean("settingsSys_rateLimitNotify", rateLimitNotify);
        saved.putInt("settingsSys_smtpTls", smtpTls);
        saved.putInt("settingsSys_labDailyRequestLimit", labDailyRequestLimit);
        saved.putInt("settingsSys_labRequestsPerSecond", labRequestsPerSecond);
        saved.putInt("settingsSys_labMaxWorkers", labMaxWorkers);
        super.onSaveInstanceState(saved);
    }

    @Override public void onDestroyView() {
        readGeneration++;
        loading = false;
        sysLoading = false;
        scroll = null;
        body = null;
        super.onDestroyView();
    }

    // ------------------------------------------------------------------- data

    private void load() {
        if (!alive() || loading || sysLoading || busy) return;
        final int request = ++readGeneration;
        loading = true;
        render();
        Api.async(() -> Api.batch(Api.query("config", "/api/config"), Api.query("settings", "/api/admin/settings"),
                Api.query("smtp", "/api/admin/smtp"), Api.query("oauth", "/api/admin/oauth"),
                Api.query("key", "/api/admin/encryption-key")), payload -> {
            if (!alive() || request != readGeneration) return;
            loading = false;
            applyConfig(orEmpty(payload.optJSONObject("config")));
            applySysPayload(payload);
            statusMessage = "";
            render();
            loadIcon();
        }, failure -> {
            if (!alive() || request != readGeneration) return;
            loading = false;
            statusMessage = I18n.text("m_01de8216e0d0") + failure.getMessage();
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
        if (!alive() || sysLoading || loading || busy) return;
        final int request = ++readGeneration;
        sysLoading = true;
        Api.async(() -> Api.batch(Api.query("settings", "/api/admin/settings"), Api.query("smtp", "/api/admin/smtp"),
                Api.query("oauth", "/api/admin/oauth"), Api.query("key", "/api/admin/encryption-key")), payload -> {
            if (!alive() || request != readGeneration) return;
            sysLoading = false;
            applySysPayload(payload);
            render();
        }, failure -> { if (!alive() || request != readGeneration) return; sysLoading = false; render(); });
    }

    private void applySysPayload(JSONObject payload) {
        sysLoaded = true;
        try {
            JSONObject safeSettings = new JSONObject(orEmpty(payload.optJSONObject("settings")).toString());
            safeSettings.remove("turnstile_secret_key"); safeSettings.remove("turnstile_secret");
            JSONObject safeSMTP = new JSONObject(orEmpty(payload.optJSONObject("smtp")).toString()); safeSMTP.remove("password");
            JSONObject safeOAuth = new JSONObject(orEmpty(payload.optJSONObject("oauth")).toString()); safeOAuth.remove("client_secret");
            JSONObject keyMetadata = orEmpty(payload.optJSONObject("key"));
            sysSnapshot = new JSONObject().put("settings", safeSettings).put("smtp", safeSMTP).put("oauth", safeOAuth)
                    .put("key", new JSONObject().put("source", keyMetadata.optString("source", "none")));
        } catch (Exception ignored) { sysSnapshot = new JSONObject(); }
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
        labDailyRequestLimit = sysSettings.optInt("lab_daily_request_limit", 100000);
        labRequestsPerSecond = sysSettings.optInt("lab_requests_per_second", 10);
        labMaxWorkers = sysSettings.optInt("lab_max_workers", 32);
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
        configLoaded = true;
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

        TextView subtitle = UI.muted(requireContext(), I18n.text("m_304845c555ff") + "。" + I18n.text("slim_8523c968bd2b"));
        UI.margin(subtitle, 0, UI.XS, 0, UI.MD);
        body.addView(subtitle);

        if (loading) {
            body.addView(UI.muted(requireContext(), I18n.text("m_4927a53bcc88")));
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

        if (!sysLoaded) {
            UI.addRow(body, UI.muted(requireContext(), I18n.text("m_4927a53bcc88")), UI.MD);
        } else {
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

        }

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
        TextView button = UI.button(requireContext(), busy ? I18n.text("m_d3d21191f32e") : label, UI.BTN_PRIMARY);
        button.setEnabled(!busy);
        button.setOnClickListener(v -> action.run());
        return button;
    }

    // ------------------------------------------------------------------ cards

    private View siteCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(cardHead(I18n.text("m_a719b867f3fc"), I18n.text("m_22196a1072dc")));

        TextView previewLabel = UI.label(requireContext(), I18n.text("m_a76fdaad7519"));
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
                descriptionDraft.trim().isEmpty() ? I18n.text("m_c2c8982623de") : descriptionDraft.trim());
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
        card.addView(UI.field(requireContext(), I18n.text("m_8ac4f0447b99"), nameInput, UI.MD));

        descriptionInput = UI.input(requireContext(), I18n.text("m_c2c8982623de"));
        descriptionInput.setText(descriptionDraft);
        descriptionInput.addTextChangedListener(new SimpleWatcher(this::refreshPreview));
        card.addView(UI.field(requireContext(), I18n.text("m_9ddde2b92952"), descriptionInput, UI.MD));

        iconInput = UI.input(requireContext(), I18n.text("m_22f956aa4f06"));
        iconInput.setText(iconDraft);
        iconInput.addTextChangedListener(new SimpleWatcher(() -> {
            iconDraft = iconInput.getText().toString().trim();
            ui.removeCallbacks(iconReload);
            ui.postDelayed(iconReload, 450);
        }));
        card.addView(UI.field(requireContext(), I18n.text("m_e220e37bbb93"), iconInput, UI.MD));

        LinearLayout iconActions = UI.row(requireContext());
        TextView upload = UI.button(requireContext(), I18n.text("m_9fbb4750bb39"), UI.BTN_SECONDARY);
        upload.setOnClickListener(v -> iconPicker.launch("image/*"));
        iconActions.addView(upload);
        if (!iconDraft.isEmpty()) {
            TextView clear = UI.button(requireContext(), I18n.text("m_10e0fd099bd2"), UI.BTN_GHOST);
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
        note(card, I18n.text("m_8647947e6b61"), UI.SM);

        panelHostInput = UI.input(requireContext(), "panel.example.com");
        panelHostInput.setText(panelHostDraft);
        card.addView(UI.field(requireContext(), I18n.text("m_d22b7b6d0776"), panelHostInput, UI.MD));
        note(card, I18n.text("m_51b4048beb35"), UI.SM);

        UI.Toggle landing = new UI.Toggle(requireContext(), landingEnabled);
        LinearLayout landingRow = UI.row(requireContext());
        LinearLayout landingText = UI.column(requireContext());
        landingText.addView(UI.strong(requireContext(), I18n.text("m_87b31918524a")));
        TextView landingHint = UI.muted(requireContext(),
                I18n.text("m_287303a83357"));
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
        card.addView(primaryAction(I18n.text("m_bcf1ace464e5"), this::saveSite));
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
        ((TextView) text.getChildAt(1)).setText(desc.isEmpty() ? I18n.text("m_c2c8982623de") : desc);
    }

    /**
     * Finds the preview row by its shape rather than by index.
     *
     * The card grew a I18n.text("m_a76fdaad7519") label above it, and anything else added later
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
        card.addView(cardHead(I18n.text("m_f60a2d511962"), I18n.text("m_7e7f12cf948f")));

        preferredInput = UI.input(requireContext(), "cf.090227.xyz");
        preferredInput.setText(preferredDraft);
        card.addView(UI.field(requireContext(), I18n.text("m_71b3aeb8f73c"), preferredInput));
        note(card, I18n.text("m_bb8bd65fc13f"), UI.SM);

        TextView savePreferred = UI.button(requireContext(), I18n.text("m_9e9ef2c6a1ec"), UI.BTN_SECONDARY);
        savePreferred.setOnClickListener(v -> savePreferredCNAME(preferredInput.getText().toString().trim()));
        UI.addRow(card, savePreferred, UI.SM);

        card.addView(UI.divider(requireContext()));

        for (int i = 0; i < presets.size(); i++) {
            card.addView(presetRow(i));
        }

        TextView add = UI.button(requireContext(), I18n.text("m_f47cac452640"), UI.BTN_SECONDARY);
        add.setEnabled(presets.size() < MAX_PRESETS);
        add.setOnClickListener(v -> {
            capture();
            if (presets.size() >= MAX_PRESETS) return;
            presets.add(new String[]{"", ""});
            render();
        });
        UI.addRow(card, add, UI.MD);

        card.addView(UI.spacer(requireContext(), UI.MD));
        card.addView(primaryAction(I18n.text("m_801a8daafa2e"), this::savePresets));
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

        EditText name = UI.input(requireContext(), I18n.text("m_93bc08a447ce"));
        name.setText(presets.get(index)[0]);
        presetNameInputs.add(name);
        box.addView(UI.field(requireContext(), I18n.text("m_2035432e8a18"), name, UI.SM));

        EditText value = UI.input(requireContext(), I18n.text("m_91d03241c65a"));
        value.setText(presets.get(index)[1]);
        presetValueInputs.add(value);
        box.addView(UI.field(requireContext(), I18n.text("m_ce05579d19d5"), value, UI.SM));

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
        card.addView(cardHead(I18n.text("m_119e1fd67f70"), I18n.text("m_855cdae806a9")));

        fallbackInput = UI.input(requireContext(), I18n.text("m_078389cf3046"));
        fallbackInput.setText(fallbackDraft);
        card.addView(fallbackInput);

        TextView save = UI.button(requireContext(), I18n.text("m_0bb58707f90c"), UI.BTN_SECONDARY);
        save.setOnClickListener(v -> saveFallback(fallbackInput.getText().toString().trim()));
        UI.addRow(card, save, UI.MD);
        return card;
    }

    private View tunnelCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(cardHead(I18n.text("m_b903d203f70a"), I18n.text("m_8401fe7d0746")));

        String tunnelId = config.optString("tunnel_id", "");
        String tunnelName = config.optString("tunnel_name", "");
        LinearLayout row = UI.row(requireContext());
        LinearLayout text = UI.column(requireContext());
        text.addView(UI.strong(requireContext(), tunnelId.isEmpty() ? I18n.text("m_0e2cf15a2702")
                : (tunnelName.isEmpty() ? I18n.text("m_8ce5320404d5") : tunnelName)));
        if (!tunnelId.isEmpty()) {
            text.addView(UI.mono(requireContext(), tunnelId, Theme.p().mute));
        }
        row.addView(text, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        TextView go = UI.button(requireContext(), tunnelId.isEmpty() ? I18n.text("m_acfb8f32676c") : I18n.text("m_42b8762044e2"),
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
            statusMessage = I18n.text("m_85c2f8365c82");
            statusOk = false;
            render();
            return;
        }
        if (!beginWrite()) return;
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
            statusMessage = I18n.text("m_1964b400b352");
            statusOk = true;
            render();
            loadIcon();
        }, failure -> {
            busy = false;
            statusMessage = I18n.text("m_6ade7baf8737") + failure.getMessage();
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
            statusMessage = I18n.text("m_5799ba3fe8dc");
            statusOk = false;
            render();
            return;
        }
        if (!beginWrite()) return;
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("value", value);
            return Api.post("/api/config/preferred-cname", payload);
        }, result -> {
            busy = false;
            preferredDraft = value;
            statusMessage = I18n.text("m_20737048a527");
            statusOk = true;
            render();
        }, failure -> {
            busy = false;
            statusMessage = I18n.text("m_6ade7baf8737") + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    private void savePresets() {
        capture();
        for (String[] preset : presets) {
            if (preset[0].trim().isEmpty() || preset[1].trim().isEmpty()) {
                statusMessage = I18n.text("m_ea92a5f61f83");
                statusOk = false;
                render();
                return;
            }
        }
        if (!beginWrite()) return;
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
            statusMessage = I18n.text("m_d12fd29ea414");
            statusOk = true;
            render();
        }, failure -> {
            busy = false;
            statusMessage = I18n.text("m_6ade7baf8737") + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    private void saveFallback(String domain) {
        if (domain.isEmpty()) {
            statusMessage = I18n.text("m_6301facafab9");
            statusOk = false;
            render();
            return;
        }
        if (!beginWrite()) return;
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("domain", domain);
            return Api.post("/api/domain/fallback", payload);
        }, result -> {
            busy = false;
            fallbackDraft = domain;
            statusMessage = result.optString("message", I18n.text("m_b9ed9651aa57"));
            statusOk = true;
            render();
        }, failure -> {
            busy = false;
            statusMessage = I18n.text("m_a3614e8103ec") + failure.getMessage();
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
                statusMessage = I18n.text("m_35fc560913c0");
                statusOk = false;
                render();
                return;
            }
            byte[] bytes = readAll(uri);
            if (bytes.length > MAX_ICON_BYTES) {
                statusMessage = I18n.text("m_78c1f649e77d");
                statusOk = false;
                render();
                return;
            }
            capture();
            iconDraft = "data:" + mime + ";base64," + Base64.encodeToString(bytes, Base64.NO_WRAP);
            iconBitmap = BitmapFactory.decodeByteArray(bytes, 0, bytes.length);
            iconBitmapKey = iconDraft;
            statusMessage = I18n.text("m_5bda44454e75");
            statusOk = true;
            render();
        } catch (Exception e) {
            statusMessage = I18n.text("m_b1feeac12438") + e.getMessage();
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
        enabled = enabled && !busy && !loading && !sysLoading;
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
        input.setSaveEnabled(false);
        input.setImportantForAutofill(View.IMPORTANT_FOR_AUTOFILL_NO);
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

    /** Invalidates any earlier read before submitting a configuration change. */
    private boolean beginWrite() {
        if (!alive() || busy) return false;
        readGeneration++;
        loading = false;
        sysLoading = false;
        busy = true;
        return true;
    }

    /** Runs a write, reloads the system settings, and reports the outcome. */
    private void run(Callable call, String success) {
        if (!beginWrite()) return;
        statusMessage = "";
        render();
        Api.async(call::call, result -> {
            busy = false;
            statusMessage = success;
            statusOk = true;
            loadSys();
        }, failure -> {
            busy = false;
            statusMessage = I18n.text("m_6ade7baf8737") + failure.getMessage();
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
            payload.put("lab_daily_request_limit", labDailyRequestLimit);
            payload.put("lab_requests_per_second", labRequestsPerSecond);
            payload.put("lab_max_workers", labMaxWorkers);
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
            JSONObject saved = Api.put("/api/admin/settings", body);
            Session.applyConfig(saved);
            return saved;
        }, success);
    }

    private View turnstileCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), I18n.text("m_e4f6aa5a185d")));
        TextView hint = UI.muted(requireContext(),
                "可选防护：开启后登录、找回密码需要完成 Cloudflare 人机验证。"
                        + I18n.text("m_078ba7491c8b"));
        UI.margin(hint, 0, UI.XS, 0, UI.MD);
        card.addView(hint);

        UI.addRow(card, toggleRow(I18n.text("m_b05703226c5e"), turnstileEnabled, true, next -> {
            capture();
            turnstileEnabled = next;
            saveAppSettings(I18n.text("m_763363acbbeb"));
        }), 0);

        siteKeyInput = UI.input(requireContext(), "Site Key（0x4A…）");
        siteKeyInput.setText(turnstileSiteKey);
        card.addView(UI.field(requireContext(), "Site Key", siteKeyInput, UI.MD));

        secretKeyInput = passwordInput(I18n.text("m_96656dd2d25f"));
        secretKeyInput.setText(turnstileSecret);
        card.addView(UI.field(requireContext(), "Secret Key", secretKeyInput, UI.MD));

        TextView state = tag(turnstileHasSecret ? I18n.text("m_c0ee21a90d7b") : I18n.text("m_5061a4ea571c"), turnstileHasSecret);
        UI.margin(state, 0, UI.SM, 0, UI.MD);
        card.addView(state);

        TextView save = UI.button(requireContext(), busy ? I18n.text("m_ff509c9ba052") : I18n.text("m_a3030bf8f16d"), UI.BTN_PRIMARY);
        save.setEnabled(!busy && !loading && !sysLoading);
        save.setOnClickListener(v -> saveAppSettings(I18n.text("m_763363acbbeb")));
        UI.addRow(card, save, 0);
        return card;
    }

    private View oauthCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), I18n.text("m_a44454e9d95a")));
        TextView hint = UI.muted(requireContext(),
                I18n.text("m_c634788aa628")
                        + I18n.text("m_5e27d6849810"));
        UI.margin(hint, 0, UI.XS, 0, UI.MD);
        card.addView(hint);

        oauthIdInput = UI.input(requireContext(), "Client ID");
        oauthIdInput.setText(oauthClientId);
        card.addView(UI.field(requireContext(), "Client ID", oauthIdInput));

        oauthSecretInput = passwordInput(I18n.text("m_be5f73c5b889"));
        oauthSecretInput.setText(oauthClientSecret);
        card.addView(UI.field(requireContext(), "Client Secret", oauthSecretInput, UI.MD));

        oauthRedirectInput = UI.input(requireContext(), I18n.text("m_9df3fe96b710"));
        oauthRedirectInput.setText(oauthRedirect);
        card.addView(UI.field(requireContext(), I18n.text("m_9ca160d65985"), oauthRedirectInput, UI.MD));

        oauthScopesInput = UI.input(requireContext(), I18n.text("m_b813dbaced7d"));
        oauthScopesInput.setText(oauthScopes);
        card.addView(UI.field(requireContext(), "Scopes", oauthScopesInput, UI.MD));

        TextView state = tag(oauthHasSecret ? I18n.text("m_c0ee21a90d7b") : I18n.text("m_5061a4ea571c"), oauthHasSecret);
        UI.margin(state, 0, UI.SM, 0, UI.MD);
        card.addView(state);

        TextView save = UI.button(requireContext(), busy ? I18n.text("m_ff509c9ba052") : I18n.text("m_a3030bf8f16d"), UI.BTN_PRIMARY);
        save.setEnabled(!busy && !loading && !sysLoading);
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
        }, I18n.text("m_12c3bd5c0eeb"));
    }

    private View encryptionCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), I18n.text("m_dbf3b43ce49a")));
        TextView hint = UI.muted(requireContext(),
                I18n.text("m_319cbe979eb9")
                        + I18n.text("m_a65c3f6bb334"));
        UI.margin(hint, 0, UI.XS, 0, UI.MD);
        card.addView(hint);

        encKeyField = UI.input(requireContext(), I18n.text("m_6171d4f4500d"));
        encKeyField.setText(encKeyInput);
        card.addView(encKeyField);

        String source = "env".equals(encKeySource) ? I18n.text("m_ae27b474ea4d") : "stored".equals(encKeySource) ? I18n.text("m_263200d7d529") : I18n.text("m_2f5f1d6fbfb0");
        TextView state = tag(I18n.text("m_1f45e225299d") + source, !"none".equals(encKeySource));
        UI.margin(state, 0, UI.SM, 0, UI.MD);
        card.addView(state);

        TextView save = UI.button(requireContext(), busy ? I18n.text("m_ff509c9ba052") : I18n.text("m_a334bb468aba"), UI.BTN_PRIMARY);
        save.setEnabled(!busy && !encKeyInput.trim().isEmpty());
        save.setOnClickListener(v -> saveEncKey());
        UI.addRow(card, save, 0);
        return card;
    }

    private void saveEncKey() {
        capture();
        if (encKeyInput.isEmpty()) {
            statusMessage = I18n.text("m_24d0752eb75d");
            statusOk = false;
            render();
            return;
        }
        String key = encKeyInput;
        encKeyInput = "";
        run(() -> Api.put("/api/admin/encryption-key", new JSONObject().put("key", key)), I18n.text("m_764e8c4abd92"));
    }

    private View passkeySettingsCard() {
        LinearLayout card = titledCard(I18n.text("m_102824e51a9d"),
                I18n.text("m_714c69b463c4"));

        passkeyRPIDInput = UI.input(requireContext(), I18n.text("m_9b9ca2069036"));
        passkeyRPIDInput.setText(passkeyRPID);
        card.addView(UI.field(requireContext(), I18n.text("m_041601fcfdb9"), passkeyRPIDInput, UI.SM));

        passkeyOriginsInput = UI.input(requireContext(), I18n.text("m_9fefc8c169c3"));
        passkeyOriginsInput.setText(passkeyOrigins);
        card.addView(UI.field(requireContext(), I18n.text("m_08976c1b5998"), passkeyOriginsInput, UI.SM));

        androidPackageInput = UI.input(requireContext(), "com.tunnelmanager.app");
        androidPackageInput.setText(androidPackage);
        card.addView(UI.field(requireContext(), I18n.text("m_fdc7245f3cb7"), androidPackageInput, UI.SM));

        androidFingerprintsInput = UI.textArea(requireContext(), I18n.text("m_d83849747d4b"));
        androidFingerprintsInput.setText(androidFingerprints);
        card.addView(UI.field(requireContext(), I18n.text("m_0f65f682db8f"), androidFingerprintsInput, UI.SM));
        card.addView(UI.muted(requireContext(), androidFingerprintsEffective.isEmpty()
                ? I18n.text("m_bd29f1abcca3")
                : I18n.text("m_04f2b844522a") + androidFingerprintsEffective));
        card.addView(UI.muted(requireContext(), I18n.text("m_eba5f2cf8804")));

        TextView save = UI.button(requireContext(), busy ? I18n.text("m_ff509c9ba052") : I18n.text("m_bbf51c405246"), UI.BTN_PRIMARY);
        save.setEnabled(!busy && !loading && !sysLoading);
        save.setOnClickListener(v -> saveAppSettings(I18n.text("m_a50d4226fe2e")));
        UI.margin(save, 0, UI.MD, 0, 0);
        card.addView(save);

        card.addView(UI.divider(requireContext()));
        UI.addRow(card, toggleRow(I18n.text("m_f1d371db2841"), passwordLoginDisabled, passkeyAdminReady, next -> {
            passwordLoginDisabled = next;
            saveAppSettings(next ? I18n.text("m_01b240b5fcfd") : I18n.text("m_f5844ba700c6"));
        }), UI.MD);
        card.addView(UI.muted(requireContext(), passkeyAdminReady
                ? "开启后所有账户都只能用通行密钥登录；账户丢失通行密钥时，可用 CLI 的 --allow-password-login 恢复。"
                : "需管理员已绑定通行密钥，避免面板被锁死。"));
        return card;
    }

    private View rateLimitCard() {
        LinearLayout card = titledCard(I18n.text("m_37c24b911b27"),
                "登录、二次验证、找回与重置密码都受此限制。账号额度是真正的防线，换 IP 绕不过去；IP 额度用来挡住一台机器横扫多个账号。任一触发即返回 429 并附带解锁时间。");

        UI.addRow(card, toggleRow(I18n.text("m_829140bedce5"), rateLimitEnabled, true, next -> {
            rateLimitEnabled = next;
            saveAppSettings(I18n.text("m_ae654df5d8e1"));
        }), UI.SM);

        rateLimitPerAccountInput = signedInput(I18n.text("m_455c4401e923"), rateLimitPerAccount);
        card.addView(UI.field(requireContext(), I18n.text("m_671a08f4e738"), rateLimitPerAccountInput, UI.SM));

        rateLimitPerIPInput = signedInput(I18n.text("m_697a0edd8846"), rateLimitPerIP);
        card.addView(UI.field(requireContext(), I18n.text("m_cf5a28cff90f"), rateLimitPerIPInput, UI.SM));

        rateLimitWindowInput = signedInput(I18n.text("m_5f566691080a"), rateLimitWindow);
        card.addView(UI.field(requireContext(), I18n.text("m_088598d743e5"), rateLimitWindowInput, UI.SM));

        rateLimitFamiliarInput = signedInput(I18n.text("m_e93cb34571fb"), rateLimitFamiliar);
        card.addView(UI.field(requireContext(), I18n.text("m_b9ca02da2aa9"), rateLimitFamiliarInput, UI.SM));
        card.addView(UI.muted(requireContext(), I18n.text("m_8a3dd29e76b4")));

        TextView save = UI.button(requireContext(), busy ? I18n.text("m_ff509c9ba052") : I18n.text("m_a23922612975"), UI.BTN_PRIMARY);
        save.setEnabled(!busy && !loading && !sysLoading);
        save.setOnClickListener(v -> saveAppSettings(I18n.text("m_ae654df5d8e1")));
        UI.margin(save, 0, UI.MD, 0, 0);
        card.addView(save);

        UI.addRow(card, toggleRow(I18n.text("m_ab5fdea0b36d"), rateLimitNotify, true, next -> {
            rateLimitNotify = next;
            saveAppSettings(I18n.text("m_ae654df5d8e1"));
        }), UI.MD);
        card.addView(UI.muted(requireContext(), "走管理员配置的通知渠道（TG / 邮件），并沿用「登录通知」开关。"));
        return card;
    }

    private View experimentalCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), I18n.text("m_b145d5c1a57b")));
        TextView hint = UI.muted(requireContext(),
                I18n.text("m_53053a487f1f"));
        UI.margin(hint, 0, UI.XS, 0, UI.MD);
        card.addView(hint);
        UI.addRow(card, toggleRow(I18n.text("m_06566a8c40f2"), experimentalEnabled, !busy, next -> {
            experimentalEnabled = next;
            saveAppSettings("实验性功能设置已保存");
        }), 0);
        EditText daily = numberInput(I18n.text("m_43daf24d5cfd"), String.valueOf(labDailyRequestLimit));
        EditText rate = numberInput(I18n.text("m_5ed8f4d1306b"), String.valueOf(labRequestsPerSecond));
        EditText maximum = numberInput(I18n.text("m_a071ff7b7d10"), String.valueOf(labMaxWorkers));
        daily.setEnabled(!busy);
        rate.setEnabled(!busy);
        maximum.setEnabled(!busy);
        card.addView(UI.field(requireContext(), I18n.text("m_8b57f87fb23a"), daily, UI.MD));
        card.addView(UI.field(requireContext(), I18n.text("m_b8f8cb247e4d"), rate, UI.MD));
        card.addView(UI.field(requireContext(), I18n.text("m_652a6f861b6d"), maximum, UI.MD));
        TextView save = UI.button(requireContext(), I18n.text("m_eedfd7ecc753"), UI.BTN_PRIMARY);
        save.setEnabled(!busy && !loading && !sysLoading);
        save.setOnClickListener(view -> {
            int dailyValue = parseInt(daily.getText().toString());
            int rateValue = parseInt(rate.getText().toString());
            int maximumValue = parseInt(maximum.getText().toString());
            if (dailyValue < 1 || dailyValue > 10000000 || rateValue < 1 || rateValue > 1000 || maximumValue < 1 || maximumValue > 256) {
                new AlertDialog.Builder(requireContext()).setMessage(I18n.text("m_c2d9dd20948b"))
                        .setPositiveButton(I18n.text("m_de32e20193ad"), null).show();
                return;
            }
            labDailyRequestLimit = dailyValue;
            labRequestsPerSecond = rateValue;
            labMaxWorkers = maximumValue;
            saveAppSettings("优选限制已保存");
        });
        UI.addRow(card, save, UI.MD);
        return card;
    }

    private View smtpCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), I18n.text("m_dd15bdd9da5a")));
        TextView hint = UI.muted(requireContext(),
                I18n.text("slim_75c52d958508")
                        + (smtpReady ? "" : I18n.text("m_f7b32edf785b")));
        UI.margin(hint, 0, UI.XS, 0, UI.MD);
        card.addView(hint);

        smtpHostInput = UI.input(requireContext(), "smtp.example.com");
        smtpHostInput.setText(smtpHost);
        card.addView(UI.field(requireContext(), I18n.text("m_5344a59d11c5"), smtpHostInput));

        smtpPortInput = numberInput(I18n.text("m_7a9d079c1fb3"), smtpPort);
        card.addView(UI.field(requireContext(), I18n.text("m_e71ac32b544b"), smtpPortInput, UI.MD));

        smtpUserInput = UI.input(requireContext(), I18n.text("m_6a432ca72cc9"));
        smtpUserInput.setText(smtpUsername);
        card.addView(UI.field(requireContext(), I18n.text("m_1a3f0617d6de"), smtpUserInput, UI.MD));

        smtpPasswordInput = passwordInput(I18n.text("m_077af8674c6a"));
        smtpPasswordInput.setText(smtpPassword);
        card.addView(UI.field(requireContext(), I18n.text("m_32676619bb95"), smtpPasswordInput, UI.MD));

        smtpFromInput = UI.input(requireContext(), "panel@example.com");
        smtpFromInput.setText(smtpFrom);
        card.addView(UI.field(requireContext(), I18n.text("m_bae32961a3b5"), smtpFromInput, UI.MD));

        TextView tlsLabel = UI.label(requireContext(), I18n.text("m_b31967d52c23"));
        UI.margin(tlsLabel, 0, UI.MD, 0, UI.SM);
        card.addView(tlsLabel);
        UI.addRow(card, UI.segmented(requireContext(), new String[]{I18n.text("m_320c254572c4"), I18n.text("m_69f3b744f3f2")}, smtpTls, index -> {
            capture();
            smtpTls = index;
            render();
        }), 0);

        TextView save = UI.button(requireContext(), busy ? I18n.text("m_ff509c9ba052") : I18n.text("m_c8550237ba70"), UI.BTN_PRIMARY);
        save.setEnabled(!busy && !loading && !sysLoading);
        save.setOnClickListener(v -> saveSmtp());
        UI.addRow(card, save, UI.MD);
        return card;
    }

    private View smtpTestCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), I18n.text("m_f9ae1a31638d")));
        testMailInput = UI.input(requireContext(), I18n.text("m_27558ddc9803"));
        testMailInput.setInputType(InputType.TYPE_TEXT_VARIATION_EMAIL_ADDRESS);
        testMailInput.setText(testMailTo);
        card.addView(UI.field(requireContext(), I18n.text("m_b84b2b30c3d1"), testMailInput));
        TextView send = UI.button(requireContext(), busy ? I18n.text("m_364970d8168c") : I18n.text("m_f9ae1a31638d"), UI.BTN_SECONDARY);
        send.setEnabled(!busy);
        send.setOnClickListener(v -> {
            capture();
            if (testMailTo.isEmpty()) {
                statusMessage = I18n.text("m_2da3d7207b44");
                statusOk = false;
                render();
                return;
            }
            String to = testMailTo;
            run(() -> Api.post("/api/admin/smtp/test", new JSONObject().put("to", to)), I18n.text("m_cfb7989b7a73"));
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
        }, I18n.text("m_dddcb690d105"));
    }
}
