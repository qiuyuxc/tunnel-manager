package com.tunnelmanager.app;

import android.Manifest;
import android.content.Intent;
import android.content.SharedPreferences;
import android.content.pm.PackageManager;
import android.graphics.Typeface;
import android.net.Uri;
import android.os.Build;
import android.os.PowerManager;
import android.provider.Settings;
import android.text.InputType;
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
import androidx.core.content.ContextCompat;

import org.json.JSONArray;
import org.json.JSONObject;

import java.util.LinkedHashSet;
import java.util.Set;

/**
 * 通知设置 — the server-side channels plus the phone-only background pusher.
 *
 * The web page is one long scroll of small cards; on the phone the native-only
 * "本机通知" card sits first because it is the one thing the browser cannot do,
 * and everything under it is the same four decisions: where to send, what to
 * send, who to email, and which bot token to use.
 */
public class NotificationsFragment extends PageFragment {

    private static final int REQ_NOTIFY = 71;
    private static final String[] CHANNEL_LABELS = {"关闭通知", "仅邮箱", "仅 Telegram", "邮箱 + Telegram"};

    private ScrollView scroll;
    private LinearLayout body;

    private SharedPreferences prefs;

    private final Set<String> channels = new LinkedHashSet<>();
    private boolean loginEvent = true;
    private boolean tokenSet = false;
    private boolean remoteBotSet = false;

    /** Drafts survive a re-render, which is what a toggle press triggers. */
    private String emailsDraft = "";
    private String tokenDraft = "";
    private String chatDraft = "";

    private EditText emailsInput;
    private EditText tokenInput;
    private EditText chatInput;

    private boolean loaded = false;
    private String loadError = "";
    private String statusMessage = "";
    private boolean statusOk = true;
    private boolean busy = false;

    @Override
    String route() {
        return "/notifications";
    }

    @Override
    protected View build(@NonNull LayoutInflater inflater, @Nullable ViewGroup container) {
        prefs = requireContext().getSharedPreferences(MainActivity.PREFS, android.content.Context.MODE_PRIVATE);
        scroll = new ScrollView(requireContext());
        body = UI.column(requireContext());
        UI.pagePadding(body);
        scroll.addView(body);

        body.addView(UI.pageTitle(requireContext(), "通知设置"));
        body.addView(UI.muted(requireContext(), "加载中…"));
        load();
        return scroll;
    }

    // ------------------------------------------------------------------- data

    private void load() {
        Api.async(() -> Api.get("/api/notify/settings"), settings -> {
            applySettings(settings);
            loaded = true;
            loadError = "";
            render();
        }, failure -> {
            loadError = failure.getMessage();
            render();
        });
    }

    private void applySettings(JSONObject settings) {
        channels.clear();
        JSONArray list = settings.optJSONArray("channels");
        if (list != null) {
            for (int i = 0; i < list.length(); i++) {
                channels.add(list.optString(i));
            }
        }
        JSONObject events = settings.optJSONObject("events");
        loginEvent = events == null || events.optBoolean("login", false);
        emailsDraft = settings.optString("emails", "");
        tokenDraft = "";
        tokenSet = settings.optBoolean("tg_bot_token_set", false);
        remoteBotSet = settings.optBoolean("tg_remote_bot_set", false);
        chatDraft = settings.optString("tg_notify_chat_id", "");
    }

    /** The four web radio options, expressed as the channel set they produce. */
    private int channelChoice() {
        boolean email = channels.contains("email");
        boolean telegram = channels.contains("telegram");
        if (email && telegram) return 3;
        if (email) return 1;
        if (telegram) return 2;
        return 0;
    }

    private void setChannelChoice(int index) {
        channels.clear();
        if (index == 1 || index == 3) channels.add("email");
        if (index == 2 || index == 3) channels.add("telegram");
    }

    // -------------------------------------------------------------- rendering

    private void captureDrafts() {
        if (emailsInput != null) emailsDraft = emailsInput.getText().toString();
        if (tokenInput != null) tokenDraft = tokenInput.getText().toString();
        if (chatInput != null) chatDraft = chatInput.getText().toString();
    }

    private void render() {
        if (body == null || !alive()) return;
        // Toggling a switch or a channel rebuilds the page; without this the
        // scroll would snap back to the title every time.
        int keepScroll = scroll.getScrollY();
        body.removeAllViews();

        body.addView(UI.pageTitle(requireContext(), "通知设置"));
        TextView subtitle = UI.muted(requireContext(), "选择通知渠道与事件，登录成功等事件会按你的配置发送提醒");
        UI.margin(subtitle, 0, UI.XS, 0, UI.MD);
        body.addView(subtitle);

        if (!loadError.isEmpty()) {
            body.addView(UI.banner(requireContext(), "加载失败：" + loadError, true));
            body.addView(UI.spacer(requireContext(), UI.MD));
        }

        body.addView(hostCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(channelCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(eventsCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(emailsCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(telegramCard());
        body.addView(UI.spacer(requireContext(), UI.LG));
        body.addView(actionRow());

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

    /** Title + description on the left, a control on the right. */
    private LinearLayout settingRow(String title, String desc, View control) {
        LinearLayout row = UI.row(requireContext());
        LinearLayout text = UI.column(requireContext());
        text.addView(UI.strong(requireContext(), title));
        if (desc != null && !desc.isEmpty()) {
            TextView hint = UI.muted(requireContext(), desc);
            UI.margin(hint, 0, UI.XS, 0, 0);
            text.addView(hint);
        }
        row.addView(text, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        if (control != null) {
            LinearLayout.LayoutParams lp = control instanceof UI.Toggle
                    ? new LinearLayout.LayoutParams(UI.dp(34), UI.dp(20))
                    : new LinearLayout.LayoutParams(
                            ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);
            lp.leftMargin = UI.dp(UI.MD);
            row.addView(control, lp);
        }
        return row;
    }

    private LinearLayout rowsCard(String title, String desc) {
        LinearLayout card = UI.card(requireContext());
        card.addView(cardHead(title, desc));
        return card;
    }

    // ------------------------------------------------------------- host card

    private View hostCard() {
        LinearLayout card = rowsCard("本机通知",
                "由 App 自己在后台轮询服务器，不用开着浏览器也能收到告警；只在这台设备生效，需要手动开启。");

        boolean enabled = hostEnabled();
        boolean permission = notificationsAllowed();

        LinearLayout push = settingRow("后台告警推送", hostHint(enabled, permission), hostToggle(enabled));
        card.addView(push);

        LinearLayout batteryRow = settingRow("忽略电池优化",
                "系统在息屏后可能限制后台轮询，允许后告警更及时。", null);
        TextView battery = UI.button(requireContext(),
                ignoringBattery() ? "已允许" : "去允许", UI.BTN_SECONDARY);
        battery.setOnClickListener(v -> openBatterySettings());
        LinearLayout.LayoutParams blp = new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);
        blp.leftMargin = UI.dp(UI.MD);
        batteryRow.addView(battery, blp);
        UI.addRow(card, batteryRow, UI.MD);

        TextView note = UI.muted(requireContext(),
                "推送与邮件告警同源：需要在具体监控项目里开启「告警」，且只在状态发生变化时触发。");
        UI.margin(note, 0, UI.MD, 0, 0);
        card.addView(note);

        TextView test = UI.button(requireContext(), "发送本机测试通知", UI.BTN_SECONDARY);
        test.setOnClickListener(v -> AlertService.postTest(requireContext()));
        UI.margin(test, 0, UI.MD, 0, 0);
        card.addView(test);
        return card;
    }

    private String hostHint(boolean enabled, boolean permission) {
        if (!permission) return "系统还没允许本机发送通知，打开开关时会弹出授权。";
        if (!enabled) return "已关闭。开启后 App 会常驻一条低调通知来维持轮询。";
        if (!ignoringBattery()) return "运行中；建议同时允许忽略电池优化，避免息屏后被系统掐掉。";
        return "运行中，服务端一有状态变化就会推送到这里。";
    }

    private UI.Toggle hostToggle(boolean enabled) {
        UI.Toggle toggle = new UI.Toggle(requireContext(), enabled);
        LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(UI.dp(34), UI.dp(20));
        lp.leftMargin = UI.dp(UI.MD);
        toggle.setLayoutParams(lp);
        toggle.setClickable(true);
        toggle.setOnClickListener(v -> {
            boolean want = !toggle.isOn();
            toggle.setOn(want);
            captureDrafts();
            if (want) enableHost();
            else disableHost();
            render();
        });
        return toggle;
    }

    // --------------------------------------------------------- channel cards

    private View channelCard() {
        LinearLayout card = rowsCard("通知渠道",
                "可选邮箱、Telegram，或两者同时发送；选择「关闭通知」则不会发送任何提醒。");
        int choice = channelChoice();
        for (int i = 0; i < CHANNEL_LABELS.length; i++) {
            final int index = i;
            card.addView(channelOption(CHANNEL_LABELS[i], index == choice, () -> {
                captureDrafts();
                setChannelChoice(index);
                render();
            }));
            if (i < CHANNEL_LABELS.length - 1) card.addView(UI.spacer(requireContext(), UI.SM));
        }
        return card;
    }

    /** The web's `.channel-option`: a radio dot and its label, full width. */
    private View channelOption(String label, boolean active, Runnable onClick) {
        Palette p = Theme.p();
        LinearLayout row = UI.row(requireContext());
        row.setPadding(UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD));
        row.setBackground(UI.pressable(UI.roundedStroke(
                active ? p.canvasSoft2 : p.canvasRaised,
                UI.RADIUS_MD,
                active ? p.ink : p.hairline,
                1), p.btnGhostHover));

        View dot = new View(requireContext());
        dot.setBackground(UI.circle(active ? p.ink : 0, active ? p.ink : p.hairlineStrong, 1.5f));
        row.addView(dot, new LinearLayout.LayoutParams(UI.dp(16), UI.dp(16)));

        TextView text = UI.text(requireContext(), label, 14, active ? p.ink : p.body,
                active ? Typeface.BOLD : Typeface.NORMAL);
        LinearLayout.LayoutParams tlp = new LinearLayout.LayoutParams(
                0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
        tlp.leftMargin = UI.dp(UI.MD);
        row.addView(text, tlp);

        row.setClickable(true);
        row.setOnClickListener(v -> onClick.run());
        return row;
    }

    private View eventsCard() {
        LinearLayout card = rowsCard("通知事件", "开启的事件才会发送提醒。");
        UI.Toggle toggle = new UI.Toggle(requireContext(), loginEvent);
        LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(UI.dp(34), UI.dp(20));
        lp.leftMargin = UI.dp(UI.MD);
        toggle.setLayoutParams(lp);
        toggle.setClickable(true);
        toggle.setOnClickListener(v -> {
            captureDrafts();
            loginEvent = !toggle.isOn();
            toggle.setOn(loginEvent);
            render();
        });
        card.addView(settingRow("登录通知", "每次账户登录成功后发送提醒（含时间与 IP）。", toggle));
        return card;
    }

    private View emailsCard() {
        LinearLayout card = rowsCard("邮箱收件人", "每行一个邮箱，支持多个收件人；仅在启用邮箱渠道时生效。");
        emailsInput = UI.textArea(requireContext(), "you@example.com\nops@example.com");
        emailsInput.setText(emailsDraft);
        UI.fill(emailsInput);
        card.addView(emailsInput);
        return card;
    }

    private View telegramCard() {
        LinearLayout card = rowsCard("Telegram 机器人",
                "使用你自己的 Bot（由 @BotFather 创建），与面板管理员无关；仅在启用 Telegram 渠道时生效。");

        tokenInput = UI.input(requireContext(), tokenSet ? "留空保持不变" : "123456:ABC…");
        tokenInput.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_PASSWORD);
        tokenInput.setText(tokenDraft);
        card.addView(UI.field(requireContext(), "Bot Token", tokenInput));

        TextView tag = UI.tag(requireContext(), tokenSet ? "已设置" : "未设置", tokenSet);
        UI.margin(tag, 0, UI.SM, 0, UI.MD);
        card.addView(tag);

        chatInput = UI.input(requireContext(), "123456789");
        chatInput.setText(chatDraft);
        card.addView(UI.field(requireContext(), "接收 Chat ID", chatInput));

        TextView hint = UI.muted(requireContext(),
                "获取方式：先向自己的 Bot 发送任意消息，再打开 https://api.telegram.org/bot<TOKEN>/getUpdates ，取返回结果中的 chat.id。");
        UI.margin(hint, 0, UI.MD, 0, 0);
        card.addView(hint);

        if (!tokenSet && remoteBotSet) {
            TextView reuse = UI.button(requireContext(),
                    busy ? "复用中…" : "一键复用远程控制的 Bot", UI.BTN_SECONDARY);
            reuse.setEnabled(!busy);
            reuse.setOnClickListener(v -> reuseFromTelegram());
            UI.margin(reuse, 0, UI.MD, 0, 0);
            card.addView(reuse);
        }
        return card;
    }

    private View actionRow() {
        LinearLayout row = UI.row(requireContext());
        row.setGravity(Gravity.CENTER_VERTICAL);

        TextView save = UI.button(requireContext(), busy ? "保存中…" : "保存设置", UI.BTN_PRIMARY);
        save.setEnabled(!busy);
        save.setOnClickListener(v -> save());
        UI.weight(save, 1f);
        row.addView(save);

        TextView test = UI.button(requireContext(), "发送测试通知", UI.BTN_SECONDARY);
        test.setOnClickListener(v -> sendTest());
        UI.weight(test, 1f);
        UI.margin(test, UI.SM, 0, 0, 0);
        row.addView(test);
        return row;
    }

    // ------------------------------------------------------------ host state

    private boolean hostEnabled() {
        return prefs != null && prefs.getBoolean(AlertService.KEY_ENABLED, false);
    }

    private boolean notificationsAllowed() {
        if (Build.VERSION.SDK_INT < 33) return true;
        return ContextCompat.checkSelfPermission(requireContext(), Manifest.permission.POST_NOTIFICATIONS)
                == PackageManager.PERMISSION_GRANTED;
    }

    private boolean ignoringBattery() {
        PowerManager pm = (PowerManager) requireContext().getSystemService(android.content.Context.POWER_SERVICE);
        return pm != null && pm.isIgnoringBatteryOptimizations(requireContext().getPackageName());
    }

    private void enableHost() {
        if (!notificationsAllowed()) {
            requestPermissions(new String[]{Manifest.permission.POST_NOTIFICATIONS}, REQ_NOTIFY);
            return;
        }
        startAlerts();
    }

    private void startAlerts() {
        try {
            Intent intent = new Intent(requireContext(), AlertService.class);
            intent.setAction(MainActivity.ACTION_START_ALERTS);
            ContextCompat.startForegroundService(requireContext(), intent);
            prefs.edit().putBoolean(AlertService.KEY_ENABLED, true).apply();
        } catch (Exception e) {
            statusMessage = "无法启动后台告警：" + e.getMessage();
            statusOk = false;
        }
    }

    private void disableHost() {
        requireContext().stopService(new Intent(requireContext(), AlertService.class));
        prefs.edit().putBoolean(AlertService.KEY_ENABLED, false).apply();
    }

    private void openBatterySettings() {
        try {
            Intent intent = new Intent(Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS);
            intent.setData(Uri.parse("package:" + requireContext().getPackageName()));
            startActivity(intent);
        } catch (Exception e) {
            // Some OEM builds hide that screen; the app's own settings page is
            // the closest thing that always exists.
            try {
                startActivity(new Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS,
                        Uri.parse("package:" + requireContext().getPackageName())));
            } catch (Exception ignored) {
                statusMessage = "无法打开电池优化设置，请到系统设置里手动允许。";
                statusOk = false;
                render();
            }
        }
    }

    @Override
    public void onRequestPermissionsResult(int requestCode, @NonNull String[] permissions,
                                           @NonNull int[] grantResults) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults);
        if (requestCode != REQ_NOTIFY) return;
        if (grantResults.length > 0 && grantResults[0] == PackageManager.PERMISSION_GRANTED) {
            startAlerts();
        } else {
            statusMessage = "系统未允许通知权限，后台告警无法开启。";
            statusOk = false;
        }
        render();
    }

    @Override
    void onShown() {
        // Coming back from the battery-optimisation screen: its row is stale.
        if (loaded) render();
    }

    // ---------------------------------------------------------------- actions

    private void save() {
        captureDrafts();
        busy = true;
        statusMessage = "";
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("channels", new JSONArray(channels));
            JSONObject events = new JSONObject();
            events.put("login", loginEvent);
            payload.put("events", events);
            payload.put("emails", emailsDraft);
            if (!tokenDraft.isEmpty()) payload.put("tg_bot_token", tokenDraft);
            payload.put("tg_notify_chat_id", chatDraft.trim());
            return Api.put("/api/notify/settings", payload);
        }, settings -> {
            busy = false;
            applySettings(settings);
            statusMessage = "通知设置已保存";
            statusOk = true;
            render();
        }, failure -> {
            busy = false;
            statusMessage = "保存失败：" + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    private void sendTest() {
        captureDrafts();
        statusMessage = "";
        render();
        Api.async(() -> Api.post("/api/notify/test", null), ok -> {
            statusMessage = "测试通知已发送，请检查配置的渠道";
            statusOk = true;
            render();
        }, failure -> {
            statusMessage = "发送失败：" + failure.getMessage();
            statusOk = false;
            render();
        });
    }

    private void reuseFromTelegram() {
        captureDrafts();
        busy = true;
        render();
        Api.async(() -> Api.post("/api/notify/reuse", null), settings -> {
            busy = false;
            applySettings(settings);
            statusMessage = "已复用远程控制的 Bot Token";
            statusOk = true;
            render();
        }, failure -> {
            busy = false;
            statusMessage = "复用失败：" + failure.getMessage();
            statusOk = false;
            render();
        });
    }
}
