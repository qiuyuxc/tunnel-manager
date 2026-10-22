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

    /** /api/account/passkeys: bound credentials, relying party and the switch. */
    private JSONObject pk = new JSONObject();
    private EditText pkNameInput;
    private EditText pkPasswordInput;
    private boolean pkBusy = false;

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
            try {
                payload.put("pk", Api.get("/api/account/passkeys"));
            } catch (Api.Failure ignored) {
            }
            return payload;
        }, payload -> {
            loading = false;
            me = payload.optJSONObject("me") == null ? new JSONObject() : payload.optJSONObject("me");
            tf = payload.optJSONObject("tf") == null ? new JSONObject() : payload.optJSONObject("tf");
            cf = payload.optJSONObject("cf") == null ? new JSONObject() : payload.optJSONObject("cf");
            cfError = payload.optString("cf_error", "");
            pk = payload.optJSONObject("pk") == null ? new JSONObject() : payload.optJSONObject("pk");
            Session.applyConfig(me);
            nicknameDraft = me.optString("nickname", "");
            avatarUrl = me.optString("avatar", "");
            if (!avatarUrl.equals(avatarBitmapUrl)) avatarBitmap = null;
            render();
            loadAvatar();
        }, failure -> {
            loading = false;
            statusMessage = I18n.text("m_01de8216e0d0") + failure.getMessage();
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

        body.addView(UI.pageTitle(requireContext(), I18n.text("m_a720b4000be6")));
        TextView subtitle = UI.muted(requireContext(), I18n.text("m_d86f5c72c33e"));
        UI.margin(subtitle, 0, UI.XS, 0, UI.MD);
        body.addView(subtitle);

        if (loading) {
            body.addView(UI.muted(requireContext(), I18n.text("m_4927a53bcc88")));
            return;
        }

        body.addView(profileCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(emailCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(twoFactorCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(passkeyCard());
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
        LinearLayout card = card(I18n.text("m_227eef9cd7b1"), I18n.text("m_c9ce2f44a713"));

        LinearLayout row = UI.row(requireContext());
        row.setGravity(Gravity.CENTER_VERTICAL);
        row.addView(avatarPreview(), new LinearLayout.LayoutParams(UI.dp(64), UI.dp(64)));

        LinearLayout text = UI.column(requireContext());
        LinearLayout.LayoutParams textLp = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
        textLp.leftMargin = UI.dp(UI.LG);
        text.addView(UI.strong(requireContext(), Session.nickname()));
        TextView user = UI.muted(requireContext(), I18n.text("m_6bee6bc667d2") + me.optString("username", Session.username()));
        UI.margin(user, 0, UI.XS, 0, 0);
        text.addView(user);
        row.addView(text, textLp);
        card.addView(row);

        LinearLayout actions = UI.row(requireContext());
        TextView upload = UI.button(requireContext(), I18n.text("m_d7d2344d545d"), UI.BTN_SECONDARY);
        upload.setOnClickListener(v -> avatarPicker.launch("image/*"));
        actions.addView(upload);
        if (!avatarUrl.isEmpty()) {
            TextView remove = UI.button(requireContext(), I18n.text("m_6135d4159e89"), UI.BTN_GHOST);
            remove.setOnClickListener(v -> updateProfile(nicknameDraft.trim(), ""));
            UI.margin(remove, UI.SM, 0, 0, 0);
            actions.addView(remove);
        }
        UI.margin(actions, 0, UI.MD, 0, 0);
        card.addView(actions);
        note(card, I18n.text("m_489bb505af13"), UI.SM);

        nicknameInput = UI.input(requireContext(), I18n.text("m_9e366ced887c"));
        nicknameInput.setText(nicknameDraft);
        card.addView(UI.field(requireContext(), I18n.text("m_bf3b4816f16e"), nicknameInput, UI.MD));

        TextView save = UI.button(requireContext(), busy ? I18n.text("m_ff509c9ba052") : I18n.text("m_c0a70fcb56f0"), UI.BTN_PRIMARY);
        save.setEnabled(!busy);
        save.setOnClickListener(v -> {
            String value = nicknameInput.getText().toString().trim();
            if (value.length() > 64) {
                statusMessage = I18n.text("m_dae4662865e0");
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
                statusMessage = I18n.text("m_eca726423f04");
            } else {
                statusMessage = I18n.text("m_b844ea2e13f3");
            }
            statusOk = true;
            reloadIdentity();
        }, failure -> {
            busy = false;
            statusMessage = I18n.text("m_6ade7baf8737") + failure.getMessage();
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
                statusMessage = I18n.text("m_24a3b2d5136c");
                statusOk = false;
                render();
                return;
            }
            byte[] bytes = readAll(uri);
            if (bytes.length == 0) {
                statusMessage = I18n.text("m_c72ad3f4aa4e");
                statusOk = false;
                render();
                return;
            }
            if (bytes.length > MAX_AVATAR_BYTES) {
                statusMessage = I18n.text("m_9d0b8bbb7900");
                statusOk = false;
                render();
                return;
            }
            uploadAvatar(bytes, mime == null ? "image/png" : mime);
        } catch (Exception e) {
            statusMessage = I18n.text("m_b1feeac12438") + e.getMessage();
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
            statusMessage = I18n.text("m_a938ed15b3e5");
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
            statusMessage = I18n.text("m_72e1fdcf8b37") + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    // ---------------------------------------------------------------- email

    private View emailCard() {
        LinearLayout card = UI.card(requireContext());
        String email = me.optString("email", "");

        LinearLayout head = UI.row(requireContext());
        head.addView(UI.tag(requireContext(), email.isEmpty() ? I18n.text("m_e026c6693dc5") : I18n.text("m_f70c0007f30f"), !email.isEmpty()));
        TextView title = UI.cardTitle(requireContext(), I18n.text("m_f8f1b032f460"));
        UI.margin(title, UI.SM, 0, 0, 0);
        head.addView(title);
        card.addView(head);

        note(card, email.isEmpty()
                ? I18n.text("m_e6bddbeaa97b")
                : I18n.text("m_af4dddc3c14e") + email, UI.SM);

        emailInput = UI.input(requireContext(), "name@example.com");
        emailInput.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_EMAIL_ADDRESS);
        card.addView(UI.field(requireContext(), email.isEmpty() ? I18n.text("m_2e412e03837f") : I18n.text("m_3067c39fa4b2"), emailInput, UI.MD));

        emailPasswordInput = passwordInput(I18n.text("m_632da57acf53"));
        card.addView(UI.field(requireContext(), I18n.text("m_ab48ee83149b"), emailPasswordInput, UI.MD));

        TextView save = UI.button(requireContext(), email.isEmpty() ? I18n.text("m_6d4b7dce1ee6") : I18n.text("m_da0f5e37e88b"), UI.BTN_SECONDARY);
        save.setOnClickListener(v -> {
            String value = emailInput.getText().toString().trim();
            String password = emailPasswordInput.getText().toString();
            if (value.isEmpty() || password.isEmpty()) {
                statusMessage = I18n.text("m_bdb626f8b359");
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
            statusMessage = I18n.text("m_0870a627daa8");
            statusOk = true;
            emailInput = null;
            emailPasswordInput = null;
            reloadIdentity();
        }, failure -> {
            busy = false;
            statusMessage = I18n.text("m_6ade7baf8737") + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    // ------------------------------------------------------------- two factor

    private View twoFactorCard() {
        LinearLayout card = UI.card(requireContext());
        boolean enabled = tf.optBoolean("enabled", false);

        LinearLayout head = UI.row(requireContext());
        head.addView(UI.tag(requireContext(), enabled ? I18n.text("m_dfb802238b38") : I18n.text("m_f95ea7f4c063"), enabled));
        TextView title = UI.cardTitle(requireContext(), I18n.text("m_cb5e99d1a4d8"));
        UI.margin(title, UI.SM, 0, 0, 0);
        head.addView(title);
        card.addView(head);

        note(card, enabled
                ? I18n.text("m_cf4ef07e86b4") + tf.optInt("recovery_codes_remaining", 0) + I18n.text("m_ed4515a71f08")
                : I18n.text("m_303aeceaf321"), UI.SM);

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
            box.addView(UI.muted(requireContext(), I18n.text("m_28ea3904e014")));
            return box;
        }
        TextView start = UI.button(requireContext(), I18n.text("m_feccf6563bd7"), UI.BTN_PRIMARY);
        start.setOnClickListener(v -> startSetup());
        box.addView(start);
        return box;
    }

    private View setupBlock() {
        LinearLayout box = UI.column(requireContext());
        String secret = tfSetup.optString("secret", "");
        String uri = tfSetup.optString("otpauth_uri", "");

        box.addView(UI.muted(requireContext(), I18n.text("m_e2e92bf46cd5")));
        TextView key = UI.mono(requireContext(), secret, Theme.p().ink);
        key.setTextSize(15);
        key.setTextIsSelectable(true);
        key.setBackground(UI.roundedStroke(Theme.p().canvasSoft2, UI.RADIUS_MD, Theme.p().hairline, 1));
        key.setPadding(UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD));
        UI.margin(key, 0, UI.MD, 0, 0);
        box.addView(key);

        LinearLayout actions = UI.row(requireContext());
        TextView add = UI.button(requireContext(), I18n.text("m_0f63b4519aa9"), UI.BTN_SECONDARY);
        add.setOnClickListener(v -> openAuthenticator(uri, secret));
        UI.weight(add, 1f);
        actions.addView(add);
        TextView copy = UI.button(requireContext(), I18n.text("m_d22d5fd54ca0"), UI.BTN_SECONDARY);
        copy.setOnClickListener(v -> copyText(secret, I18n.text("m_7bd34f6f2054")));
        UI.weight(copy, 1f);
        UI.margin(copy, UI.SM, 0, 0, 0);
        actions.addView(copy);
        UI.margin(actions, 0, UI.MD, 0, 0);
        box.addView(actions);

        tfCodeInput = UI.input(requireContext(), I18n.text("m_97b339ba13ec"));
        tfCodeInput.setInputType(InputType.TYPE_CLASS_NUMBER);
        box.addView(UI.field(requireContext(), I18n.text("m_52517b6304de"), tfCodeInput, UI.MD));

        TextView confirm = UI.button(requireContext(), busy ? I18n.text("m_7f2d0739a240") : I18n.text("m_1e1a5e14d128"), UI.BTN_PRIMARY);
        confirm.setEnabled(!busy);
        confirm.setOnClickListener(v -> confirmSetup(tfCodeInput.getText().toString().trim()));
        UI.margin(confirm, 0, UI.MD, 0, 0);
        box.addView(confirm);

        TextView cancel = UI.button(requireContext(), I18n.text("m_3709bc9325e8"), UI.BTN_GHOST);
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
        box.addView(UI.strong(requireContext(), I18n.text("m_faa3e777003d")));
        box.addView(UI.muted(requireContext(),
                I18n.text("m_472373797aa6")));

        TextView codes = UI.mono(requireContext(), String.join("\n", recoveryCodes), Theme.p().ink);
        codes.setTextSize(15);
        codes.setLineSpacing(UI.dp(4), 1f);
        codes.setTextIsSelectable(true);
        codes.setBackground(UI.roundedStroke(Theme.p().canvasSoft2, UI.RADIUS_MD, Theme.p().hairline, 1));
        codes.setPadding(UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD));
        UI.margin(codes, 0, UI.MD, 0, 0);
        box.addView(codes);

        TextView copy = UI.button(requireContext(), I18n.text("m_75c393923f87"), UI.BTN_SECONDARY);
        copy.setOnClickListener(v -> copyText(String.join("\n", recoveryCodes), I18n.text("m_6f47d1a74036")));
        UI.margin(copy, 0, UI.MD, 0, 0);
        box.addView(copy);

        TextView done = UI.button(requireContext(), I18n.text("m_581139134404"), UI.BTN_PRIMARY);
        done.setOnClickListener(v -> signOut(I18n.text("m_20217ed0a7a5")));
        UI.margin(done, 0, UI.SM, 0, 0);
        box.addView(done);
        return box;
    }

    private View disableTwoFactorBlock() {
        LinearLayout box = UI.column(requireContext());
        box.addView(UI.strong(requireContext(), I18n.text("m_2d971783b5af")));
        note(box, I18n.text("m_5fe4317c5b79"), UI.XS);

        tfPasswordInput = passwordInput(I18n.text("m_a114cfb687e6"));
        box.addView(UI.field(requireContext(), I18n.text("m_a114cfb687e6"), tfPasswordInput, UI.MD));
        tfCodeInput = UI.input(requireContext(), I18n.text("m_97b339ba13ec"));
        tfCodeInput.setInputType(InputType.TYPE_CLASS_NUMBER);
        box.addView(UI.field(requireContext(), I18n.text("m_52517b6304de"), tfCodeInput, UI.MD));

        TextView off = UI.button(requireContext(), busy ? I18n.text("m_5afb5adc5295") : I18n.text("m_4e6fd0e28c55"), UI.BTN_DANGER);
        off.setEnabled(!busy);
        off.setOnClickListener(v -> {
            String password = tfPasswordInput.getText().toString();
            String code = tfCodeInput.getText().toString().trim();
            if (password.isEmpty() || code.isEmpty()) {
                statusMessage = I18n.text("m_256ad4b67d22");
                statusOk = false;
                render();
                return;
            }
            Modal.of(requireContext(), I18n.text("m_4997b109f406"))
                    .message(I18n.text("m_5a37fd588ae2"))
                    .cancel(I18n.text("m_2cd0f3be8738"))
                    .confirm(I18n.text("m_4e6fd0e28c55"), true, () -> disableTwoFactor(password, code))
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
            statusMessage = I18n.text("m_5309145db3c9") + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    private void confirmSetup(String code) {
        if (!code.matches("\\d{6}")) {
            statusMessage = I18n.text("m_ce74946c1891");
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
            statusMessage = I18n.text("m_41ed9c653648") + failure.getMessage();
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
            statusMessage = I18n.text("m_6d26d492839c");
            statusOk = true;
            render();
        }, failure -> {
            busy = false;
            statusMessage = I18n.text("m_a62af649e44f") + failure.getMessage();
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
        copyText(secret, I18n.text("m_153e5e230c12"));
    }

    // ------------------------------------------------------------- cloudflare

    private View cloudflareCard() {
        LinearLayout card = UI.card(requireContext());
        boolean connected = cf.optBoolean("connected", false);
        String source = cf.optString("source", "");

        LinearLayout head = UI.row(requireContext());
        TextView title = UI.cardTitle(requireContext(), I18n.text("m_c3ee7f86565f"));
        head.addView(title, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        head.addView(UI.tag(requireContext(), connected ? I18n.text("m_5be0323e8adc") : I18n.text("m_3d52574ce150"), connected));
        card.addView(head);
        note(card, I18n.text("m_ee436ca7436d"), UI.XS);

        if (!cfError.isEmpty()) {
            TextView error = UI.banner(requireContext(), cfError, true);
            UI.margin(error, 0, UI.MD, 0, 0);
            card.addView(error);
            return card;
        }

        card.addView(UI.divider(requireContext()));
        card.addView(statusRow(I18n.text("m_46cab3bd6fb3"), "oauth".equals(source) ? I18n.text("m_f26b8c653d56") : I18n.text("m_c57b367a8b76")));
        String accountName = cf.optString("account_name", "");
        String accountId = cf.optString("account_id", "");
        card.addView(statusRow(I18n.text("m_2997f476d9bc"), accountName.isEmpty() ? (accountId.isEmpty() ? I18n.text("m_c5768ca9726c") : accountId) : accountName));

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
            TextView start = UI.button(requireContext(), "oauth".equals(source) ? I18n.text("m_978fa225040c") : I18n.text("m_70c33ec4f5aa"), UI.BTN_PRIMARY);
            start.setOnClickListener(v -> startOAuth());
            UI.margin(start, 0, UI.MD, 0, 0);
            card.addView(start);
        } else {
            note(card, I18n.text("m_1eb80f7b3b41"), UI.MD);
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
        if (name.isEmpty()) name = conn.optString("label", I18n.text("m_a5574109f020"));
        text.addView(UI.strong(requireContext(), name));
        String accountId = conn.optString("account_id", "");
        if (!accountId.isEmpty()) {
            text.addView(UI.mono(requireContext(), accountId, Theme.p().mute));
        }
        row.addView(text, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));

        String id = conn.optString("id", "");
        if (active) {
            row.addView(UI.tag(requireContext(), I18n.text("m_fa48e8938940"), true));
        } else {
            TextView use = UI.button(requireContext(), I18n.text("m_21c86745003a"), UI.BTN_GHOST);
            use.setOnClickListener(v -> activateConnection(id));
            row.addView(use);
        }
        TextView remove = UI.button(requireContext(), I18n.text("m_2f9daa828907"), UI.BTN_GHOST);
        remove.setTextColor(Theme.p().error);
        remove.setOnClickListener(v -> Modal.of(requireContext(), I18n.text("m_92e11347ed70"))
                .message(I18n.text("m_6240abcd928b"))
                .cancel(I18n.text("m_2cd0f3be8738"))
                .confirm(I18n.text("m_2f9daa828907"), true, () -> disconnectConnection(id))
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
                statusMessage = I18n.text("m_149201bb7459");
                statusOk = false;
                render();
                return;
            }
            try {
                startActivity(new Intent(Intent.ACTION_VIEW, Uri.parse(url)));
                statusMessage = I18n.text("m_08ab7cea1c23");
                statusOk = true;
            } catch (Exception e) {
                copyText(url, I18n.text("m_12646e7d0d65"));
                statusMessage = I18n.text("m_b88961dbbe8e");
                statusOk = true;
            }
            render();
        }, failure -> {
            busy = false;
            statusMessage = I18n.text("m_b860104232b6") + failure.getMessage();
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
            statusMessage = I18n.text("m_20cb517c36ac");
            statusOk = true;
            reloadCloudflare();
        }, failure -> {
            busy = false;
            statusMessage = I18n.text("m_0c16f8364dbd") + failure.getMessage();
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
                    statusMessage = payload.optString("warning", I18n.text("m_bcb4b0b8810c"));
                    statusOk = true;
                    reloadCloudflare();
                }, failure -> {
                    busy = false;
                    statusMessage = I18n.text("m_0580fcee37e0") + failure.getMessage();
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
        LinearLayout card = card(I18n.text("m_2309fd06dbb1"), I18n.text("m_98fa19ba6d85") + me.optString("username", Session.username()));
        usernameInput = UI.input(requireContext(), I18n.text("m_87de734107bf"));
        card.addView(UI.field(requireContext(), I18n.text("m_0ccf18acc1c8"), usernameInput));
        usernamePasswordInput = passwordInput(I18n.text("m_632da57acf53"));
        card.addView(UI.field(requireContext(), I18n.text("m_ab48ee83149b"), usernamePasswordInput, UI.MD));

        TextView save = UI.button(requireContext(), busy ? I18n.text("m_ff509c9ba052") : I18n.text("m_dbf86ad13caf"), UI.BTN_SECONDARY);
        save.setEnabled(!busy);
        save.setOnClickListener(v -> {
            String value = usernameInput.getText().toString().trim();
            String password = usernamePasswordInput.getText().toString();
            if (value.isEmpty() || password.isEmpty()) {
                statusMessage = I18n.text("m_18de25f0d593");
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
        LinearLayout card = card(I18n.text("m_08d008062411"), I18n.text("m_cd4dc30f1318"));
        currentPasswordInput = passwordInput(I18n.text("m_a114cfb687e6"));
        card.addView(UI.field(requireContext(), I18n.text("m_a114cfb687e6"), currentPasswordInput));
        newPasswordInput = passwordInput(I18n.text("m_515e9c7cf7b2"));
        card.addView(UI.field(requireContext(), I18n.text("m_515e9c7cf7b2"), newPasswordInput, UI.MD));

        TextView save = UI.button(requireContext(), busy ? I18n.text("m_ff509c9ba052") : I18n.text("m_828170843089"), UI.BTN_PRIMARY);
        save.setEnabled(!busy);
        save.setOnClickListener(v -> {
            String current = currentPasswordInput.getText().toString();
            String next = newPasswordInput.getText().toString();
            if (current.isEmpty() || next.isEmpty()) {
                statusMessage = I18n.text("m_824cc55354e0");
                statusOk = false;
                render();
                return;
            }
            if (next.length() < 6) {
                statusMessage = I18n.text("m_56e015fc995a");
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
        }, payload -> signOut(I18n.text("m_dc6cae76cd7a")), failure -> {
            busy = false;
            statusMessage = I18n.text("m_4f2e8a8d4cb3") + failure.getMessage();
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
        }, payload -> signOut(I18n.text("m_e559ccc5f437")), failure -> {
            busy = false;
            statusMessage = I18n.text("m_4f2e8a8d4cb3") + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    // -------------------------------------------------------------- passkeys

    /**
     * Passkeys: bind, rename and remove credentials, and switch password sign-in
     * off for this account.
     *
     * The panel resolves the relying party, so this card only reports what it
     * answers and hides the actions when the domain cannot run WebAuthn at all
     * (a bare IP address or a plain http origin, for instance).
     */
    private View passkeyCard() {
        Context ctx = requireContext();
        LinearLayout card = card(I18n.text("m_102824e51a9d"), I18n.text("m_9ed4805b36b1"));

        if (!Passkey.supported()) {
            card.addView(UI.banner(ctx, I18n.text("m_b47a9b615baf"), true));
            return card;
        }
        if (!pk.optBoolean("available", false)) {
            card.addView(UI.banner(ctx,
                    I18n.text("m_8d480e7296c1"), true));
            return card;
        }

        JSONArray list = pk.optJSONArray("passkeys");
        if (list == null || list.length() == 0) {
            card.addView(UI.muted(ctx, I18n.text("m_709aa2bbba49") + pk.optString("rp_id", "—")));
        } else {
            for (int i = 0; i < list.length(); i++) {
                JSONObject item = list.optJSONObject(i);
                if (item == null) continue;
                if (i > 0) card.addView(UI.divider(ctx));
                card.addView(passkeyRow(item));
            }
        }

        pkNameInput = UI.input(ctx, I18n.text("m_dddb4afac36b"));
        card.addView(UI.field(ctx, I18n.text("m_7c21d524aa00"), pkNameInput, UI.MD));
        pkPasswordInput = UI.input(ctx, I18n.text("m_a114cfb687e6"));
        pkPasswordInput.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_PASSWORD);
        card.addView(UI.field(ctx, I18n.text("m_a114cfb687e6"), pkPasswordInput, UI.SM));
        card.addView(UI.muted(ctx, I18n.text("m_25c2dceabf57")));

        TextView add = UI.button(ctx, pkBusy ? I18n.text("m_d3d21191f32e") : I18n.text("m_a24e4cc96fdc"), UI.BTN_PRIMARY);
        add.setEnabled(!pkBusy);
        add.setOnClickListener(v -> addPasskey());
        UI.margin(add, 0, UI.MD, 0, 0);
        card.addView(add);

        card.addView(UI.divider(ctx));

        boolean off = pk.optBoolean("password_login_disabled", false);
        LinearLayout switchRow = UI.row(ctx);
        switchRow.setGravity(Gravity.CENTER_VERTICAL);
        LinearLayout copy = UI.column(ctx);
        copy.addView(UI.strong(ctx, I18n.text("m_3a027241650b")));
        TextView hint = UI.muted(ctx, off
                ? I18n.text("m_ca044638c719")
                : I18n.text("m_7ec5421a8ae6"));
        UI.margin(hint, 0, 2, 0, 0);
        copy.addView(hint);
        if (pk.optBoolean("password_login_disabled_globally", false)) {
            TextView global = UI.muted(ctx, I18n.text("m_9b7d052ac30d"));
            UI.margin(global, 0, 2, 0, 0);
            copy.addView(global);
        }
        UI.weight(copy, 1f);
        switchRow.addView(copy);

        TextView toggle = UI.button(ctx, off ? I18n.text("m_466c8603a96a") : I18n.text("m_3a027241650b"),
                off ? UI.BTN_SECONDARY : UI.BTN_DANGER);
        toggle.setEnabled(!pkBusy);
        toggle.setOnClickListener(v -> togglePasswordLogin());
        switchRow.addView(toggle);
        UI.margin(switchRow, 0, UI.MD, 0, 0);
        card.addView(switchRow);

        return card;
    }

    private View passkeyRow(JSONObject item) {
        Context ctx = requireContext();
        LinearLayout row = UI.row(ctx);
        row.setGravity(Gravity.CENTER_VERTICAL);

        LinearLayout info = UI.column(ctx);
        LinearLayout nameRow = UI.row(ctx);
        nameRow.setGravity(Gravity.CENTER_VERTICAL);
        nameRow.addView(UI.strong(ctx, item.optString("name", I18n.text("m_102824e51a9d"))));
        if (item.optBoolean("backup_eligible", false)) {
            TextView tag = UI.tag(ctx, item.optBoolean("backup_state", false) ? I18n.text("m_04062edb583b") : I18n.text("m_f450fddb8f2f"), true);
            UI.margin(tag, UI.SM, 0, 0, 0);
            nameRow.addView(tag);
        }
        info.addView(nameRow);
        info.addView(UI.muted(ctx, passkeyMeta(item)));
        UI.weight(info, 1f);
        row.addView(info);

        TextView rename = UI.button(ctx, I18n.text("m_0d0cbac2eee5"), UI.BTN_GHOST);
        rename.setEnabled(!pkBusy);
        rename.setOnClickListener(v -> renamePasskey(item));
        row.addView(rename);

        TextView remove = UI.button(ctx, I18n.text("m_2f9daa828907"), UI.BTN_GHOST);
        remove.setTextColor(Theme.p().error);
        remove.setEnabled(!pkBusy);
        remove.setOnClickListener(v -> confirmRemovePasskey(item));
        UI.margin(remove, UI.XS, 0, 0, 0);
        row.addView(remove);

        return row;
    }

    /** "添加于 … · 最近使用 …", in the console's local time. */
    private String passkeyMeta(JSONObject item) {
        StringBuilder meta = new StringBuilder(I18n.text("m_749cc5f884d6")).append(stamp(item.optLong("created_at", 0)));
        long used = item.optLong("last_used_at", 0);
        if (used > 0) meta.append(I18n.text("m_3219f9714d97")).append(stamp(used));
        return meta.toString();
    }

    private static String stamp(long seconds) {
        if (seconds <= 0) return "—";
        java.text.DateFormat format = java.text.DateFormat.getDateTimeInstance(
                java.text.DateFormat.SHORT, java.text.DateFormat.SHORT);
        return format.format(new java.util.Date(seconds * 1000L));
    }

    private void addPasskey() {
        String password = pkPassword();
        if (password.isEmpty()) {
            statusMessage = I18n.text("m_05b1298143b6");
            statusOk = false;
            render();
            return;
        }
        final String name = pkNameInput == null ? "" : pkNameInput.getText().toString().trim();
        pkBusy = true;
        statusMessage = "";
        render();
        Api.async(() -> {
            JSONObject body = new JSONObject();
            body.put("password", password);
            return Api.post("/api/account/passkeys/begin", body);
        }, begin -> Passkey.register(requireContext(), begin.optJSONObject("public_key"), new Passkey.Callback() {
            @Override
            public void onResult(JSONObject credential) {
                finishAddPasskey(begin.optString("ceremony_token", ""), name, credential);
            }

            @Override
            public void onError(Exception error) {
                pkBusy = false;
                statusMessage = error.getMessage();
                statusOk = false;
                render();
            }
        }), failure -> {
            pkBusy = false;
            statusMessage = I18n.text("m_4efd22ea6489") + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    private void finishAddPasskey(String ceremonyToken, String name, JSONObject credential) {
        Api.async(() -> {
            JSONObject body = new JSONObject();
            body.put("ceremony_token", ceremonyToken);
            body.put("name", name);
            body.put("credential", credential);
            return Api.post("/api/account/passkeys/finish", body);
        }, payload -> {
            pkBusy = false;
            statusMessage = I18n.text("m_716a1862dd06");
            statusOk = true;
            reloadPasskeys();
        }, failure -> {
            pkBusy = false;
            statusMessage = I18n.text("m_4efd22ea6489") + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    private void renamePasskey(JSONObject item) {
        final String id = item.optString("id", "");
        EditText input = UI.input(requireContext(), I18n.text("m_e8e46cb47287"));
        input.setText(item.optString("name", ""));
        Modal.of(requireContext(), I18n.text("m_c384f03c2393"))
                .content(input)
                .cancel(I18n.text("m_2cd0f3be8738"))
                .confirm(I18n.text("m_a3030bf8f16d"), false, () -> {
                    String name = input.getText().toString().trim();
                    if (name.isEmpty()) return;
                    pkBusy = true;
                    render();
                    Api.async(() -> {
                        JSONObject body = new JSONObject();
                        body.put("name", name);
                        return Api.put("/api/account/passkeys/" + id, body);
                    }, payload -> {
                        pkBusy = false;
                        statusMessage = I18n.text("m_06afd1f0352e");
                        statusOk = true;
                        reloadPasskeys();
                    }, failure -> {
                        pkBusy = false;
                        statusMessage = I18n.text("m_0ab10c33f462") + failure.getMessage();
                        statusOk = false;
                        render();
                    });
                })
                .show();
    }

    private void confirmRemovePasskey(JSONObject item) {
        Modal.of(requireContext(), I18n.text("m_fc3f78b499e1"))
                .message(I18n.text("m_72f49a525798") + item.optString("name", I18n.text("m_102824e51a9d")) + I18n.text("m_139f37a3ae7c"))
                .cancel(I18n.text("m_2cd0f3be8738"))
                .confirm(I18n.text("m_2f9daa828907"), true, () -> removePasskey(item.optString("id", "")))
                .show();
    }

    private void removePasskey(String id) {
        String password = pkPassword();
        if (password.isEmpty()) {
            statusMessage = I18n.text("m_05b1298143b6");
            statusOk = false;
            render();
            return;
        }
        pkBusy = true;
        statusMessage = "";
        render();
        Api.async(() -> {
            JSONObject body = new JSONObject();
            body.put("password", password);
            return Api.delete("/api/account/passkeys/" + id, body);
        }, payload -> {
            pkBusy = false;
            statusMessage = I18n.text("m_dbef251d5058");
            statusOk = true;
            reloadPasskeys();
        }, failure -> {
            pkBusy = false;
            statusMessage = failure.getMessage();
            statusOk = false;
            render();
        });
    }

    private void togglePasswordLogin() {
        String password = pkPassword();
        if (password.isEmpty()) {
            statusMessage = I18n.text("m_05b1298143b6");
            statusOk = false;
            render();
            return;
        }
        final boolean disabled = !pk.optBoolean("password_login_disabled", false);
        pkBusy = true;
        statusMessage = "";
        render();
        Api.async(() -> {
            JSONObject body = new JSONObject();
            body.put("disabled", disabled);
            body.put("password", password);
            return Api.put("/api/account/password-login", body);
        }, payload -> {
            pkBusy = false;
            pk = payload;
            statusMessage = disabled ? I18n.text("m_7cda7e6a2511") : I18n.text("m_40ea52edefce");
            statusOk = true;
            pkPasswordInput = null;
            render();
        }, failure -> {
            pkBusy = false;
            statusMessage = failure.getMessage();
            statusOk = false;
            render();
        });
    }

    private void reloadPasskeys() {
        pkBusy = true;
        render();
        Api.async(() -> Api.get("/api/account/passkeys"), payload -> {
            pkBusy = false;
            pk = payload;
            pkPasswordInput = null;
            pkNameInput = null;
            render();
        }, failure -> {
            pkBusy = false;
            statusMessage = I18n.text("m_38da6e5f2d40") + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    private String pkPassword() {
        return pkPasswordInput == null ? "" : pkPasswordInput.getText().toString();
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
