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

    private static final String[] TABS = {"用户", "用户组", "邀请码", "系统设置", "邮件服务", "审计"};
    /** Audit categories, mirroring the web panel's filter list. */
    private static final String[] AUDIT_CATEGORIES = {
            "", "auth", "passkey", "user", "group", "invite", "settings",
            "tunnel", "domain", "dns", "monitor", "lab", "telegram"};
    private static final String[] AUDIT_CATEGORY_LABELS = {
            "全部", "登录认证", "通行密钥", "用户管理", "用户组", "邀请码", "系统设置",
            "隧道管理", "域名绑定", "DNS 记录", "服务监控", "IP 优选实验室", "TG 机器人"};
    private static final String[] AUDIT_RANGE_LABELS = {"全部", "今天", "近 7 天", "近 30 天"};
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
            "lab_settings_update", "lab_run", "telegram_endpoint_update"};
    private static final String[] AUDIT_ACTION_LABELS = {
            "登录成功", "登录失败", "退出登录",
            "绑定通行密钥", "删除通行密钥", "重命名通行密钥", "通行密钥登录", "通行密钥登录失败",
            "禁用密码登录", "恢复密码登录",
            "全局禁用密码登录", "全局恢复密码登录",
            "创建用户", "启用 / 禁用用户", "调整用户组", "重置用户密码", "删除用户",
            "创建用户组", "修改用户组", "删除用户组",
            "生成邀请码", "启用 / 停用邀请码", "删除邀请码",
            "修改注册策略", "修改站点设置", "修改 CNAME 预设", "修改优选 CNAME",
            "修改邮件服务", "发送测试邮件", "修改 OAuth 客户端", "修改加密密钥",
            "创建隧道", "删除隧道", "新增隧道规则", "修改隧道规则", "删除隧道规则",
            "绑定域名", "批量绑定域名", "设置回退源",
            "新增 DNS 记录", "修改 DNS 记录", "删除 DNS 记录",
            "创建监控项目", "修改监控项目", "删除监控项目", "手动检测监控",
            "新增监控目标", "修改监控目标", "删除监控目标",
            "修改实验室设置", "执行 IP 优选", "修改 TG API 端点"};
    private static final String[] INVITE_MODES = {"off", "optional", "required"};
    private static final String[] INVITE_LABELS = {"关闭", "选填", "必填"};
    private static final String[] PERMISSIONS = {"tunnels", "domain_bind", "dns", "monitors", "oauth_connect"};
    private static final String[] PERMISSION_LABELS = {"隧道管理", "域名绑定", "DNS 记录", "服务监控", "Cloudflare 授权"};
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
            applySmtp(payload.optJSONObject("smtp"));
            applyOauth(payload.optJSONObject("oauth"));
            JSONObject key = payload.optJSONObject("key");
            encKeySource = key == null ? "none" : key.optString("source", "none");
            applySettingsDrafts();
            render();
        }, failure -> {
            loading = false;
            banner = "加载失败：" + failure.getMessage();
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

        header.addView(UI.pageTitle(requireContext(), "管理后台"));
        TextView subtitle = UI.muted(requireContext(), "管理用户、用户组、邀请码与注册策略");
        UI.margin(subtitle, 0, UI.XS, 0, 0);
        header.addView(subtitle);

        if (loading) {
            body.addView(UI.spacer(requireContext(), UI.MD));
            body.addView(UI.muted(requireContext(), "加载中…"));
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
        return new SimpleDateFormat("yyyy-MM-dd HH:mm", Locale.getDefault()).format(new Date(seconds * 1000L));
    }

    // --------------------------------------------------------------- users tab

    private void usersTab() {
        LinearLayout card = cardHead("用户列表", null, showCreateUser ? "收起" : "新建用户", () -> {
            capture();
            showCreateUser = !showCreateUser;
            render();
        });

        if (showCreateUser) {
            newUsernameInput = UI.input(requireContext(), "用户名");
            newUsernameInput.setText(newUsername);
            card.addView(UI.field(requireContext(), "用户名", newUsernameInput));

            newEmailInput = UI.input(requireContext(), "邮箱（可留空）");
            newEmailInput.setInputType(InputType.TYPE_TEXT_VARIATION_EMAIL_ADDRESS);
            newEmailInput.setText(newEmail);
            card.addView(UI.field(requireContext(), "邮箱", newEmailInput, UI.MD));

            newPasswordInput = passwordInput("初始密码（≥6 位）");
            newPasswordInput.setText(newPassword);
            card.addView(UI.field(requireContext(), "初始密码", newPasswordInput, UI.MD));

            card.addView(picker("用户组", groupName(newUserGroupId), this::pickUserGroup));
            UI.margin(card.getChildAt(card.getChildCount() - 1), 0, UI.MD, 0, 0);

            UI.addRow(card, toggleRow("管理员", newUserAdmin, true, next -> {
                capture();
                newUserAdmin = next;
                render();
            }), UI.MD);

            TextView create = UI.button(requireContext(), busy ? "创建中…" : "创建", UI.BTN_PRIMARY);
            create.setEnabled(!busy);
            create.setOnClickListener(v -> submitCreateUser());
            UI.margin(create, 0, UI.MD, 0, 0);
            card.addView(create);
            card.addView(UI.divider(requireContext()));
        }

        userSearchInput = UI.input(requireContext(), "搜索用户名或邮箱");
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
                ? "共 " + users.length() + " 个账户"
                : "匹配 " + matches.size() + " / " + users.length() + " 个账户";
        userListSlot.addView(UI.muted(requireContext(), caption));

        if (matches.isEmpty()) {
            userListSlot.addView(emptyState("没有匹配的用户。"));
            return;
        }

        int limit = Math.min(userShown, matches.size());
        for (int i = 0; i < limit; i++) {
            UI.addRow(userListSlot, userRow(matches.get(i)), UI.MD);
        }
        if (matches.size() > limit) {
            TextView more = UI.button(requireContext(),
                    "显示更多（还有 " + (matches.size() - limit) + " 个）", UI.BTN_SECONDARY);
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
        top.addView(tag(admin ? "管理员" : "用户", admin));
        UI.margin(top.getChildAt(top.getChildCount() - 1), UI.SM, 0, 0, 0);
        boolean active = "active".equals(user.optString("status"));
        top.addView(tag(active ? "正常" : "已禁用", active));
        UI.margin(top.getChildAt(top.getChildCount() - 1), UI.XS, 0, 0, 0);
        int passkeys = user.optInt("passkeys", 0);
        top.addView(tag(passkeys > 0 ? passkeys + " 个通行密钥" : "仅密码", passkeys > 0));
        UI.margin(top.getChildAt(top.getChildCount() - 1), UI.XS, 0, 0, 0);
        if (user.optBoolean("password_login_disabled", false)) {
            top.addView(tag("已禁用密码", false));
            UI.margin(top.getChildAt(top.getChildCount() - 1), UI.XS, 0, 0, 0);
        }
        box.addView(top);

        List<String> meta = new ArrayList<>();
        String email = user.optString("email", "");
        meta.add(email.isEmpty() ? "未填邮箱" : email);
        meta.add("用户组：" + defaulted(groupName(user.optString("group_id"))));
        meta.add("最近登录：" + formatTime(user.optLong("last_login_at", 0)));
        TextView caption = UI.muted(requireContext(), join(meta, " · "));
        UI.margin(caption, 0, UI.XS, 0, UI.SM);
        box.addView(caption);

        TextView manage = UI.button(requireContext(), "管理", UI.BTN_SECONDARY);
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
                .item(R.drawable.ic_nav_logout, active ? "禁用账户" : "启用账户", () -> run(
                        () -> Api.put("/api/admin/users/" + user.optString("id") + "/status",
                                new JSONObject().put("status", active ? "disabled" : "active")),
                        active ? "已禁用" : "已启用"))
                .item(R.drawable.ic_nav_admin, "更换用户组", () -> pickGroupFor(user))
                .item(R.drawable.ic_nav_edit, "重置密码", () -> promptResetPassword(user))
                .item(R.drawable.ic_nav_logout, user.optBoolean("password_login_disabled", false)
                                ? "恢复密码登录" : "禁用密码登录",
                        () -> setUserPasswordLogin(user, !user.optBoolean("password_login_disabled", false)))
                .item(R.drawable.ic_nav_trash, "删除用户", () -> confirmRemoveUser(user))
                .show();
    }

    private void pickGroupFor(JSONObject user) {
        List<String> labels = new ArrayList<>();
        List<String> ids = new ArrayList<>();
        labels.add("默认用户组");
        ids.add("");
        for (int i = 0; i < groups.length(); i++) {
            JSONObject group = groups.optJSONObject(i);
            if (group == null) continue;
            labels.add(group.optString("name"));
            ids.add(group.optString("id"));
        }
        Sheet.Builder sheet = Sheet.of(requireContext(), "更换用户组").label(user.optString("username"));
        for (int i = 0; i < labels.size(); i++) {
            final String id = ids.get(i);
            sheet.item(R.drawable.ic_nav_admin, labels.get(i), () -> run(
                    () -> Api.put("/api/admin/users/" + user.optString("id") + "/group",
                            new JSONObject().put("group_id", id)),
                    "用户组已更新"));
        }
        sheet.show();
    }

    private void promptResetPassword(JSONObject user) {
        EditText field = passwordInput("新密码（≥6 位）");
        final String[] value = {""};
        Modal.of(requireContext(), "重置密码")
                .message("为 " + user.optString("username") + " 设置新密码，保存后该用户会被登出。")
                .content(field)
                .cancel("取消")
                .confirm("重置", false, () -> {
                    value[0] = field.getText().toString();
                    if (value[0].length() < 6) {
                        banner = "密码至少 6 位";
                        bannerError = true;
                        render();
                        return;
                    }
                    run(() -> Api.put("/api/admin/users/" + user.optString("id") + "/password",
                                    new JSONObject().put("new_password", value[0])),
                            "密码已重置，该用户已被登出");
                })
                .show();
    }

    private void confirmRemoveUser(JSONObject user) {
        Modal.of(requireContext(), "删除用户")
                .message("确定删除 " + user.optString("username") + "？该操作不可恢复。")
                .cancel("取消")
                .confirm("删除", true, () -> run(
                        () -> Api.delete("/api/admin/users/" + user.optString("id")),
                        "用户已删除"))
                .show();
    }

    private void submitCreateUser() {
        capture();
        if (newUsername.isEmpty() || newPassword.length() < 6) {
            banner = "用户名必填，密码至少 6 位";
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
        }, "用户已创建", () -> {
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
        labels.add("默认用户组");
        ids.add("");
        for (int i = 0; i < groups.length(); i++) {
            JSONObject group = groups.optJSONObject(i);
            if (group == null) continue;
            labels.add(group.optString("name"));
            ids.add(group.optString("id"));
        }
        Sheet.Builder sheet = Sheet.of(requireContext(), "选择用户组");
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
        LinearLayout card = cardHead("用户组", groups.length() + " 个用户组",
                showCreateGroup ? "收起" : "新建用户组", () -> {
                    capture();
                    showCreateGroup = !showCreateGroup;
                    render();
                });

        if (showCreateGroup) {
            newGroupNameInput = UI.input(requireContext(), "用户组名称");
            newGroupNameInput.setText(newGroupName);
            card.addView(UI.field(requireContext(), "用户组名称", newGroupNameInput));
            TextView create = UI.button(requireContext(), busy ? "创建中…" : "创建", UI.BTN_PRIMARY);
            create.setEnabled(!busy);
            create.setOnClickListener(v -> {
                capture();
                if (newGroupName.isEmpty()) return;
                run(() -> Api.post("/api/admin/groups",
                                new JSONObject().put("name", newGroupName).put("permissions", new JSONArray())),
                        "用户组已创建", () -> {
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
        if (group.optBoolean("builtin", false)) top.addView(tag("内置", false));
        card.addView(top);

        TextView count = UI.muted(requireContext(), groupMemberCount(group.optString("id")) + " 名成员");
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
                        "权限已更新");
            }), i == 0 ? 0 : UI.SM);
        }

        LinearLayout actions = UI.row(requireContext());
        TextView rename = UI.button(requireContext(), "重命名", UI.BTN_SECONDARY);
        rename.setOnClickListener(v -> promptRenameGroup(group));
        UI.weight(rename, 1f);
        actions.addView(rename);
        if (!group.optBoolean("builtin", false)) {
            TextView remove = UI.button(requireContext(), "删除", UI.BTN_DANGER);
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
            banner = "内置用户组不可重命名";
            bannerError = true;
            render();
            return;
        }
        EditText field = UI.input(requireContext(), "新的用户组名称");
        field.setText(group.optString("name"));
        Modal.of(requireContext(), "重命名用户组")
                .content(field)
                .cancel("取消")
                .confirm("保存", false, () -> {
                    String name = field.getText().toString().trim();
                    if (name.isEmpty() || name.equals(group.optString("name"))) return;
                    JSONArray perms = group.optJSONArray("permissions");
                    run(() -> Api.put("/api/admin/groups/" + group.optString("id"),
                                    new JSONObject().put("name", name)
                                            .put("permissions", perms == null ? new JSONArray() : perms)),
                            "名称已更新");
                })
                .show();
    }

    private void confirmRemoveGroup(JSONObject group) {
        Modal.of(requireContext(), "删除用户组")
                .message("确定删除用户组 " + group.optString("name") + "？")
                .cancel("取消")
                .confirm("删除", true, () -> run(
                        () -> Api.delete("/api/admin/groups/" + group.optString("id")),
                        "用户组已删除"))
                .show();
    }

    // ------------------------------------------------------------- invites tab

    private void invitesTab() {
        LinearLayout card = cardHead("邀请码", invites.length() + " 个邀请码",
                showCreateInvite ? "收起" : "生成邀请码", () -> {
                    capture();
                    showCreateInvite = !showCreateInvite;
                    render();
                });

        if (showCreateInvite) {
            card.addView(picker("用户组", groupName(inviteGroupId), this::pickInviteGroup));
            UI.margin(card.getChildAt(card.getChildCount() - 1), 0, UI.MD, 0, 0);

            inviteMaxInput = numberInput("0 = 不限", inviteMaxUses);
            card.addView(UI.field(requireContext(), "可用次数（0 = 不限）", inviteMaxInput, UI.MD));

            inviteExpireInput = numberInput("0 = 永久", inviteExpireDays);
            card.addView(UI.field(requireContext(), "有效天数（0 = 永久）", inviteExpireInput, UI.MD));

            TextView create = UI.button(requireContext(), busy ? "生成中…" : "生成", UI.BTN_PRIMARY);
            create.setEnabled(!busy);
            create.setOnClickListener(v -> submitCreateInvite());
            UI.margin(create, 0, UI.MD, 0, 0);
            card.addView(create);
            card.addView(UI.divider(requireContext()));
        }

        if (invites.length() == 0) {
            card.addView(emptyState("暂无邀请码。"));
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
        top.addView(tag(on ? "启用" : "停用", on));
        box.addView(top);

        int max = invite.optInt("max_uses", 0);
        long expires = invite.optLong("expires_at", 0);
        String meta = "用户组：" + defaulted(groupName(invite.optString("group_id")))
                + " · 已用 " + invite.optInt("used_count", 0) + " / " + (max > 0 ? String.valueOf(max) : "∞")
                + " · " + (expires > 0 ? formatTime(expires) : "永久");
        TextView caption = UI.muted(requireContext(), meta);
        UI.margin(caption, 0, UI.XS, 0, UI.SM);
        box.addView(caption);

        LinearLayout actions = UI.row(requireContext());
        TextView toggle = UI.button(requireContext(), on ? "停用" : "启用", UI.BTN_SECONDARY);
        toggle.setOnClickListener(v -> run(
                () -> Api.put("/api/admin/invites/" + invite.optString("code"),
                        new JSONObject().put("enabled", !on)),
                on ? "已停用" : "已启用"));
        UI.weight(toggle, 1f);
        actions.addView(toggle);
        TextView remove = UI.button(requireContext(), "删除", UI.BTN_DANGER);
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
        labels.add("默认用户组");
        ids.add("");
        for (int i = 0; i < groups.length(); i++) {
            JSONObject group = groups.optJSONObject(i);
            if (group == null) continue;
            labels.add(group.optString("name"));
            ids.add(group.optString("id"));
        }
        Sheet.Builder sheet = Sheet.of(requireContext(), "选择用户组");
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
        }, "邀请码已生成", () -> {
            showCreateInvite = false;
            inviteMaxUses = "";
            inviteExpireDays = "";
        });
    }

    private void confirmRemoveInvite(JSONObject invite) {
        Modal.of(requireContext(), "删除邀请码")
                .message("确定删除邀请码 " + invite.optString("code") + "？")
                .cancel("取消")
                .confirm("删除", true, () -> run(
                        () -> Api.delete("/api/admin/invites/" + invite.optString("code")),
                        "邀请码已删除"))
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
	LinearLayout card = cardHead("登录保护",
		"登录、二次验证、注册、验证码、找回与重置密码都受此限制。账号额度是真正的防线，换 IP 绕不过去；IP 额度用来挡住一台机器横扫多个账号。任一触发即返回 429 并附带解锁时间。",
		null, null);

	UI.addRow(card, toggleRow("启用登录限流", settings.optBoolean("rate_limit_enabled", false), true, next -> {
	settingsPut("rate_limit_enabled", next);
	saveSettings();
	}), UI.SM);

	rateLimitPerAccountInput = numberInput(ctx, "0 = 默认 5 次，负数 = 不限", rateLimitPerAccount);
	card.addView(UI.field(ctx, "每账号失败次数", rateLimitPerAccountInput, UI.SM));

	rateLimitPerIPInput = numberInput(ctx, "0 = 默认 20 次，负数 = 不限", rateLimitPerIP);
	card.addView(UI.field(ctx, "每 IP 失败次数", rateLimitPerIPInput, UI.SM));

	rateLimitWindowInput = numberInput(ctx, "0 = 默认 15 分钟", rateLimitWindow);
	card.addView(UI.field(ctx, "统计窗口（分钟）", rateLimitWindowInput, UI.SM));

	rateLimitFamiliarInput = numberInput(ctx, "0 = 默认 3 倍，1 = 不放宽", rateLimitFamiliar);
	card.addView(UI.field(ctx, "熟悉来源的额度倍数", rateLimitFamiliarInput, UI.SM));
	card.addView(UI.muted(ctx, "90 天内登录过的网段（IPv4 记 /24，IPv6 记 /64）会给更宽的账号额度。放宽是倍数而不是放行，所以熟悉的网段也会用完。"));

	TextView save = UI.button(ctx, busy ? "保存中…" : "保存限流设置", UI.BTN_PRIMARY);
	save.setEnabled(!busy);
	save.setOnClickListener(v -> saveRateLimitSettings());
	UI.margin(save, 0, UI.MD, 0, 0);
	card.addView(save);

	UI.addRow(card, toggleRow("锁定时通知管理员", settings.optBoolean("rate_limit_notify", false), true, next -> {
	settingsPut("rate_limit_notify", next);
	saveSettings();
	}), UI.MD);
	card.addView(UI.muted(ctx, "走各管理员自己配置的通知渠道（TG / 邮件），并沿用「登录通知」开关。"));
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
	banner = "登录保护设置已保存";
	bannerError = false;
	load();
	}, failure -> {
	busy = false;
	banner = "保存失败: " + failure.getMessage();
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
        card.addView(UI.cardTitle(requireContext(), "注册策略"));

        UI.addRow(card, toggleRow("开放注册", regOpen(), true, next -> {
            settings.remove("registration_enabled");
            settingsPut("registration_enabled", next);
            saveSettings();
        }), UI.MD);

        boolean verify = emailVerifyEnabled();
        boolean canVerify = smtpReady && regOpen();
        UI.addRow(card, toggleRow("注册邮箱验证", verify, canVerify, next -> {
            settingsPut("email_verify_disabled", !next);
            saveSettings();
        }), UI.MD);
        if (!smtpReady) {
            TextView hint = UI.muted(requireContext(), "未配置邮件服务，验证码无法发送。");
            UI.margin(hint, 0, UI.XS, 0, 0);
            card.addView(hint);
        }

        TextView inviteLabel = UI.label(requireContext(), "邀请码");
        UI.margin(inviteLabel, 0, UI.MD, 0, UI.SM);
        card.addView(inviteLabel);
        int mode = indexOf(INVITE_MODES, settings.optString("invite_mode", "off"));
        UI.addRow(card, UI.segmented(requireContext(), INVITE_LABELS, mode, index -> {
            settingsPut("invite_mode", INVITE_MODES[index]);
            saveSettings();
        }), 0);

        card.addView(picker("默认用户组", defaulted(groupName(settings.optString("default_group_id", ""))),
                this::pickDefaultGroup));
        UI.margin(card.getChildAt(card.getChildCount() - 1), 0, UI.MD, 0, 0);
        return card;
    }

    private View experimentalCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), "实验性功能"));
        TextView hint = UI.muted(requireContext(),
                "开启后，管理员侧边栏会显示「IP 优选实验室」。关闭后入口与 API 均不暴露。");
        UI.margin(hint, 0, UI.XS, 0, UI.MD);
        card.addView(hint);
        UI.addRow(card, toggleRow("开启实验性功能",
                settings.optBoolean("experimental_features_enabled", false), true, next -> {
                    settingsPut("experimental_features_enabled", next);
                    saveSettings();
                }), 0);
        return card;
    }

    private View turnstileCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), "人机验证（Cloudflare Turnstile）"));
        TextView hint = UI.muted(requireContext(),
                "可选防护：开启后登录、注册与找回密码需要完成 Cloudflare 人机验证。"
                        + "先在 Cloudflare 控制台创建 Turnstile widget（把本站域名加入允许域名），再填写 Site Key 与 Secret Key。");
        UI.margin(hint, 0, UI.XS, 0, UI.MD);
        card.addView(hint);

        UI.addRow(card, toggleRow("启用人机验证", settings.optBoolean("turnstile_enabled", false), true, next -> {
            capture();
            turnstileSiteKey = siteKeyInput == null ? turnstileSiteKey : siteKeyInput.getText().toString().trim();
            settingsPut("turnstile_enabled", next);
            saveTurnstile();
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
        save.setOnClickListener(v -> saveTurnstile());
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

    private void pickDefaultGroup() {
        List<String> labels = new ArrayList<>();
        List<String> ids = new ArrayList<>();
        labels.add("（未设置）");
        ids.add("");
        for (int i = 0; i < groups.length(); i++) {
            JSONObject group = groups.optJSONObject(i);
            if (group == null) continue;
            labels.add(group.optString("name"));
            ids.add(group.optString("id"));
        }
        Sheet.Builder sheet = Sheet.of(requireContext(), "默认用户组");
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
        run(() -> {
            Session.applyConfig(settings);
            return Api.put("/api/admin/settings", settings);
        }, "设置已保存");
    }

    private void saveTurnstile() {
        capture();
        JSONObject payload = turnstilePayload();
        turnstileSecret = "";
        run(() -> Api.put("/api/admin/settings", payload), "人机验证设置已保存");
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
        }, "OAuth 客户端配置已保存");
    }

    private void saveEncKey() {
        capture();
        if (encKeyInput.isEmpty()) {
            banner = "请输入 64 位十六进制密钥";
            bannerError = true;
            render();
            return;
        }
        String key = encKeyInput;
        encKeyInput = "";
        run(() -> Api.put("/api/admin/encryption-key", new JSONObject().put("key", key)), "加密密钥已保存");
    }

    // ----------------------------------------------------------------- smtp

    private void smtpTab() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), "SMTP 邮件服务"));
        TextView hint = UI.muted(requireContext(),
                "用于注册验证码与监控告警邮件。密码留空表示保持原值不变。"
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
        body.addView(card);

        body.addView(UI.spacer(requireContext(), UI.MD));

        LinearLayout test = UI.card(requireContext());
        test.addView(UI.cardTitle(requireContext(), "发送测试邮件"));
        testMailInput = UI.input(requireContext(), "你的邮箱");
        testMailInput.setInputType(InputType.TYPE_TEXT_VARIATION_EMAIL_ADDRESS);
        testMailInput.setText(testMailTo);
        test.addView(UI.field(requireContext(), "收件地址", testMailInput));
        TextView send = UI.button(requireContext(), busy ? "发送中…" : "发送测试邮件", UI.BTN_SECONDARY);
        send.setEnabled(!busy);
        send.setOnClickListener(v -> {
            capture();
            if (testMailTo.isEmpty()) {
                banner = "请输入收件邮箱";
                bannerError = true;
                render();
                return;
            }
            String to = testMailTo;
            run(() -> Api.post("/api/admin/smtp/test", new JSONObject().put("to", to)), "测试邮件已发送");
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
        }, "SMTP 设置已保存");
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
        LinearLayout card = cardHead("通行密钥",
                "通行密钥要求 HTTPS，依赖方 ID 只能是访问域名本身或其父域。两个字段都留空时按访问域名自动推导，同域下的子域都能用。「允许的来源」只在需要限定同一依赖方下放开哪几个子域时才填，须与依赖方 ID 同域或为其子域；一个依赖方 ID 覆盖不了两个不同的域名。",
                null, null);

        passkeyRPIDInput = UI.input(ctx, "依赖方 ID，如 panel.example.com");
        passkeyRPIDInput.setText(passkeyRPID);
        card.addView(UI.field(ctx, "依赖方 ID", passkeyRPIDInput, UI.SM));

        passkeyOriginsInput = UI.input(ctx, "允许的来源，逗号分隔");
        passkeyOriginsInput.setText(passkeyOrigins);
        card.addView(UI.field(ctx, "允许的来源", passkeyOriginsInput, UI.SM));

        androidPackageInput = UI.input(ctx, "com.tunnelmanager.app");
        androidPackageInput.setText(androidPackage);
        card.addView(UI.field(ctx, "Android 包名", androidPackageInput, UI.SM));

        androidFingerprintsInput = UI.textArea(ctx, "SHA-256 指纹，逗号或换行分隔");
        androidFingerprintsInput.setText(androidFingerprints);
        card.addView(UI.field(ctx, "Android 签名指纹", androidFingerprintsInput, UI.SM));
        card.addView(UI.muted(ctx, androidFingerprintsEffective.isEmpty()
                ? "当前没有生效的签名指纹，App 无法使用通行密钥。"
                : "当前生效：" + androidFingerprintsEffective));
        card.addView(UI.muted(ctx, "App 通过 /.well-known/assetlinks.json 校验域名归属，指纹取自签名证书（keytool 或 apksigner 的 SHA-256）。"));

        TextView save = UI.button(ctx, busy ? "保存中…" : "保存通行密钥设置", UI.BTN_PRIMARY);
        save.setEnabled(!busy);
        save.setOnClickListener(v -> savePasskeySettings());
        UI.margin(save, 0, UI.MD, 0, 0);
        card.addView(save);

        card.addView(UI.divider(ctx));
        UI.addRow(card, toggleRow("禁用密码登录（全站）", passwordLoginDisabled, passkeyAdminReady,
                next -> setGlobalPasswordLogin(next)), UI.MD);
        card.addView(UI.muted(ctx, passkeyAdminReady
                ? "开启后所有账户都只能用通行密钥登录；某个账户丢失通行密钥时，可在用户列表里恢复它的密码登录。"
                : "需至少一名启用的管理员已绑定通行密钥，避免面板被锁死。"));
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
            banner = "通行密钥设置已保存";
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
            banner = disabled ? "已禁用密码登录（全站）" : "已恢复密码登录（全站）";
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
                disabled ? "已禁用该账户的密码登录" : "已恢复该账户的密码登录");
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
        LinearLayout card = cardHead("审计概览", "近 7 天的操作统计。", null, null);
        LinearLayout row = UI.row(ctx);
        row.addView(auditStat(ctx, String.valueOf(auditStats.optInt("total", 0)), "操作"));
        row.addView(auditStat(ctx, String.valueOf(auditStats.optInt("failed", 0)), "失败"));
        row.addView(auditStat(ctx, String.valueOf(auditStats.optInt("today", 0)), "今日"));
        row.addView(auditStat(ctx, String.valueOf(auditStats.optInt("actors", 0)), "活跃账号"));
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
        LinearLayout card = cardHead("筛选", null, "查询", () -> {
            auditPage = 1;
            loadAudit();
        });

        auditActorInput = UI.input(ctx, "搜索用户（用户名）");
        auditActorInput.setText(auditActor);
        UI.addRow(card, auditActorInput, UI.SM);

        UI.addRow(card, auditPicker(ctx, "时间范围", AUDIT_RANGE_LABELS[auditRange],
            "选择时间范围", AUDIT_RANGE_LABELS, index -> {
            auditRange = index;
            auditPage = 1;
            loadAudit();
        }), UI.MD);

        UI.addRow(card, auditPicker(ctx, "操作类型", AUDIT_CATEGORY_LABELS[auditCategory],
            "选择操作类型", AUDIT_CATEGORY_LABELS, index -> {
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
        LinearLayout card = cardHead("日志", null, "刷新", () -> loadAudit());
        if (auditLoading) {
            card.addView(UI.muted(ctx, "加载中…"));
            return card;
        }
        if (auditLogs.length() == 0) {
            card.addView(emptyState("没有符合条件的日志。"));
            return card;
        }
        int pages = Math.max(1, (auditTotal + AUDIT_PAGE_SIZE - 1) / AUDIT_PAGE_SIZE);
        card.addView(UI.muted(ctx, "共 " + auditTotal + " 条 · 第 " + auditPage + " / " + pages + " 页"));
        for (int i = 0; i < auditLogs.length(); i++) {
            JSONObject entry = auditLogs.optJSONObject(i);
            if (entry == null) continue;
            UI.addRow(card, auditEntryRow(entry), UI.MD);
        }

        LinearLayout pager = UI.row(ctx);
        TextView prev = UI.button(ctx, "上一页", UI.BTN_SECONDARY);
        prev.setEnabled(auditPage > 1);
        prev.setOnClickListener(v -> {
            auditPage--;
            loadAudit();
        });
        pager.addView(prev);
        TextView next = UI.button(ctx, "下一页", UI.BTN_SECONDARY);
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
        top.addView(tag(success ? "成功" : "失败", success));
        box.addView(top);

        List<String> meta = new ArrayList<>();
        meta.add("操作人：" + defaulted(entry.optString("actor_name", "")));
        String target = entry.optString("target", "");
        if (!target.isEmpty()) meta.add("对象：" + target);
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
            banner = "读取审计日志失败：" + failure.getMessage();
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

    private static String auditActionLabel(String action) {
        for (int i = 0; i < AUDIT_ACTIONS.length; i++) {
            if (AUDIT_ACTIONS[i].equals(action)) return AUDIT_ACTION_LABELS[i];
        }
        return action.isEmpty() ? "未知操作" : action;
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
        return value == null || value.isEmpty() ? "默认" : value;
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
