package com.tunnelmanager.app;

import android.content.Context;
import android.graphics.Typeface;
import android.text.InputType;
import android.view.Gravity;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.EditText;
import android.widget.HorizontalScrollView;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.appcompat.app.AlertDialog;

import org.json.JSONArray;
import org.json.JSONObject;

import java.net.URLEncoder;
import java.nio.charset.StandardCharsets;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Date;
import java.util.List;
import java.util.Locale;
import java.util.function.IntConsumer;

/**
 * 管理后台 — users, groups, invites, registration policy and mail.
 *
 * The web page is a five-tab table layout; on a phone the tab strip becomes a
 * horizontally scrolling chip row and every table row becomes a card, because
 * seven columns never survive a 384dp viewport.
 */
public class AdminFragment extends PageFragment {

    private final String[] TABS = {I18n.text("m_0d0e1a86b3aa"), I18n.text("m_963ead08d78d"), I18n.text("m_c8b56dca7aed"), I18n.text("m_68ea5dd4d7af"), I18n.text("m_0ac53099a515"), I18n.text("m_e73d84773a07")};
    /** Audit categories, mirroring the web panel's filter list. */
    private static final String[] AUDIT_CATEGORIES = {
            "", "auth", "passkey", "user", "group", "invite", "settings",
            "tunnel", "domain", "dns", "monitor", "lab", "telegram"};
    private final String[] AUDIT_CATEGORY_LABELS = {
            I18n.text("m_5c55a67935af"), I18n.text("m_bbf58983ccf5"), I18n.text("m_102824e51a9d"), I18n.text("m_fbf413d429bd"), I18n.text("m_963ead08d78d"), I18n.text("m_c8b56dca7aed"), I18n.text("m_68ea5dd4d7af"),
            I18n.text("m_e920f47d0c05"), I18n.text("m_4bf8b66d7390"), I18n.text("m_72af91239cad"), I18n.text("m_7fab82054a8f"), I18n.text("m_3db29474a184"), I18n.text("m_805dcced3f07")};
    private final String[] AUDIT_RANGE_LABELS = {I18n.text("m_5c55a67935af"), I18n.text("m_d5f5a7a01073"), I18n.text("m_2261b06712a3"), I18n.text("m_f729bb3d3f7e")};
    private static final String[] AUDIT_ACTIONS = {
            "login", "login_failed", "logout",
            "passkey_add", "passkey_remove", "passkey_rename", "passkey_login", "passkey_login_failed",
            "password_login_disable", "password_login_enable",
            "password_login_global_disable", "password_login_global_enable",
            "user_create", "user_status", "user_group", "user_password", "user_delete",
            "group_create", "group_update", "group_delete",
            "invite_create", "invite_update", "invite_delete",
            "settings_update", "site_update", "cname_presets_update", "preferred_cname_update",
            "smtp_update", "smtp_test", "oauth_update", "encryption_key_update",
            "tunnel_create", "tunnel_delete", "ingress_add", "ingress_update", "ingress_delete",
            "domain_bind", "domain_bind_batch", "domain_fallback",
            "dns_create", "dns_update", "dns_delete",
            "monitor_create", "monitor_update", "monitor_delete", "monitor_check",
            "target_add", "target_update", "target_delete",
            "lab_settings_update", "lab_run", "lab_stop", "lab_ownership_challenge", "lab_ownership_verify", "lab_ownership_dns", "telegram_endpoint_update"};
    private final String[] AUDIT_ACTION_LABELS = {
            I18n.text("m_645b934deb86"), I18n.text("m_e4343921c928"), I18n.text("m_3ab8cc15939f"),
            I18n.text("m_54255856f81f"), I18n.text("m_fc3f78b499e1"), I18n.text("m_c384f03c2393"), I18n.text("m_5903fbc149e8"), I18n.text("m_9b4340ac3382"),
            I18n.text("m_3a027241650b"), I18n.text("m_466c8603a96a"),
            I18n.text("m_aa2fb06da444"), I18n.text("m_7022fc61d84f"),
            I18n.text("m_37c825ff40e7"), I18n.text("m_088829f635e5"), I18n.text("m_3f18c306c522"), I18n.text("m_69d844482cde"), I18n.text("m_b75872bba6f1"),
            I18n.text("m_f5c0c8360a8a"), I18n.text("m_880dc01d1427"), I18n.text("m_2aacf3a5dbf0"),
            I18n.text("m_5f6a55e484cc"), I18n.text("m_0f69f3cff13d"), I18n.text("m_6f2555f607af"),
            I18n.text("m_e04f5a9c92cb"), I18n.text("m_9c4e82ea3a5e"), I18n.text("m_e3a19ed99df8"), I18n.text("m_33a47993a6eb"),
            I18n.text("m_b6d496d548f8"), I18n.text("m_f9ae1a31638d"), I18n.text("m_30ebe9296b01"), I18n.text("m_026bad67a38d"),
            I18n.text("m_e33080d615cb"), I18n.text("m_cf5e530b1ad3"), I18n.text("m_b6c70706eb7d"), I18n.text("m_9c66533b5540"), I18n.text("m_7d67e8e89593"),
            I18n.text("m_dd799f925f94"), I18n.text("m_d7220322e884"), I18n.text("m_0bb58707f90c"),
            I18n.text("m_d9099a5e05d2"), I18n.text("m_ce46f070601d"), I18n.text("m_ab190f1faa64"),
            I18n.text("m_f9313095ee64"), I18n.text("m_40b947bb3c5e"), I18n.text("m_b2fc749edfa3"), I18n.text("m_579e2b6a5204"),
            I18n.text("m_b8c4303aff45"), I18n.text("m_c21bd0ef300e"), I18n.text("m_bf7241072328"),
            I18n.text("m_247f5a5d0147"), I18n.text("m_15cce67fe3ee"), I18n.text("m_5dc85ec6744a"), I18n.text("m_9a0ffaa81be1"), I18n.text("m_35001ceda70c"), I18n.text("m_eb83a1ede9ac"), I18n.text("m_68dc91ef78e7")};
    private static final String[] INVITE_MODES = {"off", "optional", "required"};
    private final String[] INVITE_LABELS = {I18n.text("m_3fd47edce45b"), I18n.text("m_ebdcbdcf9c99"), I18n.text("m_11da9dc44285")};
    private static final String[] PERMISSIONS = {"tunnels", "domain_bind", "dns", "monitors", "oauth_connect"};
    private final String[] PERMISSION_LABELS = {I18n.text("m_e920f47d0c05"), I18n.text("m_4bf8b66d7390"), I18n.text("m_72af91239cad"), I18n.text("m_7fab82054a8f"), I18n.text("m_1d81613aa5b5")};
    /** User cards shown before the "显示更多" button appears. */
    private static final int PAGE_SIZE = 8;

    private ScrollView scroll;
    private LinearLayout body;
    /** Title and tab strip, pinned above the scrolling content. */
    private LinearLayout header;
    /** Set on a tab change so the new tab opens at the top. */
    private boolean resetScroll = false;

    private boolean loading = true;
    private boolean busy = false;
    private int tab = 0;
    private String banner = "";
    private boolean bannerError = true;

    private JSONArray users = new JSONArray();
    private JSONArray groups = new JSONArray();
    private JSONArray invites = new JSONArray();
    private JSONObject settings = new JSONObject();

    private boolean showCreateUser = false;
    private String userQuery = "";
    private int userShown = PAGE_SIZE;
    private EditText userSearchInput;
    private LinearLayout userListSlot;
    private String newUsername = "";
    private String newEmail = "";
    private String newPassword = "";
    private String newUserGroupId = "";
    private boolean newUserAdmin = false;
    private EditText newUsernameInput;
    private EditText newEmailInput;
    private EditText newPasswordInput;

    private boolean showCreateGroup = false;
    private String newGroupName = "";
    private EditText newGroupNameInput;

    private boolean showCreateInvite = false;
    private String inviteGroupId = "";
    private String inviteMaxUses = "";
    private String inviteExpireDays = "";
    private EditText inviteMaxInput;
    private EditText inviteExpireInput;

    private String turnstileSiteKey = "";
    private String turnstileSecret = "";
    private boolean turnstileHasSecret = false;
    private EditText siteKeyInput;
    private EditText secretKeyInput;

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

    /** Passkey settings, edited on the 系统设置 tab. */
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

	/** Sign-in rate limit, edited on the 系统设置 tab. Blank means "use the
	 *  default", which is how the backend reads a zero. */
	private String rateLimitPerAccount = "0";
	private String rateLimitPerIP = "0";
	private String rateLimitWindow = "0";
	private String rateLimitFamiliar = "0";
	private EditText rateLimitPerAccountInput;
	private EditText rateLimitPerIPInput;
	private EditText rateLimitWindowInput;
	private EditText rateLimitFamiliarInput;

	/** Audit tab state; the trail loads the first time the tab is opened. */
    private static final int AUDIT_PAGE_SIZE = 20;
    private JSONArray auditLogs = new JSONArray();
    private JSONObject auditStats = new JSONObject();
    private int auditTotal = 0;
    private int auditPage = 1;
    private int auditCategory = 0;
    private int auditRange = 0;
    private String auditActor = "";
    private EditText auditActorInput;
    private boolean auditLoading = false;
    private boolean auditLoaded = false;

    @Override
    String route() {
        return "/admin";
    }

    @Override
    protected View build(@NonNull LayoutInflater inflater, @Nullable ViewGroup container) {
        LinearLayout root = UI.column(requireContext());
        header = UI.column(requireContext());
        header.setPadding(UI.dp(UI.LG), UI.dp(UI.SM), UI.dp(UI.LG), 0);
        scroll = new ScrollView(requireContext());
        body = UI.column(requireContext());
        UI.pagePadding(body);
        scroll.addView(body);
        root.addView(header, new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));
        root.addView(scroll, new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, 0, 1f));
        render();
        load();
        return root;
    }

    // ------------------------------------------------------------------- data

    private void load() {
        loading = true;
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("users", Api.get("/api/admin/users"));
            payload.put("groups", Api.get("/api/admin/groups"));
            payload.put("invites", Api.get("/api/admin/invites"));
            payload.put("settings", Api.get("/api/admin/settings"));
            payload.put("smtp", Api.get("/api/admin/smtp"));
            payload.put("oauth", Api.get("/api/admin/oauth"));
            payload.put("key", Api.get("/api/admin/encryption-key"));
            return payload;
        }, payload -> {
            loading = false;
            users = array(payload.optJSONObject("users"), "users");
            groups = array(payload.optJSONObject("groups"), "groups");
            invites = array(payload.optJSONObject("invites"), "invites");
            settings = orEmpty(payload.optJSONObject("settings"));
            Session.applyConfig(settings);
            if (alive()) console().rebuildNav();
            applySmtp(payload.optJSONObject("smtp"));
            applyOauth(payload.optJSONObject("oauth"));
            JSONObject key = payload.optJSONObject("key");
            encKeySource = key == null ? "none" : key.optString("source", "none");
            applySettingsDrafts();
            render();
        }, failure -> {
            loading = false;
            banner = I18n.text("m_01de8216e0d0") + failure.getMessage();
            bannerError = true;
            render();
        });
    }

    private static JSONArray array(JSONObject payload, String key) {
        JSONArray value = payload == null ? null : payload.optJSONArray(key);
        return value == null ? new JSONArray() : value;
    }

    private static JSONObject orEmpty(JSONObject value) {
        return value == null ? new JSONObject() : value;
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

    private void applySettingsDrafts() {
        turnstileSiteKey = settings.optString("turnstile_site_key", "");
        turnstileHasSecret = settings.optBoolean("turnstile_has_secret", false);
        turnstileSecret = "";
        passkeyRPID = settings.optString("passkey_rp_id", "");
        passkeyOrigins = settings.optString("passkey_origins", "");
        androidPackage = settings.optString("passkey_android_package", "");
        androidFingerprints = settings.optString("passkey_android_fingerprints", "");
        JSONArray effective = settings.optJSONArray("passkey_android_fingerprints_effective");
        androidFingerprintsEffective = effective == null ? "" : effective.toString();
        passwordLoginDisabled = settings.optBoolean("password_login_disabled", false);
        passkeyAdminReady = settings.optBoolean("passkey_admin_ready", false);
    }

    // -------------------------------------------------------------- rendering

    private void capture() {
        if (tab == 0) {
            if (newUsernameInput != null) newUsername = newUsernameInput.getText().toString().trim();
            if (newEmailInput != null) newEmail = newEmailInput.getText().toString().trim();
            if (newPasswordInput != null) newPassword = newPasswordInput.getText().toString();
        } else if (tab == 1) {
            if (newGroupNameInput != null) newGroupName = newGroupNameInput.getText().toString().trim();
        } else if (tab == 2) {
            if (inviteMaxInput != null) inviteMaxUses = inviteMaxInput.getText().toString().trim();
            if (inviteExpireInput != null) inviteExpireDays = inviteExpireInput.getText().toString().trim();
        } else if (tab == 3) {
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
        } else {
            if (smtpHostInput != null) smtpHost = smtpHostInput.getText().toString().trim();
            if (smtpPortInput != null) smtpPort = smtpPortInput.getText().toString().trim();
            if (smtpUserInput != null) smtpUsername = smtpUserInput.getText().toString().trim();
            if (smtpPasswordInput != null) smtpPassword = smtpPasswordInput.getText().toString();
            if (smtpFromInput != null) smtpFrom = smtpFromInput.getText().toString().trim();
            if (testMailInput != null) testMailTo = testMailInput.getText().toString().trim();
        }
    }

    private void render() {
        if (body == null || !alive()) return;
        int keepScroll = resetScroll ? 0 : scroll.getScrollY();
        resetScroll = false;
        header.removeAllViews();
        body.removeAllViews();

        header.addView(UI.pageTitle(requireContext(), I18n.text("m_87354adb1c31")));
        TextView subtitle = UI.muted(requireContext(), I18n.text("m_74420630cb1f"));
        UI.margin(subtitle, 0, UI.XS, 0, 0);
        header.addView(subtitle);

        if (loading) {
            body.addView(UI.spacer(requireContext(), UI.MD));
            body.addView(UI.muted(requireContext(), I18n.text("m_4927a53bcc88")));
            return;
        }

        header.addView(tabStrip());

        if (!banner.isEmpty()) {
            body.addView(UI.banner(requireContext(), banner, bannerError));
            body.addView(UI.spacer(requireContext(), UI.MD));
        }

        switch (tab) {
            case 1:
                groupsTab();
                break;
            case 2:
                invitesTab();
                break;
            case 3:
                settingsTab();
                break;
            case 4:
                smtpTab();
                break;
            case 5:
                auditTab();
                break;
            default:
                usersTab();
        }
        UI.restoreScroll(scroll, keepScroll);
    }

    private View tabStrip() {
        HorizontalScrollView scroller = new HorizontalScrollView(requireContext());
        scroller.setHorizontalScrollBarEnabled(false);
        LinearLayout strip = UI.row(requireContext());
        Palette p = Theme.p();
        for (int i = 0; i < TABS.length; i++) {
            final int index = i;
            boolean active = tab == index;
            TextView chip = UI.text(requireContext(), TABS[i], 13,
                    active ? p.ink : p.mute, active ? Typeface.BOLD : Typeface.NORMAL);
            chip.setPadding(UI.dp(10), UI.dp(7), UI.dp(10), UI.dp(7));
            chip.setBackground(active
                    ? UI.roundedStroke(p.canvasRaised, UI.RADIUS_PILL, p.ink, 1)
                    : UI.rounded(p.canvasSoft, UI.RADIUS_PILL));
            chip.setClickable(true);
            chip.setOnClickListener(v -> {
                capture();
                if (tab != index) resetScroll = true;
                tab = index;
                showCreateUser = false;
                showCreateGroup = false;
                showCreateInvite = false;
                banner = "";
                render();
            });
            LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(
                    ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);
            if (i > 0) lp.leftMargin = UI.dp(6);
            strip.addView(chip, lp);
        }
        scroller.addView(strip);
        return scroller;
    }

    /** A card with a heading and an optional action button on the right. */
    private LinearLayout cardHead(String title, String desc, String action, Runnable onClick) {
        LinearLayout card = UI.card(requireContext());
        LinearLayout head = UI.row(requireContext());
        head.addView(UI.cardTitle(requireContext(), title),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        if (action != null) {
            TextView button = UI.button(requireContext(), action, UI.BTN_SECONDARY);
            button.setOnClickListener(v -> onClick.run());
            head.addView(button);
        }
        card.addView(head);
        if (desc != null && !desc.isEmpty()) {
            TextView hint = UI.muted(requireContext(), desc);
            UI.margin(hint, 0, UI.XS, 0, UI.MD);
            card.addView(hint);
        }
        return card;
    }

    private EditText numberInput(String hint, String value) {
        EditText input = UI.input(requireContext(), hint);
        input.setInputType(InputType.TYPE_CLASS_NUMBER);
        input.setText(value);
        return input;
    }

    private EditText passwordInput(String hint) {
        EditText input = UI.input(requireContext(), hint);
        input.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_PASSWORD);
        return input;
    }

    /** A read-only box that opens a sheet; the phone stand-in for a select. */
    private View picker(String label, String value, Runnable onClick) {
        LinearLayout box = UI.row(requireContext());
        box.setBackground(UI.pressable(
                UI.roundedStroke(Theme.p().canvas, UI.RADIUS_MD, Theme.p().hairline, 1),
                Theme.p().btnGhostHover));
        box.setPadding(UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD));
        box.setClickable(true);
        box.setOnClickListener(v -> onClick.run());
        TextView text = UI.body(requireContext(), value);
        text.setTextColor(value.isEmpty() ? Theme.p().mute : Theme.p().ink);
        box.addView(text, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        box.addView(UI.text(requireContext(), "›", 18, Theme.p().mute, Typeface.NORMAL));
        return UI.field(requireContext(), label, box);
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

    private interface ToggleListener {
        void onChange(boolean next);
    }

    private TextView tag(String label, boolean ok) {
        return UI.tag(requireContext(), label, ok);
    }

    private static String formatTime(long seconds) {
        if (seconds <= 0) return "—";
        return new SimpleDateFormat("yyyy-MM-dd HH:mm", I18n.locale()).format(new Date(seconds * 1000L));
    }

    // --------------------------------------------------------------- users tab

    private void usersTab() {
        LinearLayout card = cardHead(I18n.text("m_a4660b1813b3"), null, showCreateUser ? I18n.text("m_afd4b783536b") : I18n.text("m_3ae9a33489d7"), () -> {
            capture();
            showCreateUser = !showCreateUser;
            render();
        });

        if (showCreateUser) {
            newUsernameInput = UI.input(requireContext(), I18n.text("m_1a3f0617d6de"));
            newUsernameInput.setText(newUsername);
            card.addView(UI.field(requireContext(), I18n.text("m_1a3f0617d6de"), newUsernameInput));

            newEmailInput = UI.input(requireContext(), I18n.text("m_504ebd6e7929"));
            newEmailInput.setInputType(InputType.TYPE_TEXT_VARIATION_EMAIL_ADDRESS);
            newEmailInput.setText(newEmail);
            card.addView(UI.field(requireContext(), I18n.text("m_73075237fd0f"), newEmailInput, UI.MD));

            newPasswordInput = passwordInput(I18n.text("m_f155bd296642"));
            newPasswordInput.setText(newPassword);
            card.addView(UI.field(requireContext(), I18n.text("m_df387080f986"), newPasswordInput, UI.MD));

            card.addView(picker(I18n.text("m_963ead08d78d"), groupName(newUserGroupId), this::pickUserGroup));
            UI.margin(card.getChildAt(card.getChildCount() - 1), 0, UI.MD, 0, 0);

            UI.addRow(card, toggleRow(I18n.text("m_e19796712f1c"), newUserAdmin, true, next -> {
                capture();
                newUserAdmin = next;
                render();
            }), UI.MD);

            TextView create = UI.button(requireContext(), busy ? I18n.text("m_56b63def0e29") : I18n.text("m_cde2cd071d25"), UI.BTN_PRIMARY);
            create.setEnabled(!busy);
            create.setOnClickListener(v -> submitCreateUser());
            UI.margin(create, 0, UI.MD, 0, 0);
            card.addView(create);
            card.addView(UI.divider(requireContext()));
        }

        userSearchInput = UI.input(requireContext(), I18n.text("m_5645a9f7777c"));
        userSearchInput.setText(userQuery);
        userSearchInput.addTextChangedListener(new Watcher(() -> {
            userQuery = userSearchInput.getText().toString().trim();
            userShown = PAGE_SIZE;
            renderUserList();
        }));
        UI.addRow(card, userSearchInput, showCreateUser ? UI.MD : 0);

        userListSlot = UI.column(requireContext());
        UI.addRow(card, userListSlot, UI.MD);
        renderUserList();
        body.addView(card);
    }

    /**
     * Repaints the user cards in place.
     *
     * Rebuilding the whole tab on every keystroke would tear down the search
     * field under the operator's cursor, so only the list below it is replaced.
     */
    private void renderUserList() {
        if (userListSlot == null) return;
        userListSlot.removeAllViews();

        List<JSONObject> matches = new ArrayList<>();
        for (int i = 0; i < users.length(); i++) {
            JSONObject user = users.optJSONObject(i);
            if (user == null) continue;
            if (matchesQuery(user, userQuery)) matches.add(user);
        }

        String caption = userQuery.isEmpty()
                ? I18n.text("m_2428420e407f") + users.length() + I18n.text("m_9e1d21da836c")
                : I18n.text("m_2b0dba4e5ce7") + matches.size() + " / " + users.length() + I18n.text("m_9e1d21da836c");
        userListSlot.addView(UI.muted(requireContext(), caption));

        if (matches.isEmpty()) {
            userListSlot.addView(emptyState(I18n.text("m_def2b4425033")));
            return;
        }

        int limit = Math.min(userShown, matches.size());
        for (int i = 0; i < limit; i++) {
            UI.addRow(userListSlot, userRow(matches.get(i)), UI.MD);
        }
        if (matches.size() > limit) {
            TextView more = UI.button(requireContext(),
                    I18n.text("m_8e4cffaddc6a") + (matches.size() - limit) + I18n.text("m_0e5129abb2bf"), UI.BTN_SECONDARY);
            more.setOnClickListener(v -> {
                userShown += PAGE_SIZE;
                renderUserList();
            });
            UI.addRow(userListSlot, more, UI.MD);
        }
    }

    private static boolean matchesQuery(JSONObject user, String query) {
        if (query.isEmpty()) return true;
        String needle = query.toLowerCase(Locale.ROOT);
        return user.optString("username").toLowerCase(Locale.ROOT).contains(needle)
                || user.optString("email").toLowerCase(Locale.ROOT).contains(needle)
                || user.optString("nickname").toLowerCase(Locale.ROOT).contains(needle);
    }

    /** A no-op-editable TextWatcher: only the "after" edge matters here. */
    private static final class Watcher implements android.text.TextWatcher {
        private final Runnable onChanged;

        Watcher(Runnable onChanged) {
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

    private View userRow(JSONObject user) {
        LinearLayout box = UI.column(requireContext());
        box.setBackground(UI.roundedStroke(Theme.p().canvasSoft2, UI.RADIUS_MD, Theme.p().hairline, 1));
        box.setPadding(UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD));

        LinearLayout top = UI.row(requireContext());
        top.addView(UI.strong(requireContext(), user.optString("username")),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        boolean admin = "admin".equals(user.optString("role"));
        top.addView(tag(admin ? I18n.text("m_e19796712f1c") : I18n.text("m_0d0e1a86b3aa"), admin));
        UI.margin(top.getChildAt(top.getChildCount() - 1), UI.SM, 0, 0, 0);
        boolean active = "active".equals(user.optString("status"));
        top.addView(tag(active ? I18n.text("m_296de0e31f8c") : I18n.text("m_bc5a87a757a5"), active));
        UI.margin(top.getChildAt(top.getChildCount() - 1), UI.XS, 0, 0, 0);
        int passkeys = user.optInt("passkeys", 0);
        top.addView(tag(passkeys > 0 ? passkeys + I18n.text("m_6857f925f760") : I18n.text("m_61467aa358ae"), passkeys > 0));
        UI.margin(top.getChildAt(top.getChildCount() - 1), UI.XS, 0, 0, 0);
        if (user.optBoolean("password_login_disabled", false)) {
            top.addView(tag(I18n.text("m_b7a378e767eb"), false));
            UI.margin(top.getChildAt(top.getChildCount() - 1), UI.XS, 0, 0, 0);
        }
        box.addView(top);

        List<String> meta = new ArrayList<>();
        String email = user.optString("email", "");
        meta.add(email.isEmpty() ? I18n.text("m_563e0e1dc68d") : email);
        meta.add(I18n.text("m_24b09cb8e1f2") + defaulted(groupName(user.optString("group_id"))));
        meta.add(I18n.text("m_125fd739288e") + formatTime(user.optLong("last_login_at", 0)));
        TextView caption = UI.muted(requireContext(), join(meta, " · "));
        UI.margin(caption, 0, UI.XS, 0, UI.SM);
        box.addView(caption);

        TextView manage = UI.button(requireContext(), I18n.text("m_bb6d995724f4"), UI.BTN_SECONDARY);
        manage.setOnClickListener(v -> userActions(user));
        LinearLayout.LayoutParams mlp = new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);
        box.addView(manage, mlp);
        return box;
    }

    private void userActions(JSONObject user) {
        String username = user.optString("username");
        boolean active = "active".equals(user.optString("status"));
        Sheet.of(requireContext(), username)
                .item(R.drawable.ic_nav_logout, active ? I18n.text("m_7943ebc6d9a9") : I18n.text("m_11ad8210bc60"), () -> run(
                        () -> Api.put("/api/admin/users/" + user.optString("id") + "/status",
                                new JSONObject().put("status", active ? "disabled" : "active")),
                        active ? I18n.text("m_bc5a87a757a5") : I18n.text("m_dfb802238b38")))
                .item(R.drawable.ic_nav_admin, I18n.text("m_92df516e1593"), () -> pickGroupFor(user))
                .item(R.drawable.ic_nav_edit, I18n.text("m_d70928debc2e"), () -> promptResetPassword(user))
                .item(R.drawable.ic_nav_logout, user.optBoolean("password_login_disabled", false)
                                ? I18n.text("m_466c8603a96a") : I18n.text("m_3a027241650b"),
                        () -> setUserPasswordLogin(user, !user.optBoolean("password_login_disabled", false)))
                .item(R.drawable.ic_nav_trash, I18n.text("m_b75872bba6f1"), () -> confirmRemoveUser(user))
                .show();
    }

    private void pickGroupFor(JSONObject user) {
        List<String> labels = new ArrayList<>();
        List<String> ids = new ArrayList<>();
        labels.add(I18n.text("m_e69c528d68a8"));
        ids.add("");
        for (int i = 0; i < groups.length(); i++) {
            JSONObject group = groups.optJSONObject(i);
            if (group == null) continue;
            labels.add(group.optString("name"));
            ids.add(group.optString("id"));
        }
        Sheet.Builder sheet = Sheet.of(requireContext(), I18n.text("m_92df516e1593")).label(user.optString("username"));
        for (int i = 0; i < labels.size(); i++) {
            final String id = ids.get(i);
            sheet.item(R.drawable.ic_nav_admin, labels.get(i), () -> run(
                    () -> Api.put("/api/admin/users/" + user.optString("id") + "/group",
                            new JSONObject().put("group_id", id)),
                    I18n.text("m_9afb25dec846")));
        }
        sheet.show();
    }

    private void promptResetPassword(JSONObject user) {
        EditText field = passwordInput(I18n.text("m_84c2c26ef24b"));
        final String[] value = {""};
        Modal.of(requireContext(), I18n.text("m_d70928debc2e"))
                .message(I18n.text("m_49c073f4fa15") + user.optString("username") + I18n.text("m_4b2bc6b2c569"))
                .content(field)
                .cancel(I18n.text("m_2cd0f3be8738"))
                .confirm(I18n.text("m_cb5d682bac3d"), false, () -> {
                    value[0] = field.getText().toString();
                    if (value[0].length() < 6) {
                        banner = I18n.text("m_ac31ae58b658");
                        bannerError = true;
                        render();
                        return;
                    }
                    run(() -> Api.put("/api/admin/users/" + user.optString("id") + "/password",
                                    new JSONObject().put("new_password", value[0])),
                            I18n.text("m_4c5c1892de86"));
                })
                .show();
    }

    private void confirmRemoveUser(JSONObject user) {
        Modal.of(requireContext(), I18n.text("m_b75872bba6f1"))
                .message(I18n.text("m_6fa91af954cd") + user.optString("username") + I18n.text("m_26ff3d7383ae"))
                .cancel(I18n.text("m_2cd0f3be8738"))
                .confirm(I18n.text("m_2f9daa828907"), true, () -> run(
                        () -> Api.delete("/api/admin/users/" + user.optString("id")),
                        I18n.text("m_7ad39d9c27ab")))
                .show();
    }

    private void submitCreateUser() {
        capture();
        if (newUsername.isEmpty() || newPassword.length() < 6) {
            banner = I18n.text("m_21ee040fad34");
            bannerError = true;
            render();
            return;
        }
        run(() -> {
            JSONObject payload = new JSONObject();
            payload.put("username", newUsername);
            if (!newEmail.isEmpty()) payload.put("email", newEmail);
            payload.put("password", newPassword);
            payload.put("role", newUserAdmin ? "admin" : "user");
            if (!newUserGroupId.isEmpty()) payload.put("group_id", newUserGroupId);
            return Api.post("/api/admin/users", payload);
        }, I18n.text("m_d4f7d0bf6262"), () -> {
            newUsername = "";
            newEmail = "";
            newPassword = "";
            newUserGroupId = "";
            newUserAdmin = false;
            showCreateUser = false;
        });
    }

    private void pickUserGroup() {
        List<String> labels = new ArrayList<>();
        List<String> ids = new ArrayList<>();
        labels.add(I18n.text("m_e69c528d68a8"));
        ids.add("");
        for (int i = 0; i < groups.length(); i++) {
            JSONObject group = groups.optJSONObject(i);
            if (group == null) continue;
            labels.add(group.optString("name"));
            ids.add(group.optString("id"));
        }
        Sheet.Builder sheet = Sheet.of(requireContext(), I18n.text("m_faa794fa59d6"));
        for (int i = 0; i < labels.size(); i++) {
            final String id = ids.get(i);
            sheet.item(R.drawable.ic_nav_admin, labels.get(i), () -> {
                newUserGroupId = id;
                render();
            });
        }
        sheet.show();
    }

    // -------------------------------------------------------------- groups tab

    private void groupsTab() {
        LinearLayout card = cardHead(I18n.text("m_963ead08d78d"), groups.length() + I18n.text("m_26fe2eb629d5"),
                showCreateGroup ? I18n.text("m_afd4b783536b") : I18n.text("m_e05eb0ea9c94"), () -> {
                    capture();
                    showCreateGroup = !showCreateGroup;
                    render();
                });

        if (showCreateGroup) {
            newGroupNameInput = UI.input(requireContext(), I18n.text("m_744e71a6afc1"));
            newGroupNameInput.setText(newGroupName);
            card.addView(UI.field(requireContext(), I18n.text("m_744e71a6afc1"), newGroupNameInput));
            TextView create = UI.button(requireContext(), busy ? I18n.text("m_56b63def0e29") : I18n.text("m_cde2cd071d25"), UI.BTN_PRIMARY);
            create.setEnabled(!busy);
            create.setOnClickListener(v -> {
                capture();
                if (newGroupName.isEmpty()) return;
                run(() -> Api.post("/api/admin/groups",
                                new JSONObject().put("name", newGroupName).put("permissions", new JSONArray())),
                        I18n.text("m_a44fb45d0f39"), () -> {
                            newGroupName = "";
                            showCreateGroup = false;
                        });
            });
            UI.margin(create, 0, UI.MD, 0, 0);
            card.addView(create);
            card.addView(UI.divider(requireContext()));
        }
        body.addView(card);

        for (int i = 0; i < groups.length(); i++) {
            JSONObject group = groups.optJSONObject(i);
            if (group == null) continue;
            body.addView(UI.spacer(requireContext(), UI.MD));
            body.addView(groupCard(group));
        }
    }

    private View groupCard(JSONObject group) {
        LinearLayout card = UI.card(requireContext());
        LinearLayout top = UI.row(requireContext());
        top.addView(UI.cardTitle(requireContext(), group.optString("name")),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        if (group.optBoolean("builtin", false)) top.addView(tag(I18n.text("m_95e35aabd9a9"), false));
        card.addView(top);

        TextView count = UI.muted(requireContext(), groupMemberCount(group.optString("id")) + I18n.text("m_b786ec8651f4"));
        UI.margin(count, 0, UI.XS, 0, UI.MD);
        card.addView(count);

        JSONArray perms = group.optJSONArray("permissions");
        List<String> granted = new ArrayList<>();
        if (perms != null) {
            for (int i = 0; i < perms.length(); i++) granted.add(perms.optString(i));
        }
        for (int i = 0; i < PERMISSIONS.length; i++) {
            final String perm = PERMISSIONS[i];
            boolean on = granted.contains(perm);
            UI.addRow(card, toggleRow(PERMISSION_LABELS[i], on, !busy, next -> {
                List<String> next2 = new ArrayList<>(granted);
                if (next) {
                    if (!next2.contains(perm)) next2.add(perm);
                } else {
                    next2.remove(perm);
                }
                JSONArray payload = new JSONArray();
                for (String value : next2) payload.put(value);
                run(() -> Api.put("/api/admin/groups/" + group.optString("id"),
                                new JSONObject().put("name", group.optString("name")).put("permissions", payload)),
                        I18n.text("m_d72fbc805af9"));
            }), i == 0 ? 0 : UI.SM);
        }

        LinearLayout actions = UI.row(requireContext());
        TextView rename = UI.button(requireContext(), I18n.text("m_0d0cbac2eee5"), UI.BTN_SECONDARY);
        rename.setOnClickListener(v -> promptRenameGroup(group));
        UI.weight(rename, 1f);
        actions.addView(rename);
        if (!group.optBoolean("builtin", false)) {
            TextView remove = UI.button(requireContext(), I18n.text("m_2f9daa828907"), UI.BTN_DANGER);
            remove.setOnClickListener(v -> confirmRemoveGroup(group));
            UI.weight(remove, 1f);
            UI.margin(remove, UI.SM, 0, 0, 0);
            actions.addView(remove);
        }
        UI.addRow(card, actions, UI.MD);
        return card;
    }

    private int groupMemberCount(String id) {
        int total = 0;
        for (int i = 0; i < users.length(); i++) {
            JSONObject user = users.optJSONObject(i);
            if (user != null && id.equals(user.optString("group_id"))) total++;
        }
        return total;
    }

    private void promptRenameGroup(JSONObject group) {
        if (group.optBoolean("builtin", false)) {
            banner = I18n.text("m_e29e29f7c5d3");
            bannerError = true;
            render();
            return;
        }
        EditText field = UI.input(requireContext(), I18n.text("m_e08ae7e4a8bf"));
        field.setText(group.optString("name"));
        Modal.of(requireContext(), I18n.text("m_26d99b409a65"))
                .content(field)
                .cancel(I18n.text("m_2cd0f3be8738"))
                .confirm(I18n.text("m_a3030bf8f16d"), false, () -> {
                    String name = field.getText().toString().trim();
                    if (name.isEmpty() || name.equals(group.optString("name"))) return;
                    JSONArray perms = group.optJSONArray("permissions");
                    run(() -> Api.put("/api/admin/groups/" + group.optString("id"),
                                    new JSONObject().put("name", name)
                                            .put("permissions", perms == null ? new JSONArray() : perms)),
                            I18n.text("m_06afd1f0352e"));
                })
                .show();
    }

    private void confirmRemoveGroup(JSONObject group) {
        Modal.of(requireContext(), I18n.text("m_2aacf3a5dbf0"))
                .message(I18n.text("m_5a398d79b24c") + group.optString("name") + "？")
                .cancel(I18n.text("m_2cd0f3be8738"))
                .confirm(I18n.text("m_2f9daa828907"), true, () -> run(
                        () -> Api.delete("/api/admin/groups/" + group.optString("id")),
                        I18n.text("m_2194537e8be2")))
                .show();
    }

    // ------------------------------------------------------------- invites tab

    private void invitesTab() {
        LinearLayout card = cardHead(I18n.text("m_c8b56dca7aed"), invites.length() + I18n.text("m_5d17f5a39658"),
                showCreateInvite ? I18n.text("m_afd4b783536b") : I18n.text("m_5f6a55e484cc"), () -> {
                    capture();
                    showCreateInvite = !showCreateInvite;
                    render();
                });

        if (showCreateInvite) {
            card.addView(picker(I18n.text("m_963ead08d78d"), groupName(inviteGroupId), this::pickInviteGroup));
            UI.margin(card.getChildAt(card.getChildCount() - 1), 0, UI.MD, 0, 0);

            inviteMaxInput = numberInput(I18n.text("m_56d8df3e3345"), inviteMaxUses);
            card.addView(UI.field(requireContext(), I18n.text("m_62162c91428a"), inviteMaxInput, UI.MD));

            inviteExpireInput = numberInput(I18n.text("m_c3aa8b7da12e"), inviteExpireDays);
            card.addView(UI.field(requireContext(), I18n.text("m_5a52b819b316"), inviteExpireInput, UI.MD));

            TextView create = UI.button(requireContext(), busy ? I18n.text("m_43d2661b36be") : I18n.text("m_1ad1463fe16f"), UI.BTN_PRIMARY);
            create.setEnabled(!busy);
            create.setOnClickListener(v -> submitCreateInvite());
            UI.margin(create, 0, UI.MD, 0, 0);
            card.addView(create);
            card.addView(UI.divider(requireContext()));
        }

        if (invites.length() == 0) {
            card.addView(emptyState(I18n.text("m_bb1faeb3c558")));
        }
        for (int i = 0; i < invites.length(); i++) {
            JSONObject invite = invites.optJSONObject(i);
            if (invite == null) continue;
            UI.addRow(card, inviteRow(invite), i == 0 && !showCreateInvite ? 0 : UI.MD);
        }
        body.addView(card);
    }

    private View inviteRow(JSONObject invite) {
        LinearLayout box = UI.column(requireContext());
        box.setBackground(UI.roundedStroke(Theme.p().canvasSoft2, UI.RADIUS_MD, Theme.p().hairline, 1));
        box.setPadding(UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD));

        LinearLayout top = UI.row(requireContext());
        top.addView(UI.mono(requireContext(), invite.optString("code"), Theme.p().ink),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        boolean on = invite.optBoolean("enabled", false);
        top.addView(tag(on ? I18n.text("m_f4f0ead1116b") : I18n.text("m_4e6fd0e28c55"), on));
        box.addView(top);

        int max = invite.optInt("max_uses", 0);
        long expires = invite.optLong("expires_at", 0);
        String meta = I18n.text("m_24b09cb8e1f2") + defaulted(groupName(invite.optString("group_id")))
                + I18n.text("m_136485364403") + invite.optInt("used_count", 0) + " / " + (max > 0 ? String.valueOf(max) : "∞")
                + " · " + (expires > 0 ? formatTime(expires) : I18n.text("m_3e71ccc89a43"));
        TextView caption = UI.muted(requireContext(), meta);
        UI.margin(caption, 0, UI.XS, 0, UI.SM);
        box.addView(caption);

        LinearLayout actions = UI.row(requireContext());
        TextView toggle = UI.button(requireContext(), on ? I18n.text("m_4e6fd0e28c55") : I18n.text("m_f4f0ead1116b"), UI.BTN_SECONDARY);
        toggle.setOnClickListener(v -> run(
                () -> Api.put("/api/admin/invites/" + invite.optString("code"),
                        new JSONObject().put("enabled", !on)),
                on ? I18n.text("m_a8c3698b5b8c") : I18n.text("m_dfb802238b38")));
        UI.weight(toggle, 1f);
        actions.addView(toggle);
        TextView remove = UI.button(requireContext(), I18n.text("m_2f9daa828907"), UI.BTN_DANGER);
        remove.setOnClickListener(v -> confirmRemoveInvite(invite));
        UI.weight(remove, 1f);
        UI.margin(remove, UI.SM, 0, 0, 0);
        actions.addView(remove);
        box.addView(actions);
        return box;
    }

    private void pickInviteGroup() {
        List<String> labels = new ArrayList<>();
        List<String> ids = new ArrayList<>();
        labels.add(I18n.text("m_e69c528d68a8"));
        ids.add("");
        for (int i = 0; i < groups.length(); i++) {
            JSONObject group = groups.optJSONObject(i);
            if (group == null) continue;
            labels.add(group.optString("name"));
            ids.add(group.optString("id"));
        }
        Sheet.Builder sheet = Sheet.of(requireContext(), I18n.text("m_faa794fa59d6"));
        for (int i = 0; i < labels.size(); i++) {
            final String id = ids.get(i);
            sheet.item(R.drawable.ic_nav_admin, labels.get(i), () -> {
                inviteGroupId = id;
                render();
            });
        }
        sheet.show();
    }

    private void submitCreateInvite() {
        capture();
        run(() -> {
            JSONObject payload = new JSONObject();
            if (!inviteGroupId.isEmpty()) payload.put("group_id", inviteGroupId);
            payload.put("max_uses", parseInt(inviteMaxUses));
            int days = parseInt(inviteExpireDays);
            payload.put("expires_at", days > 0 ? System.currentTimeMillis() / 1000L + (long) days * 86400L : 0);
            return Api.post("/api/admin/invites", payload);
        }, I18n.text("m_67021bc43cbd"), () -> {
            showCreateInvite = false;
            inviteMaxUses = "";
            inviteExpireDays = "";
        });
    }

    private void confirmRemoveInvite(JSONObject invite) {
        Modal.of(requireContext(), I18n.text("m_6f2555f607af"))
                .message(I18n.text("m_759aeb16d395") + invite.optString("code") + "？")
                .cancel(I18n.text("m_2cd0f3be8738"))
                .confirm(I18n.text("m_2f9daa828907"), true, () -> run(
                        () -> Api.delete("/api/admin/invites/" + invite.optString("code")),
                        I18n.text("m_c714402bd321")))
                .show();
    }

    private static int parseInt(String raw) {
        try {
            return Integer.parseInt(raw.trim());
        } catch (NumberFormatException e) {
            return 0;
        }
    }

    // ------------------------------------------------------------ settings tab

    private void settingsTab() {
        body.addView(registrationCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(experimentalCard());
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
    }

    /**
     * Sign-in rate limiting. The account budget is the real defence — it cannot
     * be shed by changing address — while the address budget stops one machine
     * working through a list of accounts.
     */
    private View rateLimitCard() {
	Context ctx = requireContext();
	LinearLayout card = cardHead(I18n.text("m_37c24b911b27"),
		I18n.text("m_305ae157faf7"),
		null, null);

	UI.addRow(card, toggleRow(I18n.text("m_829140bedce5"), settings.optBoolean("rate_limit_enabled", false), true, next -> {
	settingsPut("rate_limit_enabled", next);
	saveSettings();
	}), UI.SM);

	rateLimitPerAccountInput = numberInput(ctx, I18n.text("m_455c4401e923"), rateLimitPerAccount);
	card.addView(UI.field(ctx, I18n.text("m_671a08f4e738"), rateLimitPerAccountInput, UI.SM));

	rateLimitPerIPInput = numberInput(ctx, I18n.text("m_697a0edd8846"), rateLimitPerIP);
	card.addView(UI.field(ctx, I18n.text("m_cf5a28cff90f"), rateLimitPerIPInput, UI.SM));

	rateLimitWindowInput = numberInput(ctx, I18n.text("m_5f566691080a"), rateLimitWindow);
	card.addView(UI.field(ctx, I18n.text("m_088598d743e5"), rateLimitWindowInput, UI.SM));

	rateLimitFamiliarInput = numberInput(ctx, I18n.text("m_e93cb34571fb"), rateLimitFamiliar);
	card.addView(UI.field(ctx, I18n.text("m_b9ca02da2aa9"), rateLimitFamiliarInput, UI.SM));
	card.addView(UI.muted(ctx, I18n.text("m_8a3dd29e76b4")));

	TextView save = UI.button(ctx, busy ? I18n.text("m_ff509c9ba052") : I18n.text("m_a23922612975"), UI.BTN_PRIMARY);
	save.setEnabled(!busy);
	save.setOnClickListener(v -> saveRateLimitSettings());
	UI.margin(save, 0, UI.MD, 0, 0);
	card.addView(save);

	UI.addRow(card, toggleRow(I18n.text("m_ab5fdea0b36d"), settings.optBoolean("rate_limit_notify", false), true, next -> {
	settingsPut("rate_limit_notify", next);
	saveSettings();
	}), UI.MD);
	card.addView(UI.muted(ctx, I18n.text("m_3decc654cc2a")));
	return card;
    }

    /** A signed numeric field; the limit accepts a negative for "no limit". */
    private EditText numberInput(Context ctx, String hint, String value) {
	EditText input = UI.input(ctx, hint);
	input.setInputType(InputType.TYPE_CLASS_NUMBER | InputType.TYPE_NUMBER_FLAG_SIGNED);
	input.setText(value);
	return input;
    }

    private void saveRateLimitSettings() {
	capture();
	JSONObject payload = settingsWithPasskeys(null);
	busy = true;
	banner = "";
	render();
	Api.async(() -> Api.put("/api/admin/settings", payload), result -> {
	busy = false;
	banner = I18n.text("m_ae654df5d8e1");
	bannerError = false;
	load();
	}, failure -> {
	busy = false;
	banner = I18n.text("m_84bd3b614686") + failure.getMessage();
	bannerError = true;
	render();
	});
    }

    private boolean regOpen() {
        return settings.optBoolean("registration_enabled", true);
    }

    private boolean emailVerifyEnabled() {
        return !settings.optBoolean("email_verify_disabled", false);
    }

    private View registrationCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), I18n.text("m_5c4ef9d52d09")));

        UI.addRow(card, toggleRow(I18n.text("m_04f83ab519b2"), regOpen(), true, next -> {
            settings.remove("registration_enabled");
            settingsPut("registration_enabled", next);
            saveSettings();
        }), UI.MD);

        boolean verify = emailVerifyEnabled();
        boolean canVerify = smtpReady && regOpen();
        UI.addRow(card, toggleRow(I18n.text("m_25cece360c1c"), verify, canVerify, next -> {
            settingsPut("email_verify_disabled", !next);
            saveSettings();
        }), UI.MD);
        if (!smtpReady) {
            TextView hint = UI.muted(requireContext(), I18n.text("m_47587cbad367"));
            UI.margin(hint, 0, UI.XS, 0, 0);
            card.addView(hint);
        }

        TextView inviteLabel = UI.label(requireContext(), I18n.text("m_c8b56dca7aed"));
        UI.margin(inviteLabel, 0, UI.MD, 0, UI.SM);
        card.addView(inviteLabel);
        int mode = indexOf(INVITE_MODES, settings.optString("invite_mode", "off"));
        UI.addRow(card, UI.segmented(requireContext(), INVITE_LABELS, mode, index -> {
            settingsPut("invite_mode", INVITE_MODES[index]);
            saveSettings();
        }), 0);

        card.addView(picker(I18n.text("m_e69c528d68a8"), defaulted(groupName(settings.optString("default_group_id", ""))),
                this::pickDefaultGroup));
        UI.margin(card.getChildAt(card.getChildCount() - 1), 0, UI.MD, 0, 0);
        return card;
    }

    private View experimentalCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), I18n.text("m_b145d5c1a57b")));
        TextView hint = UI.muted(requireContext(),
                I18n.text("m_53053a487f1f"));
        UI.margin(hint, 0, UI.XS, 0, UI.MD);
        card.addView(hint);
        UI.addRow(card, toggleRow(I18n.text("m_06566a8c40f2"),
                settings.optBoolean("experimental_features_enabled", false), !busy, next -> {
                    settingsPut("experimental_features_enabled", next);
                    saveSettings();
                }), 0);
        EditText daily = numberInput(requireContext(), I18n.text("m_43daf24d5cfd"), String.valueOf(settings.optInt("lab_daily_request_limit", 100000)));
        EditText rate = numberInput(requireContext(), I18n.text("m_5ed8f4d1306b"), String.valueOf(settings.optInt("lab_requests_per_second", 10)));
        EditText maximum = numberInput(requireContext(), I18n.text("m_a071ff7b7d10"), String.valueOf(settings.optInt("lab_max_workers", 32)));
        daily.setEnabled(!busy);
        rate.setEnabled(!busy);
        maximum.setEnabled(!busy);
        card.addView(UI.field(requireContext(), I18n.text("m_8b57f87fb23a"), daily, UI.MD));
        card.addView(UI.field(requireContext(), I18n.text("m_b8f8cb247e4d"), rate, UI.MD));
        card.addView(UI.field(requireContext(), I18n.text("m_652a6f861b6d"), maximum, UI.MD));
        TextView save = UI.button(requireContext(), I18n.text("m_eedfd7ecc753"), UI.BTN_PRIMARY);
        save.setEnabled(!busy);
        save.setOnClickListener(view -> {
            int dailyValue = parseInt(daily.getText().toString());
            int rateValue = parseInt(rate.getText().toString());
            int maximumValue = parseInt(maximum.getText().toString());
            if (dailyValue < 1 || dailyValue > 10000000 || rateValue < 1 || rateValue > 1000 || maximumValue < 1 || maximumValue > 256) {
                new AlertDialog.Builder(requireContext()).setMessage(I18n.text("m_c2d9dd20948b"))
                        .setPositiveButton(I18n.text("m_de32e20193ad"), null).show();
                return;
            }
            settingsPut("lab_daily_request_limit", dailyValue);
            settingsPut("lab_requests_per_second", rateValue);
            settingsPut("lab_max_workers", maximumValue);
            saveSettings();
        });
        UI.addRow(card, save, UI.MD);
        return card;
    }

    private View turnstileCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), I18n.text("m_e4f6aa5a185d")));
        TextView hint = UI.muted(requireContext(),
                I18n.text("m_4a6fdf9e9abc")
                        + I18n.text("m_078ba7491c8b"));
        UI.margin(hint, 0, UI.XS, 0, UI.MD);
        card.addView(hint);

        UI.addRow(card, toggleRow(I18n.text("m_b05703226c5e"), settings.optBoolean("turnstile_enabled", false), true, next -> {
            capture();
            turnstileSiteKey = siteKeyInput == null ? turnstileSiteKey : siteKeyInput.getText().toString().trim();
            settingsPut("turnstile_enabled", next);
            saveTurnstile();
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
        save.setEnabled(!busy);
        save.setOnClickListener(v -> saveTurnstile());
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
        save.setEnabled(!busy);
        save.setOnClickListener(v -> saveOauth());
        UI.addRow(card, save, 0);
        return card;
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

    private void pickDefaultGroup() {
        List<String> labels = new ArrayList<>();
        List<String> ids = new ArrayList<>();
        labels.add(I18n.text("m_ccb91ce53df9"));
        ids.add("");
        for (int i = 0; i < groups.length(); i++) {
            JSONObject group = groups.optJSONObject(i);
            if (group == null) continue;
            labels.add(group.optString("name"));
            ids.add(group.optString("id"));
        }
        Sheet.Builder sheet = Sheet.of(requireContext(), I18n.text("m_e69c528d68a8"));
        for (int i = 0; i < labels.size(); i++) {
            final String id = ids.get(i);
            sheet.item(R.drawable.ic_nav_admin, labels.get(i), () -> {
                settingsPut("default_group_id", id);
                saveSettings();
            });
        }
        sheet.show();
    }

    private void settingsPut(String key, Object value) {
        try {
            settings.put(key, value);
        } catch (org.json.JSONException ignored) {
            // JSONObject.put only throws on NaN/Infinity; nothing to do here.
        }
    }

    private void saveSettings() {
        run(() -> Api.put("/api/admin/settings", settings), I18n.text("m_bd03c0aad701"), () -> {
            Session.applyConfig(settings);
            if (alive()) console().rebuildNav();
        });
    }

    private void saveTurnstile() {
        capture();
        JSONObject payload = turnstilePayload();
        turnstileSecret = "";
        run(() -> Api.put("/api/admin/settings", payload), I18n.text("m_763363acbbeb"));
    }

    private JSONObject turnstilePayload() {
        try {
            settings.put("turnstile_site_key", turnstileSiteKey);
            if (!turnstileSecret.isEmpty()) settings.put("turnstile_secret", turnstileSecret);
        } catch (org.json.JSONException ignored) {
            // See settingsPut.
        }
        return settings;
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

    private void saveEncKey() {
        capture();
        if (encKeyInput.isEmpty()) {
            banner = I18n.text("m_24d0752eb75d");
            bannerError = true;
            render();
            return;
        }
        String key = encKeyInput;
        encKeyInput = "";
        run(() -> Api.put("/api/admin/encryption-key", new JSONObject().put("key", key)), I18n.text("m_764e8c4abd92"));
    }

    // ----------------------------------------------------------------- smtp

    private void smtpTab() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), I18n.text("m_dd15bdd9da5a")));
        TextView hint = UI.muted(requireContext(),
                I18n.text("m_25ed71be1ba5")
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
        save.setEnabled(!busy);
        save.setOnClickListener(v -> saveSmtp());
        UI.addRow(card, save, UI.MD);
        body.addView(card);

        body.addView(UI.spacer(requireContext(), UI.MD));

        LinearLayout test = UI.card(requireContext());
        test.addView(UI.cardTitle(requireContext(), I18n.text("m_f9ae1a31638d")));
        testMailInput = UI.input(requireContext(), I18n.text("m_27558ddc9803"));
        testMailInput.setInputType(InputType.TYPE_TEXT_VARIATION_EMAIL_ADDRESS);
        testMailInput.setText(testMailTo);
        test.addView(UI.field(requireContext(), I18n.text("m_b84b2b30c3d1"), testMailInput));
        TextView send = UI.button(requireContext(), busy ? I18n.text("m_364970d8168c") : I18n.text("m_f9ae1a31638d"), UI.BTN_SECONDARY);
        send.setEnabled(!busy);
        send.setOnClickListener(v -> {
            capture();
            if (testMailTo.isEmpty()) {
                banner = I18n.text("m_2da3d7207b44");
                bannerError = true;
                render();
                return;
            }
            String to = testMailTo;
            run(() -> Api.post("/api/admin/smtp/test", new JSONObject().put("to", to)), I18n.text("m_cfb7989b7a73"));
        });
        UI.addRow(test, send, UI.MD);
        body.addView(test);
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

    // ---------------------------------------------------------------- helpers

    /** Runs a write, refreshes everything, and reports the outcome. */
    private void run(Callable call, String success) {
        run(call, success, null);
    }

    private void run(Callable call, String success, Runnable after) {
        busy = true;
        banner = "";
        render();
        Api.async(call::call, result -> {
            busy = false;
            banner = success;
            bannerError = false;
            if (after != null) after.run();
            load();
        }, failure -> {
            busy = false;
            banner = failure.getMessage();
            bannerError = true;
            render();
        });
    }

    private interface Callable {
        JSONObject call() throws Exception;
    }

    // --------------------------------------------------------------- passkeys

    /**
     * Relying party, Android asset links and the panel-wide password switch.
     *
     * The panel refuses to switch password sign-in off until an active
     * administrator has a passkey, so the toggle stays disabled until the server
     * says it is safe.
     */
    private View passkeySettingsCard() {
        Context ctx = requireContext();
        LinearLayout card = cardHead(I18n.text("m_102824e51a9d"),
                I18n.text("m_714c69b463c4"),
                null, null);

        passkeyRPIDInput = UI.input(ctx, I18n.text("m_9b9ca2069036"));
        passkeyRPIDInput.setText(passkeyRPID);
        card.addView(UI.field(ctx, I18n.text("m_041601fcfdb9"), passkeyRPIDInput, UI.SM));

        passkeyOriginsInput = UI.input(ctx, I18n.text("m_9fefc8c169c3"));
        passkeyOriginsInput.setText(passkeyOrigins);
        card.addView(UI.field(ctx, I18n.text("m_08976c1b5998"), passkeyOriginsInput, UI.SM));

        androidPackageInput = UI.input(ctx, "com.tunnelmanager.app");
        androidPackageInput.setText(androidPackage);
        card.addView(UI.field(ctx, I18n.text("m_fdc7245f3cb7"), androidPackageInput, UI.SM));

        androidFingerprintsInput = UI.textArea(ctx, I18n.text("m_d83849747d4b"));
        androidFingerprintsInput.setText(androidFingerprints);
        card.addView(UI.field(ctx, I18n.text("m_0f65f682db8f"), androidFingerprintsInput, UI.SM));
        card.addView(UI.muted(ctx, androidFingerprintsEffective.isEmpty()
                ? I18n.text("m_bd29f1abcca3")
                : I18n.text("m_04f2b844522a") + androidFingerprintsEffective));
        card.addView(UI.muted(ctx, I18n.text("m_eba5f2cf8804")));

        TextView save = UI.button(ctx, busy ? I18n.text("m_ff509c9ba052") : I18n.text("m_bbf51c405246"), UI.BTN_PRIMARY);
        save.setEnabled(!busy);
        save.setOnClickListener(v -> savePasskeySettings());
        UI.margin(save, 0, UI.MD, 0, 0);
        card.addView(save);

        card.addView(UI.divider(ctx));
        UI.addRow(card, toggleRow(I18n.text("m_f1d371db2841"), passwordLoginDisabled, passkeyAdminReady,
                next -> setGlobalPasswordLogin(next)), UI.MD);
        card.addView(UI.muted(ctx, passkeyAdminReady
                ? I18n.text("m_d0e92917ca4d")
                : I18n.text("m_6be5de420252")));
        return card;
    }

    private void savePasskeySettings() {
        capture();
        JSONObject payload = settingsWithPasskeys(null);
        busy = true;
        banner = "";
        render();
        Api.async(() -> Api.put("/api/admin/settings", payload), result -> {
            busy = false;
            banner = I18n.text("m_a50d4226fe2e");
            bannerError = false;
            load();
        }, failure -> {
            busy = false;
            banner = failure.getMessage();
            bannerError = true;
            load();
        });
    }

    private void setGlobalPasswordLogin(boolean disabled) {
        capture();
        JSONObject payload = settingsWithPasskeys(disabled);
        busy = true;
        banner = "";
        render();
        Api.async(() -> Api.put("/api/admin/settings", payload), result -> {
            busy = false;
            banner = disabled ? I18n.text("m_01b240b5fcfd") : I18n.text("m_f5844ba700c6");
            bannerError = false;
            load();
        }, failure -> {
            busy = false;
            banner = failure.getMessage();
            bannerError = true;
            // Re-read instead of trusting the local switch: the server refused it.
            load();
        });
    }

    /**
     * The cached settings document plus the edited passkey fields.
     *
     * The endpoint replaces the whole document, so every other field travels
     * with it; a null password switch keeps whatever the server already has.
     */
    private JSONObject settingsWithPasskeys(Boolean passwordLoginOff) {
        JSONObject payload;
        try {
            payload = new JSONObject(settings.toString());
        } catch (Exception e) {
            payload = new JSONObject();
        }
        put(payload, "passkey_rp_id", passkeyRPID);
        put(payload, "passkey_origins", passkeyOrigins);
        put(payload, "passkey_android_package", androidPackage);
        put(payload, "passkey_android_fingerprints", androidFingerprints);
		if (passwordLoginOff != null) put(payload, "password_login_disabled", passwordLoginOff);
		// Carried on every save for the same reason as the passkey fields: a
		// partial save must not reset them to zero, which would read as "use the
		// default" rather than "leave alone".
		put(payload, "rate_limit_per_account", draftInt(rateLimitPerAccount));
		put(payload, "rate_limit_per_ip", draftInt(rateLimitPerIP));
		put(payload, "rate_limit_window_minutes", draftInt(rateLimitWindow));
		put(payload, "rate_limit_familiar_multiplier", draftInt(rateLimitFamiliar));
		return payload;
    }

    /** Reads a draft into an int; anything unparseable falls back to the
     *  backend default, which is what a zero means. */
    private static int draftInt(String value) {
		try {
			return Integer.parseInt(value.trim());
		} catch (NumberFormatException e) {
			return 0;
		}
    }

    private static void put(JSONObject target, String key, Object value) {
        try {
            target.put(key, value);
        } catch (org.json.JSONException ignored) {
            // JSONObject.put only throws on NaN/Infinity.
        }
    }

    /** Restores or disables password sign-in for one account. */
    private void setUserPasswordLogin(JSONObject user, boolean disabled) {
        run(() -> Api.put("/api/admin/users/" + user.optString("id") + "/password-login",
                        new JSONObject().put("disabled", disabled)),
                disabled ? I18n.text("m_648081722ae3") : I18n.text("m_9a51c792b3f2"));
    }

    // ----------------------------------------------------------------- audit

    private void auditTab() {
        if (!auditLoaded) {
            // First visit: fetch the trail, which renders again on arrival.
            auditLoaded = true;
            loadAudit();
            return;
        }
        Context ctx = requireContext();
        body.addView(auditStatsCard());
        body.addView(UI.spacer(ctx, UI.MD));
        body.addView(auditFilterCard());
        body.addView(UI.spacer(ctx, UI.MD));
        body.addView(auditListCard());
    }

    private View auditStatsCard() {
        Context ctx = requireContext();
        LinearLayout card = cardHead(I18n.text("m_2e44c1f92d1d"), I18n.text("m_4d7fff331f6b"), null, null);
        LinearLayout row = UI.row(ctx);
        row.addView(auditStat(ctx, String.valueOf(auditStats.optInt("total", 0)), I18n.text("m_ed31fbb483ee")));
        row.addView(auditStat(ctx, String.valueOf(auditStats.optInt("failed", 0)), I18n.text("m_28384d7afd2e")));
        row.addView(auditStat(ctx, String.valueOf(auditStats.optInt("today", 0)), I18n.text("m_5c9cf5e0dd47")));
        row.addView(auditStat(ctx, String.valueOf(auditStats.optInt("actors", 0)), I18n.text("m_3d8f6f5474e3")));
        card.addView(row);
        return card;
    }

    private View auditStat(Context ctx, String value, String label) {
        LinearLayout box = UI.column(ctx);
        box.addView(UI.text(ctx, value, 20, Theme.p().ink, Typeface.BOLD));
        box.addView(UI.muted(ctx, label));
        UI.weight(box, 1f);
        return box;
    }

    private View auditFilterCard() {
        Context ctx = requireContext();
        LinearLayout card = cardHead(I18n.text("m_b5f15473fdc0"), null, I18n.text("m_bcd6771e08ec"), () -> {
            auditPage = 1;
            loadAudit();
        });

        auditActorInput = UI.input(ctx, I18n.text("m_e412e5321b6d"));
        auditActorInput.setText(auditActor);
        UI.addRow(card, auditActorInput, UI.SM);

        UI.addRow(card, auditPicker(ctx, I18n.text("m_07eb730af899"), AUDIT_RANGE_LABELS[auditRange],
            I18n.text("m_ca06fe65bc35"), AUDIT_RANGE_LABELS, index -> {
            auditRange = index;
            auditPage = 1;
            loadAudit();
        }), UI.MD);

        UI.addRow(card, auditPicker(ctx, I18n.text("m_1c80d4a064ec"), AUDIT_CATEGORY_LABELS[auditCategory],
            I18n.text("m_2cebc060d512"), AUDIT_CATEGORY_LABELS, index -> {
            auditCategory = index;
            auditPage = 1;
            loadAudit();
        }), UI.MD);

        return card;
    }

    /**
     * A labelled field that opens a bottom sheet rather than a dropdown.
     *
     * The web panel can afford an inline select; on a phone the same two lists
     * as chips and a segmented control pushed the log itself below the fold.
     * Every other picker in the app (TTL, DNS type, user group) is a sheet, so
     * these follow that instead.
     */
    private View auditPicker(Context ctx, String label, String current, String sheetTitle,
                             String[] options, IntConsumer onPick) {
        TextView button = UI.button(ctx, current, UI.BTN_SECONDARY);
        button.setOnClickListener(v -> {
            Sheet.Builder sheet = Sheet.of(ctx, sheetTitle);
            for (int i = 0; i < options.length; i++) {
                        final int index = i;
                        sheet.item(R.drawable.ic_nav_check, options[i], () -> onPick.accept(index));
            }
            sheet.show();
        });
        return UI.field(ctx, label, button);
    }

    private View auditListCard() {
        Context ctx = requireContext();
        LinearLayout card = cardHead(I18n.text("m_7dbac1c20f23"), null, I18n.text("m_aee887434131"), () -> loadAudit());
        if (auditLoading) {
            card.addView(UI.muted(ctx, I18n.text("m_4927a53bcc88")));
            return card;
        }
        if (auditLogs.length() == 0) {
            card.addView(emptyState(I18n.text("m_5f1178aa4d21")));
            return card;
        }
        int pages = Math.max(1, (auditTotal + AUDIT_PAGE_SIZE - 1) / AUDIT_PAGE_SIZE);
        card.addView(UI.muted(ctx, I18n.text("m_2428420e407f") + auditTotal + I18n.text("m_37fcd76651b7") + auditPage + " / " + pages + I18n.text("m_85ca81cf6abd")));
        for (int i = 0; i < auditLogs.length(); i++) {
            JSONObject entry = auditLogs.optJSONObject(i);
            if (entry == null) continue;
            UI.addRow(card, auditEntryRow(entry), UI.MD);
        }

        LinearLayout pager = UI.row(ctx);
        TextView prev = UI.button(ctx, I18n.text("m_c9b9ae7a6144"), UI.BTN_SECONDARY);
        prev.setEnabled(auditPage > 1);
        prev.setOnClickListener(v -> {
            auditPage--;
            loadAudit();
        });
        pager.addView(prev);
        TextView next = UI.button(ctx, I18n.text("m_8a8542f69648"), UI.BTN_SECONDARY);
        next.setEnabled(auditPage < pages);
        next.setOnClickListener(v -> {
            auditPage++;
            loadAudit();
        });
        UI.margin(next, UI.SM, 0, 0, 0);
        pager.addView(next);
        UI.addRow(card, pager, UI.MD);
        return card;
    }

    private View auditEntryRow(JSONObject entry) {
        Context ctx = requireContext();
        LinearLayout box = UI.column(ctx);
        box.setBackground(UI.roundedStroke(Theme.p().canvasSoft2, UI.RADIUS_MD, Theme.p().hairline, 1));
        box.setPadding(UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD));

        boolean success = entry.optBoolean("success", true);
        LinearLayout top = UI.row(ctx);
        top.addView(UI.strong(ctx, auditActionLabel(entry.optString("action", ""))),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        top.addView(tag(success ? I18n.text("m_053461ce86d2") : I18n.text("m_28384d7afd2e"), success));
        box.addView(top);

        List<String> meta = new ArrayList<>();
        meta.add(I18n.text("m_b0a12efde7fd") + defaulted(entry.optString("actor_name", "")));
        String target = entry.optString("target", "");
        if (!target.isEmpty()) meta.add(I18n.text("m_4fa5ed9eb23d") + target);
        String ip = entry.optString("ip", "");
        if (!ip.isEmpty()) meta.add("IP：" + ip);
        TextView caption = UI.muted(ctx, join(meta, " · "));
        UI.margin(caption, 0, UI.XS, 0, 0);
        box.addView(caption);
        TextView when = UI.mono(ctx, formatTime(entry.optLong("created_at", 0)), Theme.p().mute);
        UI.margin(when, 0, UI.XS, 0, 0);
        box.addView(when);
        return box;
    }

    private void loadAudit() {
        if (auditActorInput != null) auditActor = auditActorInput.getText().toString().trim();
        auditLoading = true;
        banner = "";
        render();

        final String actor = auditActor;
        final String category = AUDIT_CATEGORIES[auditCategory];
        final long from = auditFrom();
        final int page = auditPage;
        Api.async(() -> {
            StringBuilder path = new StringBuilder("/api/admin/audit-logs?page=").append(page)
                    .append("&page_size=").append(AUDIT_PAGE_SIZE);
            if (!actor.isEmpty()) {
                path.append("&actor=").append(URLEncoder.encode(actor, StandardCharsets.UTF_8));
            }
            if (!category.isEmpty()) path.append("&category=").append(category);
            if (from > 0) path.append("&from=").append(from);
            JSONObject payload = new JSONObject();
            payload.put("logs", Api.get(path.toString()));
            payload.put("stats", Api.get("/api/admin/audit-logs/stats?days=7"));
            return payload;
        }, payload -> {
            auditLoading = false;
            JSONObject logs = orEmpty(payload.optJSONObject("logs"));
            auditLogs = array(logs, "logs");
            auditTotal = logs.optInt("total", 0);
            auditPage = Math.max(1, logs.optInt("page", 1));
            auditStats = orEmpty(payload.optJSONObject("stats"));
            render();
        }, failure -> {
            auditLoading = false;
            banner = I18n.text("m_32ef02f91bfc") + failure.getMessage();
            bannerError = true;
            render();
        });
    }

    /** The "from" second for the selected range, in local time. */
    private long auditFrom() {
        if (auditRange == 0) return 0;
        java.util.Calendar calendar = java.util.Calendar.getInstance();
        if (auditRange == 1) {
            calendar.set(java.util.Calendar.HOUR_OF_DAY, 0);
            calendar.set(java.util.Calendar.MINUTE, 0);
            calendar.set(java.util.Calendar.SECOND, 0);
            calendar.set(java.util.Calendar.MILLISECOND, 0);
        } else {
            calendar.add(java.util.Calendar.DAY_OF_YEAR, auditRange == 2 ? -7 : -30);
        }
        return calendar.getTimeInMillis() / 1000L;
    }

    private String auditActionLabel(String action) {
        for (int i = 0; i < AUDIT_ACTIONS.length; i++) {
            if (AUDIT_ACTIONS[i].equals(action)) return AUDIT_ACTION_LABELS[i];
        }
        return action.isEmpty() ? I18n.text("m_f242d2db4cea") : action;
    }

    private View emptyState(String message) {
        TextView view = UI.muted(requireContext(), message);
        view.setGravity(Gravity.CENTER);
        view.setPadding(0, UI.dp(UI.LG), 0, UI.dp(UI.LG));
        return view;
    }

    private String groupName(String id) {
        if (id == null || id.isEmpty()) return "";
        for (int i = 0; i < groups.length(); i++) {
            JSONObject group = groups.optJSONObject(i);
            if (group != null && id.equals(group.optString("id"))) return group.optString("name");
        }
        return "";
    }

    private static String defaulted(String value) {
        return value == null || value.isEmpty() ? I18n.text("m_844b8cc8dff7") : value;
    }

    private static String join(List<String> parts, String glue) {
        StringBuilder out = new StringBuilder();
        for (String part : parts) {
            if (out.length() > 0) out.append(glue);
            out.append(part);
        }
        return out.toString();
    }

    private static int indexOf(String[] values, String want) {
        for (int i = 0; i < values.length; i++) {
            if (values[i].equals(want)) return i;
        }
        return 0;
    }
}
