package com.tunnelmanager.app;

import android.content.ClipData;
import android.content.ClipboardManager;
import android.content.Context;
import android.content.Intent;
import android.content.res.ColorStateList;
import android.graphics.Bitmap;
import android.graphics.BitmapFactory;
import android.graphics.Typeface;
import android.net.Uri;
import android.text.InputType;
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

import androidx.activity.result.ActivityResultLauncher;
import androidx.activity.result.contract.ActivityResultContracts;
import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.core.graphics.drawable.RoundedBitmapDrawable;
import androidx.core.graphics.drawable.RoundedBitmapDrawableFactory;

import org.json.JSONArray;
import org.json.JSONObject;

import java.io.InputStream;
import java.util.ArrayList;
import java.util.List;

/**
 * 账户设置 — profile, credentials and the Cloudflare connection.
 *
 * The web page is one long stack of cards; the ordering is kept, but two things
 * change for the phone. The avatar picker uses the system photo picker instead
 * of a file input, and the 2FA enrolment offers the authenticator app's own
 * {@code otpauth://} hand-off plus the manual key — a QR code is the one step a
 * phone cannot scan off its own screen.
 */
public class AccountFragment extends PageFragment {

    /** Server-side cap on uploads, mirrored here so the picker can reject early. */
    private static final long MAX_AVATAR_BYTES = 4L << 20;
    private static final String[] AVATAR_TYPES = {"image/png", "image/jpeg", "image/gif", "image/webp"};

    private ScrollView scroll;
    private LinearLayout body;

    private JSONObject me = new JSONObject();
    private JSONObject tf = new JSONObject();
    private JSONObject cf = new JSONObject();
    private String cfError = "";

    private String avatarUrl = "";
    private Bitmap avatarBitmap;
    private String avatarBitmapUrl = "";

    private String nicknameDraft = "";
    private EditText nicknameInput;
    private EditText emailInput;
    private EditText emailPasswordInput;
    private EditText usernameInput;
    private EditText usernamePasswordInput;
    private EditText currentPasswordInput;
    private EditText newPasswordInput;
    private EditText tfCodeInput;
    private EditText tfPasswordInput;

    /** Non-null while an enrolment is in flight: setup_token / secret / otpauth_uri. */
    private JSONObject tfSetup;
    private List<String> recoveryCodes = new ArrayList<>();
    private String tfNotice = "";

    private boolean loading = true;
    private boolean busy = false;
    private String statusMessage = "";
    private boolean statusOk = true;

    private final ActivityResultLauncher<String> avatarPicker =
            registerForActivityResult(new ActivityResultContracts.GetContent(), this::onAvatarPicked);

    @Override
    String route() {
        return "/account";
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
        statusMessage = "";
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("me", Api.get("/api/auth/me"));
            // Both of these can fail on their own without taking the page down.
            try {
                payload.put("tf", Api.get("/api/admin/2fa/status"));
            } catch (Api.Failure ignored) {
            }
            try {
                payload.put("cf", Api.get("/api/cloudflare/oauth/status"));
            } catch (Api.Failure failure) {
                payload.put("cf_error", failure.getMessage());
            }
            return payload;
        }, payload -> {
            loading = false;
            me = payload.optJSONObject("me") == null ? new JSONObject() : payload.optJSONObject("me");
            tf = payload.optJSONObject("tf") == null ? new JSONObject() : payload.optJSONObject("tf");
            cf = payload.optJSONObject("cf") == null ? new JSONObject() : payload.optJSONObject("cf");
            cfError = payload.optString("cf_error", "");
            Session.applyConfig(me);
            nicknameDraft = me.optString("nickname", "");
            avatarUrl = me.optString("avatar", "");
            if (!avatarUrl.equals(avatarBitmapUrl)) avatarBitmap = null;
            render();
            loadAvatar();
        }, failure -> {
            loading = false;
            statusMessage = "加载失败：" + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    private void loadAvatar() {
        final String url = avatarUrl;
        if (url.isEmpty() || url.equals(avatarBitmapUrl)) return;
        Api.async(() -> Api.asset(url), bytes -> {
            if (!url.equals(avatarUrl)) return;
            avatarBitmap = BitmapFactory.decodeByteArray(bytes, 0, bytes.length);
            avatarBitmapUrl = url;
            render();
        }, failure -> ignore());
    }

    private void ignore() {
        // The initial-circle fallback already stands in for a failed image.
    }

    // -------------------------------------------------------------- rendering

    private void render() {
        if (body == null || !alive()) return;
        int keepScroll = scroll.getScrollY();
        body.removeAllViews();

        body.addView(UI.pageTitle(requireContext(), "账户设置"));
        TextView subtitle = UI.muted(requireContext(), "管理登录凭据、Cloudflare 账户连接与账户安全");
        UI.margin(subtitle, 0, UI.XS, 0, UI.MD);
        body.addView(subtitle);

        if (loading) {
            body.addView(UI.muted(requireContext(), "加载中…"));
            return;
        }

        body.addView(profileCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(emailCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(twoFactorCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(cloudflareCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(usernameCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(passwordCard());

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
        TextView hint = UI.muted(requireContext(), desc);
        UI.margin(hint, 0, UI.XS, 0, 0);
        head.addView(hint);
        UI.margin(head, 0, 0, 0, UI.MD);
        return head;
    }

    private LinearLayout card(String title, String desc) {
        LinearLayout card = UI.card(requireContext());
        card.addView(cardHead(title, desc));
        return card;
    }

    private void note(LinearLayout card, String text, int topDp) {
        TextView note = UI.muted(requireContext(), text);
        UI.margin(note, 0, topDp, 0, 0);
        card.addView(note);
    }

    private EditText passwordInput(String hint) {
        EditText input = UI.input(requireContext(), hint);
        input.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_PASSWORD);
        return input;
    }

    // ------------------------------------------------------------ profile card

    private View profileCard() {
        LinearLayout card = card("个人主页", "设置展示名称与自定义头像，并管理账户邮箱。");

        LinearLayout row = UI.row(requireContext());
        row.setGravity(Gravity.CENTER_VERTICAL);
        row.addView(avatarPreview(), new LinearLayout.LayoutParams(UI.dp(64), UI.dp(64)));

        LinearLayout text = UI.column(requireContext());
        LinearLayout.LayoutParams textLp = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
        textLp.leftMargin = UI.dp(UI.LG);
        text.addView(UI.strong(requireContext(), Session.nickname()));
        TextView user = UI.muted(requireContext(), "登录用户名：" + me.optString("username", Session.username()));
        UI.margin(user, 0, UI.XS, 0, 0);
        text.addView(user);
        row.addView(text, textLp);
        card.addView(row);

        LinearLayout actions = UI.row(requireContext());
        TextView upload = UI.button(requireContext(), "上传头像", UI.BTN_SECONDARY);
        upload.setOnClickListener(v -> avatarPicker.launch("image/*"));
        actions.addView(upload);
        if (!avatarUrl.isEmpty()) {
            TextView remove = UI.button(requireContext(), "移除", UI.BTN_GHOST);
            remove.setOnClickListener(v -> updateProfile(nicknameDraft.trim(), ""));
            UI.margin(remove, UI.SM, 0, 0, 0);
            actions.addView(remove);
        }
        UI.margin(actions, 0, UI.MD, 0, 0);
        card.addView(actions);
        note(card, "PNG / JPG / GIF / WebP，最大 4 MiB。", UI.SM);

        nicknameInput = UI.input(requireContext(), "输入展示名称（留空则显示用户名）");
        nicknameInput.setText(nicknameDraft);
        card.addView(UI.field(requireContext(), "自定义名称", nicknameInput, UI.MD));

        TextView save = UI.button(requireContext(), busy ? "保存中…" : "保存资料", UI.BTN_PRIMARY);
        save.setEnabled(!busy);
        save.setOnClickListener(v -> {
            String value = nicknameInput.getText().toString().trim();
            if (value.length() > 64) {
                statusMessage = "自定义名称不能超过 64 个字符";
                statusOk = false;
                render();
                return;
            }
            updateProfile(value, avatarUrl);
        });
        UI.margin(save, 0, UI.MD, 0, 0);
        card.addView(save);
        return card;
    }

    private View avatarPreview() {
        if (avatarBitmap != null) {
            ImageView view = new ImageView(requireContext());
            RoundedBitmapDrawable drawable = RoundedBitmapDrawableFactory.create(getResources(), avatarBitmap);
            drawable.setCircular(true);
            view.setImageDrawable(drawable);
            view.setScaleType(ImageView.ScaleType.CENTER_CROP);
            return view;
        }
        Palette p = Theme.p();
        TextView initial = UI.text(requireContext(), initial(), 24, p.ink, Typeface.BOLD);
        initial.setGravity(Gravity.CENTER);
        initial.setBackground(UI.circle(p.canvasSoft2, p.hairline, 1));
        return initial;
    }

    private String initial() {
        String name = Session.nickname();
        return name.isEmpty() ? "?" : name.substring(0, 1).toUpperCase();
    }

    private void updateProfile(String nickname, String avatar) {
        busy = true;
        statusMessage = "";
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("nickname", nickname);
            payload.put("avatar", avatar);
            return Api.put("/api/admin/profile", payload);
        }, ignore -> {
            busy = false;
            if (avatar.isEmpty()) {
                avatarUrl = "";
                avatarBitmap = null;
                avatarBitmapUrl = "";
                statusMessage = "头像已移除";
            } else {
                statusMessage = "个人资料已更新";
            }
            statusOk = true;
            reloadIdentity();
        }, failure -> {
            busy = false;
            statusMessage = "保存失败：" + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    /** Re-reads /api/auth/me so the shell's title and this page agree. */
    private void reloadIdentity() {
        Api.async(() -> Api.get("/api/auth/me"), payload -> {
            me = payload;
            Session.applyConfig(me);
            nicknameDraft = me.optString("nickname", "");
            String url = me.optString("avatar", "");
            if (!url.equals(avatarUrl)) {
                avatarUrl = url;
                if (!url.equals(avatarBitmapUrl)) avatarBitmap = null;
            }
            render();
            loadAvatar();
        }, failure -> render());
    }

    // -------------------------------------------------------------- avatar IO

    private void onAvatarPicked(Uri uri) {
        if (uri == null) return;
        try {
            String mime = requireContext().getContentResolver().getType(uri);
            if (mime != null && !isAllowedType(mime)) {
                statusMessage = "仅支持 PNG / JPG / GIF / WebP 格式";
                statusOk = false;
                render();
                return;
            }
            byte[] bytes = readAll(uri);
            if (bytes.length == 0) {
                statusMessage = "读取图片失败";
                statusOk = false;
                render();
                return;
            }
            if (bytes.length > MAX_AVATAR_BYTES) {
                statusMessage = "图片不能超过 4 MiB";
                statusOk = false;
                render();
                return;
            }
            uploadAvatar(bytes, mime == null ? "image/png" : mime);
        } catch (Exception e) {
            statusMessage = "读取图片失败：" + e.getMessage();
            statusOk = false;
            render();
        }
    }

    private static boolean isAllowedType(String mime) {
        for (String allowed : AVATAR_TYPES) {
            if (allowed.equalsIgnoreCase(mime)) return true;
        }
        return false;
    }

    private byte[] readAll(Uri uri) throws Exception {
        try (InputStream in = requireContext().getContentResolver().openInputStream(uri)) {
            if (in == null) return new byte[0];
            java.io.ByteArrayOutputStream buffer = new java.io.ByteArrayOutputStream();
            byte[] chunk = new byte[8192];
            int read;
            while ((read = in.read(chunk)) != -1) buffer.write(chunk, 0, read);
            return buffer.toByteArray();
        }
    }

    private void uploadAvatar(byte[] data, String mime) {
        busy = true;
        statusMessage = "";
        render();
        String name = "avatar" + (mime.contains("png") ? ".png"
                : mime.contains("gif") ? ".gif"
                : mime.contains("webp") ? ".webp" : ".jpg");
        Api.async(() -> Api.postFile("/api/account/avatar", data, name, mime), result -> {
            busy = false;
            statusMessage = "头像已更新";
            statusOk = true;
            String url = result.optString("url", "");
            if (!url.isEmpty()) {
                avatarUrl = url;
                avatarBitmap = null;
                avatarBitmapUrl = "";
            }
            reloadIdentity();
        }, failure -> {
            busy = false;
            statusMessage = "上传失败：" + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    // ---------------------------------------------------------------- email

    private View emailCard() {
        LinearLayout card = UI.card(requireContext());
        String email = me.optString("email", "");

        LinearLayout head = UI.row(requireContext());
        head.addView(UI.tag(requireContext(), email.isEmpty() ? "未绑定" : "已绑定", !email.isEmpty()));
        TextView title = UI.cardTitle(requireContext(), "账户邮箱");
        UI.margin(title, UI.SM, 0, 0, 0);
        head.addView(title);
        card.addView(head);

        note(card, email.isEmpty()
                ? "绑定邮箱后可用于找回密码与接收服务告警。"
                : "当前邮箱：" + email, UI.SM);

        emailInput = UI.input(requireContext(), "name@example.com");
        emailInput.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_EMAIL_ADDRESS);
        card.addView(UI.field(requireContext(), email.isEmpty() ? "邮箱地址" : "新邮箱", emailInput, UI.MD));

        emailPasswordInput = passwordInput("输入当前密码确认");
        card.addView(UI.field(requireContext(), "当前密码确认", emailPasswordInput, UI.MD));

        TextView save = UI.button(requireContext(), email.isEmpty() ? "绑定邮箱" : "修改邮箱", UI.BTN_SECONDARY);
        save.setOnClickListener(v -> {
            String value = emailInput.getText().toString().trim();
            String password = emailPasswordInput.getText().toString();
            if (value.isEmpty() || password.isEmpty()) {
                statusMessage = "请填写邮箱和当前密码";
                statusOk = false;
                render();
                return;
            }
            changeEmail(value, password);
        });
        UI.margin(save, 0, UI.MD, 0, 0);
        card.addView(save);
        return card;
    }

    private void changeEmail(String email, String password) {
        busy = true;
        statusMessage = "";
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("current_password", password);
            payload.put("new_email", email);
            return Api.put("/api/admin/email", payload);
        }, result -> {
            busy = false;
            statusMessage = "邮箱已更新";
            statusOk = true;
            emailInput = null;
            emailPasswordInput = null;
            reloadIdentity();
        }, failure -> {
            busy = false;
            statusMessage = "保存失败：" + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    // ------------------------------------------------------------- two factor

    private View twoFactorCard() {
        LinearLayout card = UI.card(requireContext());
        boolean enabled = tf.optBoolean("enabled", false);

        LinearLayout head = UI.row(requireContext());
        head.addView(UI.tag(requireContext(), enabled ? "已启用" : "未启用", enabled));
        TextView title = UI.cardTitle(requireContext(), "双重身份验证");
        UI.margin(title, UI.SM, 0, 0, 0);
        head.addView(title);
        card.addView(head);

        note(card, enabled
                ? "验证器动态口令已开启；剩余恢复代码 " + tf.optInt("recovery_codes_remaining", 0) + " 个。"
                : "使用验证器动态口令保护管理操作，即使密码泄露也能阻止直接登录。", UI.SM);

        if (!recoveryCodes.isEmpty()) {
            card.addView(UI.divider(requireContext()));
            card.addView(recoveryPanel());
            return card;
        }
        if (!tfNotice.isEmpty()) {
            TextView notice = UI.banner(requireContext(), tfNotice, false);
            UI.margin(notice, 0, UI.MD, 0, 0);
            card.addView(notice);
        }
        card.addView(UI.divider(requireContext()));
        if (enabled) card.addView(disableTwoFactorBlock());
        else if (tfSetup != null) card.addView(setupBlock());
        else card.addView(startBlock());
        return card;
    }

    private View startBlock() {
        LinearLayout box = UI.column(requireContext());
        if (!tf.optBoolean("setup_available", true)) {
            box.addView(UI.muted(requireContext(), "服务器未配置加密密钥，暂时无法开启双重验证。"));
            return box;
        }
        TextView start = UI.button(requireContext(), "开始设置", UI.BTN_PRIMARY);
        start.setOnClickListener(v -> startSetup());
        box.addView(start);
        return box;
    }

    private View setupBlock() {
        LinearLayout box = UI.column(requireContext());
        String secret = tfSetup.optString("secret", "");
        String uri = tfSetup.optString("otpauth_uri", "");

        box.addView(UI.muted(requireContext(), "在验证器 App 里添加以下密钥，然后输入它生成的 6 位动态口令。"));
        TextView key = UI.mono(requireContext(), secret, Theme.p().ink);
        key.setTextSize(15);
        key.setTextIsSelectable(true);
        key.setBackground(UI.roundedStroke(Theme.p().canvasSoft2, UI.RADIUS_MD, Theme.p().hairline, 1));
        key.setPadding(UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD));
        UI.margin(key, 0, UI.MD, 0, 0);
        box.addView(key);

        LinearLayout actions = UI.row(requireContext());
        TextView add = UI.button(requireContext(), "添加到验证器", UI.BTN_SECONDARY);
        add.setOnClickListener(v -> openAuthenticator(uri, secret));
        UI.weight(add, 1f);
        actions.addView(add);
        TextView copy = UI.button(requireContext(), "复制密钥", UI.BTN_SECONDARY);
        copy.setOnClickListener(v -> copyText(secret, "密钥已复制"));
        UI.weight(copy, 1f);
        UI.margin(copy, UI.SM, 0, 0, 0);
        actions.addView(copy);
        UI.margin(actions, 0, UI.MD, 0, 0);
        box.addView(actions);

        tfCodeInput = UI.input(requireContext(), "6 位动态口令");
        tfCodeInput.setInputType(InputType.TYPE_CLASS_NUMBER);
        box.addView(UI.field(requireContext(), "动态口令", tfCodeInput, UI.MD));

        TextView confirm = UI.button(requireContext(), busy ? "验证中…" : "确认并启用", UI.BTN_PRIMARY);
        confirm.setEnabled(!busy);
        confirm.setOnClickListener(v -> confirmSetup(tfCodeInput.getText().toString().trim()));
        UI.margin(confirm, 0, UI.MD, 0, 0);
        box.addView(confirm);

        TextView cancel = UI.button(requireContext(), "取消设置", UI.BTN_GHOST);
        cancel.setOnClickListener(v -> {
            tfSetup = null;
            tfNotice = "";
            render();
        });
        UI.margin(cancel, 0, UI.SM, 0, 0);
        box.addView(cancel);
        return box;
    }

    private View recoveryPanel() {
        LinearLayout box = UI.column(requireContext());
        box.addView(UI.strong(requireContext(), "保存恢复代码"));
        box.addView(UI.muted(requireContext(),
                "验证器不可用时，每个代码可用于一次登录。离开此页面前请复制保存，之后无法再次查看。"));

        TextView codes = UI.mono(requireContext(), String.join("\n", recoveryCodes), Theme.p().ink);
        codes.setTextSize(15);
        codes.setLineSpacing(UI.dp(4), 1f);
        codes.setTextIsSelectable(true);
        codes.setBackground(UI.roundedStroke(Theme.p().canvasSoft2, UI.RADIUS_MD, Theme.p().hairline, 1));
        codes.setPadding(UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD));
        UI.margin(codes, 0, UI.MD, 0, 0);
        box.addView(codes);

        TextView copy = UI.button(requireContext(), "复制全部", UI.BTN_SECONDARY);
        copy.setOnClickListener(v -> copyText(String.join("\n", recoveryCodes), "恢复代码已复制"));
        UI.margin(copy, 0, UI.MD, 0, 0);
        box.addView(copy);

        TextView done = UI.button(requireContext(), "完成并重新登录", UI.BTN_PRIMARY);
        done.setOnClickListener(v -> signOut("双重验证已启用，请重新登录"));
        UI.margin(done, 0, UI.SM, 0, 0);
        box.addView(done);
        return box;
    }

    private View disableTwoFactorBlock() {
        LinearLayout box = UI.column(requireContext());
        box.addView(UI.strong(requireContext(), "停用双重身份验证"));
        note(box, "需要当前密码与一次动态口令。", UI.XS);

        tfPasswordInput = passwordInput("当前密码");
        box.addView(UI.field(requireContext(), "当前密码", tfPasswordInput, UI.MD));
        tfCodeInput = UI.input(requireContext(), "6 位动态口令");
        tfCodeInput.setInputType(InputType.TYPE_CLASS_NUMBER);
        box.addView(UI.field(requireContext(), "动态口令", tfCodeInput, UI.MD));

        TextView off = UI.button(requireContext(), busy ? "停用中…" : "停用", UI.BTN_DANGER);
        off.setEnabled(!busy);
        off.setOnClickListener(v -> {
            String password = tfPasswordInput.getText().toString();
            String code = tfCodeInput.getText().toString().trim();
            if (password.isEmpty() || code.isEmpty()) {
                statusMessage = "请填写当前密码和动态口令";
                statusOk = false;
                render();
                return;
            }
            Modal.of(requireContext(), "停用双重验证")
                    .message("停用后仅凭密码即可登录，确定继续吗？")
                    .cancel("取消")
                    .confirm("停用", true, () -> disableTwoFactor(password, code))
                    .show();
        });
        UI.margin(off, 0, UI.MD, 0, 0);
        box.addView(off);
        return box;
    }

    private void startSetup() {
        busy = true;
        statusMessage = "";
        tfNotice = "";
        render();
        Api.async(() -> Api.post("/api/admin/2fa/setup", null), payload -> {
            busy = false;
            tfSetup = payload;
            render();
        }, failure -> {
            busy = false;
            statusMessage = "无法开始设置：" + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    private void confirmSetup(String code) {
        if (!code.matches("\\d{6}")) {
            statusMessage = "请输入 6 位动态口令";
            statusOk = false;
            render();
            return;
        }
        busy = true;
        statusMessage = "";
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("setup_token", tfSetup.optString("setup_token", ""));
            payload.put("code", code);
            return Api.post("/api/admin/2fa/confirm", payload);
        }, payload -> {
            busy = false;
            tfSetup = null;
            tf = new JSONObject();
            JSONArray codes = payload.optJSONArray("recovery_codes");
            recoveryCodes = new ArrayList<>();
            if (codes != null) {
                for (int i = 0; i < codes.length(); i++) recoveryCodes.add(codes.optString(i));
            }
            statusMessage = "";
            render();
        }, failure -> {
            busy = false;
            statusMessage = "验证失败：" + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    private void disableTwoFactor(String password, String code) {
        busy = true;
        statusMessage = "";
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("current_password", password);
            payload.put("code", code);
            Api.post("/api/admin/2fa/disable", payload);
            JSONObject next = new JSONObject();
            next.put("enabled", false);
            next.put("setup_available", true);
            return next;
        }, payload -> {
            busy = false;
            tf = payload;
            statusMessage = "双重验证已停用";
            statusOk = true;
            render();
        }, failure -> {
            busy = false;
            statusMessage = "停用失败：" + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    private void openAuthenticator(String uri, String secret) {
        if (uri != null && !uri.isEmpty()) {
            try {
                startActivity(new Intent(Intent.ACTION_VIEW, Uri.parse(uri)));
                return;
            } catch (Exception ignored) {
                // No authenticator registered for otpauth:// — fall through.
            }
        }
        copyText(secret, "没有找到可用的验证器 App，密钥已复制");
    }

    // ------------------------------------------------------------- cloudflare

    private View cloudflareCard() {
        LinearLayout card = UI.card(requireContext());
        boolean connected = cf.optBoolean("connected", false);
        String source = cf.optString("source", "");

        LinearLayout head = UI.row(requireContext());
        TextView title = UI.cardTitle(requireContext(), "Cloudflare 账户连接");
        head.addView(title, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        head.addView(UI.tag(requireContext(), connected ? "已连接" : "未连接", connected));
        card.addView(head);
        note(card, "OAuth 连接存在时始终优先用于账户、隧道与 Zone 请求。", UI.XS);

        if (!cfError.isEmpty()) {
            TextView error = UI.banner(requireContext(), cfError, true);
            UI.margin(error, 0, UI.MD, 0, 0);
            card.addView(error);
            return card;
        }

        card.addView(UI.divider(requireContext()));
        card.addView(statusRow("凭据来源", "oauth".equals(source) ? "OAuth 授权" : "环境变量 API Token"));
        String accountName = cf.optString("account_name", "");
        String accountId = cf.optString("account_id", "");
        card.addView(statusRow("当前账户", accountName.isEmpty() ? (accountId.isEmpty() ? "尚未选择" : accountId) : accountName));

        JSONArray connections = cf.optJSONArray("connections");
        String activeId = cf.optString("active_connection_id", "");
        if (connections != null && connections.length() > 0) {
            for (int i = 0; i < connections.length(); i++) {
                JSONObject conn = connections.optJSONObject(i);
                if (conn == null) continue;
                boolean active = !activeId.isEmpty() && activeId.equals(conn.optString("id"));
                UI.addRow(card, connectionRow(conn, active), UI.MD);
            }
        }

        if (cf.optBoolean("configured", false)) {
            TextView start = UI.button(requireContext(), "oauth".equals(source) ? "新增账户授权" : "连接 Cloudflare", UI.BTN_PRIMARY);
            start.setOnClickListener(v -> startOAuth());
            UI.margin(start, 0, UI.MD, 0, 0);
            card.addView(start);
        } else {
            note(card, "服务器尚未配置 OAuth 客户端，当前使用环境变量 API Token。", UI.MD);
        }
        return card;
    }

    private View statusRow(String label, String value) {
        LinearLayout row = UI.row(requireContext());
        row.setPadding(0, UI.dp(UI.XS), 0, UI.dp(UI.XS));
        TextView name = UI.muted(requireContext(), label);
        row.addView(name, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        TextView text = UI.text(requireContext(), value, 13, Theme.p().ink, Typeface.BOLD);
        text.setGravity(Gravity.END);
        row.addView(text);
        return row;
    }

    private View connectionRow(JSONObject conn, boolean active) {
        LinearLayout row = UI.row(requireContext());
        row.setLayoutParams(new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));
        row.setBackground(UI.roundedStroke(Theme.p().canvasSoft2, UI.RADIUS_MD, Theme.p().hairline, 1));
        row.setPadding(UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD));

        LinearLayout text = UI.column(requireContext());
        String name = conn.optString("account_name", "");
        if (name.isEmpty()) name = conn.optString("label", "连接");
        text.addView(UI.strong(requireContext(), name));
        String accountId = conn.optString("account_id", "");
        if (!accountId.isEmpty()) {
            text.addView(UI.mono(requireContext(), accountId, Theme.p().mute));
        }
        row.addView(text, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));

        String id = conn.optString("id", "");
        if (active) {
            row.addView(UI.tag(requireContext(), "使用中", true));
        } else {
            TextView use = UI.button(requireContext(), "设为当前", UI.BTN_GHOST);
            use.setOnClickListener(v -> activateConnection(id));
            row.addView(use);
        }
        TextView remove = UI.button(requireContext(), "删除", UI.BTN_GHOST);
        remove.setTextColor(Theme.p().error);
        remove.setOnClickListener(v -> Modal.of(requireContext(), "删除连接")
                .message("删除后将撤销该 OAuth 凭据，已选择的账户与隧道也会被清除。")
                .cancel("取消")
                .confirm("删除", true, () -> disconnectConnection(id))
                .show());
        row.addView(remove);

        return row;
    }

    private void startOAuth() {
        busy = true;
        statusMessage = "";
        render();
        Api.async(() -> Api.post("/api/cloudflare/oauth/start", null), payload -> {
            busy = false;
            render();
            String url = payload.optString("authorization_url", "");
            if (url.isEmpty()) {
                statusMessage = "服务器未返回授权地址";
                statusOk = false;
                render();
                return;
            }
            try {
                startActivity(new Intent(Intent.ACTION_VIEW, Uri.parse(url)));
                statusMessage = "已打开浏览器，完成授权后回到这里刷新";
                statusOk = true;
            } catch (Exception e) {
                copyText(url, "没有可用浏览器，授权地址已复制");
                statusMessage = "没有可用浏览器，授权地址已复制到剪贴板";
                statusOk = true;
            }
            render();
        }, failure -> {
            busy = false;
            statusMessage = "无法开始授权：" + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    private void activateConnection(String id) {
        busy = true;
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("connection_id", id);
            return Api.put("/api/cloudflare/oauth/connection", payload);
        }, payload -> {
            busy = false;
            statusMessage = "已切换当前连接";
            statusOk = true;
            reloadCloudflare();
        }, failure -> {
            busy = false;
            statusMessage = "切换失败：" + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    private void disconnectConnection(String id) {
        busy = true;
        render();
        Api.async(() -> Api.delete("/api/cloudflare/oauth" + (id.isEmpty() ? "" : "?connection_id=" + id)),
                payload -> {
                    busy = false;
                    statusMessage = payload.optString("warning", "连接已删除");
                    statusOk = true;
                    reloadCloudflare();
                }, failure -> {
                    busy = false;
                    statusMessage = "删除失败：" + failure.getMessage();
                    statusOk = false;
                    render();
                });
    }

    private void reloadCloudflare() {
        Api.async(() -> Api.get("/api/cloudflare/oauth/status"), payload -> {
            cf = payload;
            cfError = "";
            render();
        }, failure -> {
            cfError = failure.getMessage();
            render();
        });
    }

    // --------------------------------------------------------- username & pwd

    private View usernameCard() {
        LinearLayout card = card("修改用户名", "当前用户名：" + me.optString("username", Session.username()));
        usernameInput = UI.input(requireContext(), "输入新用户名");
        card.addView(UI.field(requireContext(), "新用户名", usernameInput));
        usernamePasswordInput = passwordInput("输入当前密码确认");
        card.addView(UI.field(requireContext(), "当前密码确认", usernamePasswordInput, UI.MD));

        TextView save = UI.button(requireContext(), busy ? "保存中…" : "更新用户名", UI.BTN_SECONDARY);
        save.setEnabled(!busy);
        save.setOnClickListener(v -> {
            String value = usernameInput.getText().toString().trim();
            String password = usernamePasswordInput.getText().toString();
            if (value.isEmpty() || password.isEmpty()) {
                statusMessage = "请填写新用户名和当前密码";
                statusOk = false;
                render();
                return;
            }
            changeUsername(value, password);
        });
        UI.margin(save, 0, UI.MD, 0, 0);
        card.addView(save);
        return card;
    }

    private View passwordCard() {
        LinearLayout card = card("修改密码", "密码长度不少于 6 位。");
        currentPasswordInput = passwordInput("当前密码");
        card.addView(UI.field(requireContext(), "当前密码", currentPasswordInput));
        newPasswordInput = passwordInput("新密码");
        card.addView(UI.field(requireContext(), "新密码", newPasswordInput, UI.MD));

        TextView save = UI.button(requireContext(), busy ? "保存中…" : "更新密码", UI.BTN_PRIMARY);
        save.setEnabled(!busy);
        save.setOnClickListener(v -> {
            String current = currentPasswordInput.getText().toString();
            String next = newPasswordInput.getText().toString();
            if (current.isEmpty() || next.isEmpty()) {
                statusMessage = "请填写当前密码和新密码";
                statusOk = false;
                render();
                return;
            }
            if (next.length() < 6) {
                statusMessage = "新密码长度不能少于 6 位";
                statusOk = false;
                render();
                return;
            }
            changePassword(current, next);
        });
        UI.margin(save, 0, UI.MD, 0, 0);
        card.addView(save);
        return card;
    }

    private void changeUsername(String username, String password) {
        busy = true;
        statusMessage = "";
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("current_password", password);
            payload.put("new_username", username);
            return Api.put("/api/admin/username", payload);
        }, payload -> signOut("用户名已更新，请重新登录"), failure -> {
            busy = false;
            statusMessage = "更新失败：" + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    private void changePassword(String current, String next) {
        busy = true;
        statusMessage = "";
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("current_password", current);
            payload.put("new_password", next);
            return Api.put("/api/admin/password", payload);
        }, payload -> signOut("密码已更新，请重新登录"), failure -> {
            busy = false;
            statusMessage = "更新失败：" + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    /**
     * Both credential changes and enabling 2FA bump the login epoch, so the
     * stored token is already dead; going back to the login screen is the only
     * honest thing left to do.
     */
    private void signOut(String message) {
        if (getContext() == null) return;
        android.widget.Toast.makeText(requireContext(), message, android.widget.Toast.LENGTH_LONG).show();
        Session.logout(requireContext());
        Intent intent = new Intent(requireContext(), MainActivity.class);
        intent.addFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP | Intent.FLAG_ACTIVITY_NEW_TASK);
        startActivity(intent);
        requireActivity().finish();
    }

    private void copyText(String value, String confirmation) {
        ClipboardManager manager = (ClipboardManager) requireContext().getSystemService(Context.CLIPBOARD_SERVICE);
        if (manager != null) {
            manager.setPrimaryClip(ClipData.newPlainText("Tunnel Manager", value));
            android.widget.Toast.makeText(requireContext(), confirmation, android.widget.Toast.LENGTH_SHORT).show();
        }
    }
}
