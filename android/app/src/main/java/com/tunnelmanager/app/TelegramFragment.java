package com.tunnelmanager.app;

import android.graphics.Typeface;
import android.os.Handler;
import android.os.Looper;
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

import org.json.JSONObject;

/**
 * TG 机器人 — one bot per account.
 *
 * Everything on the page is a setting the server applies on save, so the layout
 * follows the web's order. The status card polls while the page is open, since
 * "is the bot alive" is the one thing the operator keeps coming back to check.
 */
public class TelegramFragment extends PageFragment {

    private static final long STATUS_INTERVAL = 10_000L;
    private static final String[] COMMANDS = {
            "/当前配置", "/列出隧道", "/选择隧道", "/转发", "/直连域名", "/优选绑定", "/绑定域名",
            "/列出区域", "/DNS列表", "/DNS详情", "/DNS添加", "/DNS修改", "/DNS删除", "/确认删除",
            "/全局优选", "/设置回退源", "/help",
    };

    private ScrollView scroll;
    private LinearLayout body;
    private final Handler ui = new Handler(Looper.getMainLooper());

    private JSONObject settings = new JSONObject();
    private JSONObject status = new JSONObject();
    private String statusError = "";

    /** Index into {@link #MODES}; mirrors the web's polling / webhook radios. */
    private static final String[] MODES = {"polling", "webhook"};

    private boolean enabled = false;
    private int mode = 0;
    private String tokenDraft = "";
    private String chatIds = "";
    private String webhookUrl = "";
    private String apiEndpoint = "";

    private EditText tokenInput;
    private EditText idsInput;
    private EditText webhookInput;
    private EditText endpointInput;

    private boolean loading = true;
    private boolean busy = false;
    private boolean notifyBotSet = false;
    private String banner = "";
    private boolean bannerError = true;

    private final Runnable pollTask = new Runnable() {
        @Override
        public void run() {
            if (!isAdded()) return;
            loadStatus();
            ui.postDelayed(this, STATUS_INTERVAL);
        }
    };

    @Override
    String route() {
        return "/telegram";
    }

    @Override
    protected View build(@NonNull LayoutInflater inflater, @Nullable ViewGroup container) {
        scroll = new ScrollView(requireContext());
        body = UI.column(requireContext());
        UI.pagePadding(body);
        scroll.addView(body);
        render();
        load();
        ui.postDelayed(pollTask, STATUS_INTERVAL);
        return scroll;
    }

    @Override
    public void onDestroyView() {
        ui.removeCallbacks(pollTask);
        super.onDestroyView();
    }

    // ------------------------------------------------------------------- data

    private void load() {
        loading = true;
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("settings", Api.get("/api/telegram/settings"));
            payload.put("status", Api.get("/api/telegram/status"));
            return payload;
        }, payload -> {
            loading = false;
            applySettings(payload.optJSONObject("settings"));
            status = orEmpty(payload.optJSONObject("status"));
            statusError = "";
            render();
        }, failure -> {
            loading = false;
            banner = I18n.text("m_01de8216e0d0") + failure.getMessage();
            bannerError = true;
            render();
        });
    }

    private static JSONObject orEmpty(JSONObject value) {
        return value == null ? new JSONObject() : value;
    }

    private void applySettings(JSONObject payload) {
        JSONObject data = orEmpty(payload);
        settings = data;
        enabled = data.optBoolean("enabled", false);
        mode = "webhook".equals(data.optString("mode", "")) ? 1 : 0;
        chatIds = data.optString("admin_tg_ids", "");
        webhookUrl = data.optString("webhook_url", "");
        apiEndpoint = data.optString("api_endpoint", "");
        notifyBotSet = data.optBoolean("notify_bot_set", false);
        tokenDraft = "";
    }

    private void loadStatus() {
        Api.async(() -> Api.get("/api/telegram/status"), payload -> {
            status = payload;
            statusError = "";
            render();
        }, failure -> {
            statusError = failure.getMessage();
            render();
        });
    }

    // -------------------------------------------------------------- rendering

    private void capture() {
        if (tokenInput != null) tokenDraft = tokenInput.getText().toString();
        if (idsInput != null) chatIds = idsInput.getText().toString();
        if (webhookInput != null) webhookUrl = webhookInput.getText().toString();
        if (endpointInput != null) apiEndpoint = endpointInput.getText().toString();
    }

    private void render() {
        if (body == null || !alive()) return;
        int keepScroll = scroll.getScrollY();
        body.removeAllViews();

        body.addView(UI.pageTitle(requireContext(), I18n.text("m_3ba721cb5dd3")));
        TextView subtitle = UI.muted(requireContext(),
                I18n.text("m_d4463e3d9fb2"));
        UI.margin(subtitle, 0, UI.XS, 0, UI.MD);
        body.addView(subtitle);

        if (loading) {
            body.addView(UI.muted(requireContext(), I18n.text("m_4927a53bcc88")));
            return;
        }

        body.addView(statusCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(enableCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(modeCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(tokenCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(idsCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(endpointCard());
        body.addView(UI.spacer(requireContext(), UI.LG));
        body.addView(actionsCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(guideCard());

        if (!banner.isEmpty()) {
            TextView view = UI.banner(requireContext(), banner, bannerError);
            UI.margin(view, 0, UI.MD, 0, 0);
            body.addView(view);
        }
        UI.restoreScroll(scroll, keepScroll);
    }

    private LinearLayout card(String title, String desc) {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), title));
        if (desc != null && !desc.isEmpty()) {
            TextView hint = UI.muted(requireContext(), desc);
            UI.margin(hint, 0, UI.XS, 0, UI.MD);
            card.addView(hint);
        }
        return card;
    }

    private EditText passwordInput(String hint) {
        EditText input = UI.input(requireContext(), hint);
        input.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_PASSWORD);
        return input;
    }

    // ------------------------------------------------------------------ cards

    private View statusCard() {
        LinearLayout card = UI.card(requireContext());
        LinearLayout head = UI.row(requireContext());
        head.addView(UI.cardTitle(requireContext(), I18n.text("m_6888e1669b1b")),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        TextView refresh = UI.button(requireContext(), I18n.text("m_aee887434131"), UI.BTN_GHOST);
        refresh.setOnClickListener(v -> loadStatus());
        head.addView(refresh);
        card.addView(head);

        boolean running = status.optBoolean("running", false);
        Palette p = Theme.p();
        LinearLayout line = UI.row(requireContext());
        View dot = new View(requireContext());
        dot.setBackground(UI.circle(running ? p.statusHealthyText : p.hairlineStrong, 0, 0));
        line.addView(dot, new LinearLayout.LayoutParams(UI.dp(9), UI.dp(9)));
        String username = status.optString("bot_username", "");
        String text = running
                ? I18n.text("m_1f0eb99b7ed0") + (username.isEmpty() ? "" : " @" + username) + " · " + modeLabel()
                : I18n.text("m_f006455e3baf");
        TextView label = UI.body(requireContext(), text);
        LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
        lp.leftMargin = UI.dp(UI.SM);
        line.addView(label, lp);
        UI.margin(line, 0, 0, 0, 0);
        card.addView(line);

        if (!statusError.isEmpty()) {
            card.addView(UI.banner(requireContext(), statusError, true));
        } else if (!status.optString("last_error", "").isEmpty()) {
            TextView error = UI.banner(requireContext(), status.optString("last_error"), true);
            UI.margin(error, 0, UI.MD, 0, 0);
            card.addView(error);
        }
        String lastUpdate = status.optString("last_update_at", "");
        if (!lastUpdate.isEmpty()) {
            TextView meta = UI.muted(requireContext(), I18n.text("m_b6eb0a561692") + lastUpdate);
            UI.margin(meta, 0, UI.SM, 0, 0);
            card.addView(meta);
        }
        return card;
    }

    private String modeLabel() {
        return status.optString("mode", MODES[mode]).equals("webhook") ? I18n.text("m_96011210ccb6") : I18n.text("m_d1d1fb7d3129");
    }

    private View enableCard() {
        LinearLayout card = card(I18n.text("m_13df38f938ce"), I18n.text("m_80de49d10357"));
        LinearLayout row = UI.row(requireContext());
        UI.Toggle toggle = new UI.Toggle(requireContext(), enabled);
        toggle.setOnClickListener(v -> {
            capture();
            enabled = !toggle.isOn();
            toggle.setOn(enabled);
            render();
        });
        row.addView(toggle, new LinearLayout.LayoutParams(UI.dp(34), UI.dp(20)));
        TextView label = UI.body(requireContext(), enabled ? I18n.text("m_dfb802238b38") : I18n.text("m_bc5a87a757a5"));
        UI.margin(label, UI.MD, 0, 0, 0);
        row.addView(label);
        card.addView(row);
        return card;
    }

    private View modeCard() {
        LinearLayout card = card(I18n.text("m_87ed126f7bd1"),
                I18n.text("m_1c9d99d7be2c"));
        for (int i = 0; i < MODES.length; i++) {
            final int index = i;
            boolean active = mode == index;
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
            TextView text = UI.text(requireContext(), index == 0 ? I18n.text("m_be8c427c4c2c") : "Webhook",
                    14, active ? p.ink : p.body, active ? Typeface.BOLD : Typeface.NORMAL);
            LinearLayout.LayoutParams tlp = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
            tlp.leftMargin = UI.dp(UI.MD);
            row.addView(text, tlp);
            row.setClickable(true);
            row.setOnClickListener(v -> {
                capture();
                mode = index;
                render();
            });
            UI.addRow(card, row, index == 0 ? 0 : UI.SM);
        }

        if (mode == 1) {
            webhookInput = UI.input(requireContext(), "https://panel.example.com");
            webhookInput.setInputType(InputType.TYPE_TEXT_VARIATION_URI);
            webhookInput.setText(webhookUrl);
            card.addView(UI.field(requireContext(), I18n.text("m_e834781833a0"), webhookInput, UI.MD));
            TextView note = UI.muted(requireContext(),
                    I18n.text("m_eb874100aae4"));
            UI.margin(note, 0, UI.SM, 0, 0);
            card.addView(note);
        }
        return card;
    }

    private View tokenCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), "Bot Token"));
        String hint = settings.optString("bot_token_hint", "");
        TextView desc = UI.muted(requireContext(),
                I18n.text("m_4b81c912b6aa")
                        + (settings.optBoolean("bot_token_set", false) && !hint.isEmpty() ? I18n.text("m_cfe6f505c0b2") + hint : ""));
        UI.margin(desc, 0, UI.XS, 0, UI.MD);
        card.addView(desc);

        tokenInput = passwordInput(settings.optBoolean("bot_token_set", false) ? I18n.text("m_607873e79fce") : I18n.text("m_5e76895396d6"));
        tokenInput.setText(tokenDraft);
        card.addView(tokenInput);

        if (!settings.optBoolean("bot_token_set", false) && notifyBotSet) {
            TextView reuse = UI.button(requireContext(), I18n.text("m_888c2f8b0daf"), UI.BTN_SECONDARY);
            reuse.setOnClickListener(v -> reuseFromNotify());
            UI.margin(reuse, 0, UI.MD, 0, 0);
            card.addView(reuse);
        }
        return card;
    }

    private View idsCard() {
        LinearLayout card = card(I18n.text("m_fa8620c6138e"),
                I18n.text("m_90ffb63a8969"));
        idsInput = UI.input(requireContext(), I18n.text("m_05e89e31b863"));
        idsInput.setText(chatIds);
        card.addView(idsInput);
        return card;
    }

    private View endpointCard() {
        LinearLayout card = card(I18n.text("m_3d1ab62be024"),
                I18n.text("m_dcd7389c5f5e")
                        + I18n.text("m_861c1f4a7642"));
        endpointInput = UI.input(requireContext(), "https://api.telegram.org");
        endpointInput.setText(apiEndpoint);
        if (!Session.isAdmin()) {
            endpointInput.setEnabled(false);
        }
        card.addView(endpointInput);

        if (Session.isAdmin()) {
            TextView save = UI.button(requireContext(), busy ? I18n.text("m_ff509c9ba052") : I18n.text("m_459cf7f8c7e9"), UI.BTN_PRIMARY);
            save.setEnabled(!busy);
            save.setOnClickListener(v -> saveEndpoint(endpointInput.getText().toString().trim()));
            UI.margin(save, 0, UI.MD, 0, 0);
            card.addView(save);
        } else {
            TextView hint = UI.muted(requireContext(), I18n.text("m_4ce7ea1d6b95"));
            UI.margin(hint, 0, UI.SM, 0, 0);
            card.addView(hint);
        }
        return card;
    }

    private View actionsCard() {
        LinearLayout card = UI.card(requireContext());
        LinearLayout row = UI.row(requireContext());
        TextView save = UI.button(requireContext(), busy ? I18n.text("m_ff509c9ba052") : I18n.text("m_a7ec01a65f09"), UI.BTN_PRIMARY);
        save.setEnabled(!busy);
        save.setOnClickListener(v -> save());
        UI.weight(save, 1f);
        row.addView(save);
        TextView test = UI.button(requireContext(), I18n.text("m_15041725283e"), UI.BTN_SECONDARY);
        test.setOnClickListener(v -> test());
        UI.weight(test, 1f);
        UI.margin(test, UI.SM, 0, 0, 0);
        row.addView(test);
        card.addView(row);
        return card;
    }

    private View guideCard() {
        LinearLayout card = card(I18n.text("m_1a1b001776ed"), null);
        String[] steps = {
                I18n.text("m_f965f7e99876"),
                I18n.text("m_79e830fdd808"),
                I18n.text("m_5e8f0e26ee1b"),
                I18n.text("m_6bf58dc48018"),
        };
        for (int i = 0; i < steps.length; i++) {
            LinearLayout row = UI.row(requireContext());
            row.setGravity(Gravity.TOP);
            TextView index = UI.mono(requireContext(), (i + 1) + ".", Theme.p().mute);
            row.addView(index, new LinearLayout.LayoutParams(UI.dp(24), ViewGroup.LayoutParams.WRAP_CONTENT));
            row.addView(UI.body(requireContext(), steps[i]),
                    new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
            UI.addRow(card, row, i == 0 ? 0 : UI.SM);
        }

        TextView label = UI.label(requireContext(), I18n.text("m_185fee4ba4d3"));
        UI.margin(label, 0, UI.LG, 0, UI.SM);
        card.addView(label);
        StringBuilder commands = new StringBuilder();
        for (String command : COMMANDS) {
            if (commands.length() > 0) commands.append("  ");
            commands.append(command);
        }
        TextView list = UI.mono(requireContext(), commands.toString(), Theme.p().ink);
        list.setLineSpacing(UI.dp(3), 1f);
        card.addView(list);

        TextView note = UI.muted(requireContext(),
                I18n.text("m_0c09f6b0b1b0"));
        UI.margin(note, 0, UI.MD, 0, 0);
        card.addView(note);
        return card;
    }

    // ---------------------------------------------------------------- actions

    private void save() {
        capture();
        busy = true;
        banner = "";
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("enabled", enabled);
            payload.put("bot_token", tokenDraft);
            payload.put("admin_tg_ids", chatIds.trim());
            payload.put("mode", MODES[mode]);
            payload.put("webhook_url", webhookUrl.trim());
            return Api.put("/api/telegram/settings", payload);
        }, result -> {
            busy = false;
            String error = result.optString("error", "");
            banner = error.isEmpty() ? I18n.text("m_bd03c0aad701") : I18n.text("m_ca7aef8d16bd") + error;
            bannerError = !error.isEmpty();
            tokenDraft = "";
            load();
        }, failure -> {
            busy = false;
            banner = I18n.text("m_6ade7baf8737") + failure.getMessage();
            bannerError = true;
            render();
        });
    }

    private void saveEndpoint(String endpoint) {
        if (endpoint.isEmpty()) {
            banner = I18n.text("m_bcd372c0a699");
            bannerError = true;
            render();
            return;
        }
        busy = true;
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("api_endpoint", endpoint);
            return Api.put("/api/telegram/endpoint", payload);
        }, result -> {
            busy = false;
            apiEndpoint = result.optString("api_endpoint", endpoint);
            banner = I18n.text("m_f07cfdf5580f");
            bannerError = false;
            render();
        }, failure -> {
            busy = false;
            banner = I18n.text("m_6ade7baf8737") + failure.getMessage();
            bannerError = true;
            render();
        });
    }

    private void reuseFromNotify() {
        busy = true;
        banner = "";
        render();
        Api.async(() -> Api.post("/api/telegram/reuse", null), result -> {
            busy = false;
            banner = I18n.text("m_8f2a14658df1");
            bannerError = false;
            load();
        }, failure -> {
            busy = false;
            banner = I18n.text("m_57ef54e7a666") + failure.getMessage();
            bannerError = true;
            render();
        });
    }

    private void test() {
        Api.async(() -> Api.post("/api/telegram/test", null), result -> {
            banner = result.optString("message", I18n.text("m_a422e5f7f4c4"));
            bannerError = false;
            render();
        }, failure -> {
            banner = I18n.text("m_e44775371d6b") + failure.getMessage();
            bannerError = true;
            render();
        });
    }
}
