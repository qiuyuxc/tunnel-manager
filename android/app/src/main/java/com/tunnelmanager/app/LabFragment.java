package com.tunnelmanager.app;

import android.content.ClipData;
import android.content.ClipboardManager;
import android.content.Context;
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
import androidx.appcompat.app.AlertDialog;

import org.json.JSONArray;
import org.json.JSONObject;

import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Set;

/**
 * IP 优选实验室 — direct-IP scanning with optional Huawei Cloud DNS updates.
 *
 * Admin + experimental only, so the page leans on the console's own cards
 * rather than hiding anything behind role gates. While a run is in flight the
 * poll only repaints the progress card: rebuilding the whole page every second
 * would tear down the form fields the operator may still be editing.
 */
public class LabFragment extends PageFragment {

    private static final long POLL_INTERVAL = 1000L;

    private ScrollView scroll;
    private LinearLayout body;
    private LinearLayout progressSlot;
    private LinearLayout toolbarSlot;
    private TextView authorizationSummary;
    private EditText authorizationInput;
    private String authorizationDomain = "";
    private boolean viewActive;
    private final Handler ui = new Handler(Looper.getMainLooper());

    private JSONObject status = new JSONObject();
    private JSONObject dnsAvailability = new JSONObject();
    private JSONArray runs = new JSONArray();

    private String host = "";
    private String sni = "";
    private String probePath = "/";
    private String statuses = "200";
    private int timeout = 2;
    private int workers = 32;
    private int top = 10;
    private String ipTargets = "";
    private boolean schedule = false;
    private int intervalMinutes = 30;
    private boolean updateDns = false;
    private String zone = "";
    private String zoneId = "";
    private String record = "";
    private int ttl = 300;
    private String endpoint = "https://dns.myhuaweicloud.com";
    private String accessKey = "";
    private boolean hasSecretKey = false;
    private String secretDraft = "";

    private EditText hostInput;
    private EditText sniInput;
    private EditText pathInput;
    private EditText statusesInput;
    private EditText timeoutInput;
    private EditText workersInput;
    private EditText topInput;
    private EditText targetsInput;
    private EditText intervalInput;
    private EditText zoneInput;
    private EditText zoneIdInput;
    private EditText recordInput;
    private EditText ttlInput;
    private EditText endpointInput;
    private EditText accessKeyInput;
    private EditText secretInput;

    private boolean loading = true;
    private boolean busy = false;
    private boolean running = false;
    private String banner = "";
    private boolean bannerError = true;

    private final Runnable pollTask = new Runnable() {
        @Override
        public void run() {
            if (!viewActive || !alive()) return;
            loadStatus();
        }
    };

    @Override
    String route() {
        return "/lab/ip-selector";
    }

    @Override
    protected View build(@NonNull LayoutInflater inflater, @Nullable ViewGroup container) {
        viewActive = true;
        scroll = new ScrollView(requireContext());
        body = UI.column(requireContext());
        UI.pagePadding(body);
        scroll.addView(body);
        render();
        load();
        return scroll;
    }

    @Override
    public void onDestroyView() {
        capture();
        viewActive = false;
        ui.removeCallbacks(pollTask);
        body = null;
        progressSlot = null;
        toolbarSlot = null;
        super.onDestroyView();
    }

    // ------------------------------------------------------------------- data

    private void load() {
        loading = true;
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("settings", Api.get("/api/lab/ip-selector"));
            payload.put("status", Api.get("/api/lab/ip-selector/status"));
            try {
                payload.put("dns", Api.get("/api/lab/ip-selector/ownership/dns"));
            } catch (Exception failure) {
                payload.put("dns", new JSONObject().put("available", false).put("message", I18n.text("m_1f4884fcf7fe")));
            }
            return payload;
        }, payload -> {
            if (!viewActive || !alive()) return;
            loading = false;
            banner = "";
            applySettings(payload.optJSONObject("settings"));
            applyStatus(payload.optJSONObject("status"));
            dnsAvailability = payload.optJSONObject("dns");
            render();
            schedulePoll();
        }, failure -> {
            if (!viewActive || !alive()) return;
            loading = false;
            banner = I18n.text("m_01de8216e0d0") + failure.getMessage();
            bannerError = true;
            render();
        });
    }

    private void loadStatus() {
        Api.async(() -> Api.get("/api/lab/ip-selector/status"), payload -> {
            if (!viewActive || !alive()) return;
            boolean wasRunning = running;
            boolean wasAuthorized = authorized();
            int previousMaxWorkers = state().optInt("max_workers", 32);
            String previousToken = verification().optString("token");
            String previousRun = latestRun() == null ? "" : latestRun().optString("id");
            applyStatus(payload);
            String currentRun = latestRun() == null ? "" : latestRun().optString("id");
            if (wasRunning != running || !previousRun.equals(currentRun) || wasAuthorized != authorized()
                    || previousMaxWorkers != state().optInt("max_workers", 32)
                    || !previousToken.equals(verification().optString("token"))) {
                capture();
                if (wasRunning != running || !previousRun.equals(currentRun)) {
                    JSONObject latest = latestRun();
                    banner = running ? I18n.text("m_5df4f96d6d93") : latest == null ? I18n.text("m_7915a0e1de89") : latest.optString("error", "");
                    bannerError = !running && latest != null && !latest.optBoolean("success", false);
                }
                render();
            } else {
                fillProgress();
                fillToolbar();
                if (authorizationSummary != null) authorizationSummary.setText(authorizationText());
            }
            schedulePoll();
        }, failure -> {
            if (!viewActive || !alive()) return;
            if (authorizationSummary != null) authorizationSummary.setText(I18n.text("m_90cc944c3b69") + failure.getMessage());
            schedulePoll();
        });
    }

    private void schedulePoll() {
        ui.removeCallbacks(pollTask);
        if (viewActive && alive()) ui.postDelayed(pollTask, running ? POLL_INTERVAL : 5000L);
    }

    private void applySettings(JSONObject payload) {
        JSONObject data = payload == null ? new JSONObject() : payload;
        host = data.optString("host", "");
        sni = data.optString("sni", "");
        probePath = data.optString("path", "/");
        statuses = data.optString("statuses", "200");
        timeout = data.optInt("timeout", 2);
        workers = data.optInt("workers", 32);
        top = data.optInt("top", 10);
        ipTargets = data.optString("ip_targets", "");
        schedule = data.optBoolean("schedule", false);
        intervalMinutes = data.optInt("interval_minutes", 30);
        updateDns = data.optBoolean("update_dns", false);
        zone = data.optString("zone", "");
        zoneId = data.optString("zone_id", "");
        record = data.optString("record", "");
        ttl = data.optInt("ttl", 300);
        endpoint = data.optString("endpoint", "https://dns.myhuaweicloud.com");
        accessKey = data.optString("access_key", "");
        hasSecretKey = data.optBoolean("has_secret_key", false);
        secretDraft = "";
    }

    private void applyStatus(JSONObject payload) {
        status = payload == null ? new JSONObject() : payload;
        JSONObject state = status.optJSONObject("status");
        running = state != null && state.optBoolean("running", false);
        JSONArray list = status.optJSONArray("runs");
        runs = list == null ? new JSONArray() : list;
        if (authorizationDomain.isEmpty() && state != null) {
            authorizationDomain = verification().optString("domain", "");
            if (authorizationDomain.isEmpty()) authorizationDomain = state.optString("suggested_domain", "");
        }
    }

    private JSONObject state() {
        JSONObject value = status.optJSONObject("status");
        return value == null ? new JSONObject() : value;
    }

    private JSONObject verification() {
        JSONObject value = state().optJSONObject("verification");
        return value == null ? new JSONObject() : value;
    }

    private boolean authorized() {
        JSONObject value = verification();
        return value.optLong("verified_at", 0) > 0 && !Session.userId().isEmpty()
                && Session.userId().equals(value.optString("owner_id"))
                && !value.optString("domain").trim().isEmpty();
    }

    private String authorizationText() {
        JSONObject current = state();
        String text = (authorized() ? I18n.text("m_f95cc5f89b68") : I18n.text("m_e1cceb37fec6")) + " · UTC " + current.optString("budget_day", "")
                + I18n.text("m_d7e8f89b8663") + current.optInt("budget_used") + " / " + current.optInt("budget_limit", 100000)
                + " · " + current.optInt("requests_per_second", 10) + I18n.text("m_01e5d1dfa525") + current.optInt("max_workers", 32) + I18n.text("m_a756bd10d06d");
        long cooldown = current.optLong("next_allowed_at") - System.currentTimeMillis() / 1000;
        if (cooldown > 0) text += I18n.text("m_687462f06425") + cooldown + I18n.text("m_00fe8dda1fd7");
        String error = verification().optString("last_error", "");
        if (error.isEmpty()) error = current.optString("last_error", "");
        if (!error.isEmpty()) text += "\n" + error;
        return text;
    }

    private JSONObject progress() {
        JSONObject state = status.optJSONObject("status");
        JSONObject value = state == null ? null : state.optJSONObject("progress");
        return value == null ? new JSONObject() : value;
    }

    private JSONObject lastRun() {
        JSONObject state = status.optJSONObject("status");
        JSONObject value = state == null ? null : state.optJSONObject("last_run");
        return value;
    }

    private JSONObject latestRun() {
        if (runs.length() > 0) return runs.optJSONObject(0);
        return lastRun();
    }

    private String phase() {
        JSONObject state = status.optJSONObject("status");
        return state == null ? "" : state.optString("phase", "");
    }

    // -------------------------------------------------------------- rendering

    private void capture() {
        if (authorizationInput != null) authorizationDomain = authorizationInput.getText().toString().trim();
        if (hostInput != null) host = hostInput.getText().toString().trim();
        if (sniInput != null) sni = sniInput.getText().toString().trim();
        if (pathInput != null) probePath = pathInput.getText().toString().trim();
        if (statusesInput != null) statuses = statusesInput.getText().toString().trim();
        if (timeoutInput != null) timeout = intOr(timeoutInput.getText().toString(), timeout);
        if (workersInput != null) workers = intOr(workersInput.getText().toString(), workers);
        if (topInput != null) top = intOr(topInput.getText().toString(), top);
        if (targetsInput != null) ipTargets = targetsInput.getText().toString();
        if (intervalInput != null) intervalMinutes = intOr(intervalInput.getText().toString(), intervalMinutes);
        if (zoneInput != null) zone = zoneInput.getText().toString().trim();
        if (zoneIdInput != null) zoneId = zoneIdInput.getText().toString().trim();
        if (recordInput != null) record = recordInput.getText().toString().trim();
        if (ttlInput != null) ttl = intOr(ttlInput.getText().toString(), ttl);
        if (endpointInput != null) endpoint = endpointInput.getText().toString().trim();
        if (accessKeyInput != null) accessKey = accessKeyInput.getText().toString().trim();
        if (secretInput != null) secretDraft = secretInput.getText().toString();
    }

    private static int intOr(String raw, int fallback) {
        try {
            return Integer.parseInt(raw.trim());
        } catch (NumberFormatException e) {
            return fallback;
        }
    }

    private void render() {
        if (body == null || !alive()) return;
        int keepScroll = scroll.getScrollY();
        body.removeAllViews();
        progressSlot = null;

        body.addView(UI.pageTitle(requireContext(), I18n.text("m_3db29474a184")));
        TextView subtitle = UI.muted(requireContext(),
                I18n.text("m_c903232627f5"));
        UI.margin(subtitle, 0, UI.XS, 0, UI.MD);
        body.addView(subtitle);

        if (loading) {
            body.addView(UI.muted(requireContext(), I18n.text("m_4927a53bcc88")));
            return;
        }

        body.addView(toolbarSlot = UI.column(requireContext()));
        fillToolbar();
        UI.addRow(body, progressSlot = UI.column(requireContext()), UI.MD);
        fillProgress();
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(probeCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(ownershipCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(autoCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(resultsCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(missedCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        View diagnostics = diagnosticsCard();
        if (diagnostics != null) {
            body.addView(diagnostics);
            body.addView(UI.spacer(requireContext(), UI.MD));
        }
        body.addView(historyCard());

        if (!banner.isEmpty()) {
            TextView view = UI.banner(requireContext(), banner, bannerError);
            UI.margin(view, 0, UI.MD, 0, 0);
            body.addView(view);
        }
        UI.restoreScroll(scroll, keepScroll);
        enableFields(body, !busy);
    }

    private void enableFields(View view, boolean enabled) {
        if (view instanceof EditText) view.setEnabled(enabled);
        if (view instanceof ViewGroup) {
            ViewGroup group = (ViewGroup) view;
            for (int index = 0; index < group.getChildCount(); index++) enableFields(group.getChildAt(index), enabled);
        }
    }

    private void fillToolbar() {
        if (toolbarSlot == null) return;
        toolbarSlot.removeAllViews();
        toolbarSlot.addView(toolbar());
    }

    private View ownershipCard() {
        LinearLayout card = cardHead(I18n.text("m_fad1f3ce9254"), I18n.text("m_efff3af6baf6"));
        authorizationInput = UI.input(requireContext(), "example.com");
        authorizationInput.setText(authorizationDomain);
        card.addView(UI.field(requireContext(), I18n.text("m_ca1c8872d2ec"), authorizationInput));
        authorizationSummary = UI.body(requireContext(), authorizationText());
        UI.addRow(card, authorizationSummary, UI.MD);
        if (dnsAvailability != null) {
            String dnsInfo = dnsAvailability.optString("message", I18n.text("m_1a8bb3651868"));
            String accountName = dnsAvailability.optString("account_name", "");
            if (!accountName.isEmpty()) dnsInfo += I18n.text("m_db9f8be9eeb5") + accountName;
            UI.addRow(card, UI.muted(requireContext(), dnsInfo), UI.MD);
            if (dnsAvailability.optBoolean("available", false)) {
                TextView automatic = UI.button(requireContext(), I18n.text("m_6f6769158cd2"), UI.BTN_PRIMARY);
                automatic.setEnabled(!busy && !running);
                automatic.setOnClickListener(view -> {
                    schedule = false;
                    ownershipAction("/ownership/dns");
                });
                UI.addRow(card, automatic, UI.MD);
            }
        }
        TextView help = UI.button(requireContext(), I18n.text("m_cb4ba40bf823"), UI.BTN_GHOST);
        help.setOnClickListener(view -> new AlertDialog.Builder(requireContext()).setTitle(I18n.text("m_cb4ba40bf823"))
                .setMessage(I18n.text("m_3c48683ffca0"))
                .setPositiveButton(I18n.text("m_de32e20193ad"), null).show());
        UI.addRow(card, help, UI.MD);
        TextView challenge = UI.button(requireContext(), I18n.text("m_b906b98d56bf"), UI.BTN_SECONDARY);
        challenge.setEnabled(!busy && !running);
        challenge.setOnClickListener(view -> {
            capture();
            Runnable generate = () -> ownershipAction("/ownership/challenge");
            if (verification().optString("token").isEmpty()) generate.run();
            else new AlertDialog.Builder(requireContext()).setMessage(I18n.text("m_ad26c362ad25"))
                    .setNegativeButton(I18n.text("m_2cd0f3be8738"), null).setPositiveButton(I18n.text("m_7c9691192f1b"), (dialog, which) -> generate.run()).show();
        });
        UI.addRow(card, challenge, UI.MD);
        JSONObject verified = verification();
        if (!verified.optString("token").isEmpty()) {
            if (authorized()) {
                UI.addRow(card, UI.muted(requireContext(), I18n.text("m_00097d89c4fe") + verified.optString("domain")), UI.MD);
            } else for (String field : new String[]{"record_name", "token"}) {
                String text = verified.optString(field);
                TextView recordText = UI.body(requireContext(), ("record_name".equals(field) ? I18n.text("m_fa1ed10b3059") : I18n.text("m_e74b93aa2288")) + text);
                recordText.setTextIsSelectable(true);
                UI.addRow(card, recordText, UI.MD);
                TextView copy = UI.button(requireContext(), "record_name".equals(field) ? I18n.text("m_24b0fd0474b8") : I18n.text("m_82554e2b0462"), UI.BTN_GHOST);
                copy.setOnClickListener(view -> {
                    ClipboardManager clipboard = (ClipboardManager) requireContext().getSystemService(Context.CLIPBOARD_SERVICE);
                    if (clipboard != null) clipboard.setPrimaryClip(ClipData.newPlainText("DNS TXT", text));
                });
                card.addView(copy);
            }
            TextView verify = UI.button(requireContext(), I18n.text("m_45ef5bfe7b6f"), UI.BTN_PRIMARY);
            verify.setEnabled(!busy && !running);
            verify.setOnClickListener(view -> ownershipAction("/ownership/verify"));
            UI.addRow(card, verify, UI.MD);
        }
        return card;
    }

    /** Rebuilds only the progress block, for the once-a-second poll. */
    private void fillProgress() {
        if (progressSlot == null) return;
        progressSlot.removeAllViews();
        JSONObject value = progress();
        int total = value.optInt("total", 0);
        if (!running && total <= 0) return;

        LinearLayout card = UI.card(requireContext());
        int percent = Math.max(0, Math.min(100, value.optInt("percent", 0)));

        LinearLayout head = UI.row(requireContext());
        head.addView(UI.strong(requireContext(), phaseText()),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        TextView number = UI.mono(requireContext(), percent + "%", Theme.p().ink);
        number.setTypeface(Typeface.MONOSPACE, Typeface.BOLD);
        head.addView(number);
        card.addView(head);

        LinearLayout track = UI.row(requireContext());
        track.setBackground(UI.rounded(Theme.p().canvasSoft2, UI.RADIUS_PILL));
        View filled = new View(requireContext());
        filled.setBackground(UI.rounded(Theme.p().btnPrimaryBg, UI.RADIUS_PILL));
        LinearLayout.LayoutParams fillLp = new LinearLayout.LayoutParams(0, UI.dp(8), percent);
        track.addView(filled, fillLp);
        View rest = new View(requireContext());
        track.addView(rest, new LinearLayout.LayoutParams(0, UI.dp(8), Math.max(1, 100 - percent)));
        UI.margin(track, 0, UI.MD, 0, 0);
        card.addView(track);

        LinearLayout meta = UI.row(requireContext());
        meta.addView(UI.muted(requireContext(),
                        I18n.text("m_93ef77874ea0") + value.optInt("scanned", 0) + " / " + total),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        meta.addView(UI.muted(requireContext(), I18n.text("m_2b0dba4e5ce7") + value.optInt("matched", 0)));
        UI.margin(meta, 0, UI.SM, 0, 0);
        card.addView(meta);

        UI.addRow(progressSlot, card, 0);
    }

    private View toolbar() {
        LinearLayout card = UI.card(requireContext());
        LinearLayout head = UI.row(requireContext());
        TextView pill = running ? UI.tag(requireContext(), I18n.text("m_5df4f96d6d93"), false) : UI.tag(requireContext(), I18n.text("m_dae661d17c2d"), true);
        if (running) {
            pill.setTextColor(Theme.p().statusDegradedText);
            pill.setBackground(UI.roundedStroke(Theme.p().statusDegradedBg, UI.RADIUS_PILL,
                    Theme.p().statusDegradedBorder, 1));
        }
        LinearLayout statusBox = UI.row(requireContext());
        statusBox.setGravity(Gravity.START);
        statusBox.addView(pill);
        head.addView(statusBox, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        TextView refresh = UI.button(requireContext(), I18n.text("m_802a407c7743"), UI.BTN_GHOST);
        refresh.setEnabled(!busy);
        refresh.setOnClickListener(v -> loadStatus());
        head.addView(refresh);
        card.addView(head);

        TextView run = UI.button(requireContext(), running ? I18n.text("m_14d9f2d830ad") : I18n.text("m_126a24ed51ca"), UI.BTN_PRIMARY);
        run.setEnabled(!running && !busy && authorized()
                && state().optLong("next_allowed_at") <= System.currentTimeMillis() / 1000
                && state().optInt("budget_used") < state().optInt("budget_limit", 100000));
        run.setOnClickListener(v -> save(this::startRun));
        UI.margin(run, 0, UI.MD, 0, 0);
        card.addView(run);
        if (running || schedule) {
            TextView stop = UI.button(requireContext(), I18n.text("m_db53b055cad4"), UI.BTN_SECONDARY);
            stop.setEnabled(!busy);
            stop.setOnClickListener(view -> ownershipAction("/stop"));
            UI.addRow(card, stop, UI.MD);
        }
        return card;
    }

    private String phaseText() {
        String phase = phase();
        if ("preparing".equals(phase)) return I18n.text("m_81c4d554eaac");
        if ("scanning".equals(phase)) return I18n.text("m_37425cd304f1");
        if ("updating_dns".equals(phase)) return I18n.text("m_5b74adf3e039");
        if ("completed".equals(phase)) return I18n.text("m_e9a880e59521");
        if ("failed".equals(phase)) return I18n.text("m_3ef426846721");
        return I18n.text("m_4f021f59121c");
    }

    private LinearLayout cardHead(String title, String desc) {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), title));
        if (desc != null && !desc.isEmpty()) {
            TextView hint = UI.muted(requireContext(), desc);
            UI.margin(hint, 0, UI.XS, 0, UI.MD);
            card.addView(hint);
        }
        return card;
    }

    private EditText numberInput(String hint, int value) {
        EditText input = UI.input(requireContext(), hint);
        input.setInputType(InputType.TYPE_CLASS_NUMBER);
        input.setText(String.valueOf(value));
        return input;
    }

    /** Two fields side by side; the console's grid collapses to one column here. */
    private View pair(View left, View right) {
        LinearLayout row = UI.row(requireContext());
        row.setGravity(Gravity.TOP);
        left.setLayoutParams(new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        row.addView(left);
        UI.margin(right, UI.MD, 0, 0, 0);
        right.setLayoutParams(new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        row.addView(right);
        return row;
    }

    private View probeCard() {
        LinearLayout card = cardHead(I18n.text("m_d9cb32d240c7"),
                I18n.text("m_2d028c40657b"));

        hostInput = UI.input(requireContext(), "cdn.example.com");
        hostInput.setText(host);
        card.addView(UI.field(requireContext(), "Host（HTTP Host Header）", hostInput));

        sniInput = UI.input(requireContext(), I18n.text("m_263273daac57"));
        sniInput.setText(sni);
        card.addView(UI.field(requireContext(), "SNI（TLS Server Name）", sniInput, UI.MD));

        pathInput = UI.input(requireContext(), "/healthz");
        pathInput.setText(probePath);
        card.addView(UI.field(requireContext(), I18n.text("m_fc12218cc64a"), pathInput, UI.MD));

        statusesInput = UI.input(requireContext(), "200,204");
        statusesInput.setText(statuses);
        card.addView(UI.field(requireContext(), I18n.text("m_f881100e7e8f"), statusesInput, UI.MD));

        timeoutInput = numberInput("2", timeout);
        workersInput = numberInput("32", workers);
        UI.addRow(card, pair(
                UI.field(requireContext(), I18n.text("m_e814e078437b"), timeoutInput),
                UI.field(requireContext(), I18n.text("m_08810ec6a754") + state().optInt("max_workers", 32) + "）", workersInput)), UI.MD);

        topInput = numberInput("10", top);
        card.addView(UI.field(requireContext(), I18n.text("m_7ae5ac99ec7d"), topInput, UI.MD));

        targetsInput = UI.textArea(requireContext(),
                "1.2.3.4\n1.2.4.0/24\n1.2.5.10-1.2.5.20");
        targetsInput.setMinLines(5);
        // Without a cap the field grows to fit every target and pushes the rest
        // of the page off-screen; eight lines scrolls instead.
        targetsInput.setMaxLines(8);
        targetsInput.setText(ipTargets);
        card.addView(UI.field(requireContext(),
                I18n.text("m_273f5800a334"), targetsInput, UI.MD));
        return card;
    }

    private View autoCard() {
        LinearLayout card = cardHead(I18n.text("m_951e770ec529"),
                I18n.text("m_3490d11e4c6b"));

        card.addView(toggleRow(I18n.text("m_978cb333292d"), schedule, next -> {
            capture();
            if (next && !authorized()) {
                banner = I18n.text("m_f4f840a282f3");
                bannerError = true;
                render();
                return;
            }
            schedule = next;
            render();
        }));

        if (schedule) {
            intervalInput = numberInput("30", intervalMinutes);
            card.addView(UI.field(requireContext(), I18n.text("m_63f674fb44eb"), intervalInput, UI.MD));
        }

        UI.addRow(card, toggleRow(I18n.text("m_7ce4f8dc5c94"), updateDns, next -> {
            capture();
            updateDns = next;
            render();
        }), UI.MD);

        if (updateDns) {
            zoneInput = UI.input(requireContext(), "example.com");
            zoneInput.setText(zone);
            card.addView(UI.field(requireContext(), I18n.text("m_dd7848177924"), zoneInput, UI.MD));

            zoneIdInput = UI.input(requireContext(), I18n.text("m_8a3962db8e0b"));
            zoneIdInput.setText(zoneId);
            card.addView(UI.field(requireContext(), I18n.text("m_61aa3db2f187"), zoneIdInput, UI.MD));

            recordInput = UI.input(requireContext(), "cdn.example.com");
            recordInput.setText(record);
            card.addView(UI.field(requireContext(), I18n.text("m_1ff08664dd50"), recordInput, UI.MD));

            ttlInput = numberInput("300", ttl);
            card.addView(UI.field(requireContext(), "TTL", ttlInput, UI.MD));

            endpointInput = UI.input(requireContext(), "https://dns.myhuaweicloud.com");
            endpointInput.setInputType(InputType.TYPE_TEXT_VARIATION_URI);
            endpointInput.setText(endpoint);
            card.addView(UI.field(requireContext(), "API Endpoint", endpointInput, UI.MD));

            accessKeyInput = UI.input(requireContext(), "Access Key");
            accessKeyInput.setText(accessKey);
            card.addView(UI.field(requireContext(), "Access Key", accessKeyInput, UI.MD));

            secretInput = passwordInput(hasSecretKey ? I18n.text("m_0eeb82a0c096") : I18n.text("m_2fa84bb7ab9b"));
            secretInput.setText(secretDraft);
            LinearLayout secretField = UI.field(requireContext(), "Secret Key", secretInput, UI.MD);
            TextView state = UI.tag(requireContext(), hasSecretKey ? I18n.text("m_248a0bb2c9db") : I18n.text("m_83fe22d7f5b2"), hasSecretKey);
            UI.margin(state, 0, UI.SM, 0, 0);
            LinearLayout.LayoutParams slp = new LinearLayout.LayoutParams(
                    ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);
            state.setLayoutParams(slp);
            secretField.addView(state);
            card.addView(secretField);
        }

        TextView save = UI.button(requireContext(), busy ? I18n.text("m_ff509c9ba052") : I18n.text("m_6e584e3d5ce6"), UI.BTN_PRIMARY);
        save.setEnabled(!busy && !running);
        save.setOnClickListener(v -> save());
        UI.margin(save, 0, UI.LG, 0, 0);
        card.addView(save);
        return card;
    }

    private EditText passwordInput(String hint) {
        EditText input = UI.input(requireContext(), hint);
        input.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_PASSWORD);
        return input;
    }

    private View toggleRow(String label, boolean on, ToggleListener listener) {
        LinearLayout row = UI.row(requireContext());
        UI.Toggle toggle = new UI.Toggle(requireContext(), on);
        toggle.setEnabled(!busy && !running);
        toggle.setOnClickListener(v -> listener.onChange(!toggle.isOn()));
        row.addView(toggle, new LinearLayout.LayoutParams(UI.dp(34), UI.dp(20)));
        TextView text = UI.body(requireContext(), label);
        UI.margin(text, UI.MD, 0, 0, 0);
        row.addView(text);
        return row;
    }

    private interface ToggleListener {
        void onChange(boolean next);
    }

    // ------------------------------------------------------------- results

    private View resultsCard() {
        JSONObject latest = latestRun();
        String summary = I18n.text("m_8854c2f7d83b");
        if (latest != null) {
            summary = formatTime(latest.optLong("started_at", 0))
                    + I18n.text("m_6427855ccbd2") + latest.optInt("scanned", 0) + I18n.text("m_94ed3ac99f3c") + latest.optInt("matched", 0) + I18n.text("m_7aaf96ad52e0");
            JSONArray selected = latest.optJSONArray("selected_ips");
            if (selected != null && selected.length() > 0) {
                summary += I18n.text("m_1d3f49b1b16c") + selected.length() + I18n.text("m_7aaf96ad52e0");
            }
        }
        LinearLayout card = cardHead(I18n.text("m_8f0ef3b2b86d"), summary);

        JSONArray results = latest == null ? null : latest.optJSONArray("results");
        if (results == null || results.length() == 0) {
            card.addView(emptyState(latest == null ? I18n.text("m_a1b1a1640ed1") : I18n.text("m_3318ecab73f4")));
            return card;
        }
        for (int i = 0; i < results.length(); i++) {
            JSONObject item = results.optJSONObject(i);
            if (item == null) continue;
            UI.addRow(card, resultRow(i + 1, item), i == 0 ? 0 : UI.SM);
        }
        return card;
    }

    private View resultRow(int rank, JSONObject item) {
        LinearLayout row = UI.row(requireContext());
        row.setPadding(0, UI.dp(UI.SM), 0, UI.dp(UI.SM));
        TextView index = UI.mono(requireContext(), String.valueOf(rank), Theme.p().mute);
        row.addView(index, new LinearLayout.LayoutParams(UI.dp(28), ViewGroup.LayoutParams.WRAP_CONTENT));
        row.addView(UI.mono(requireContext(), item.optString("ip"), Theme.p().ink),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));

        StringBuilder meta = new StringBuilder();
        int statusCode = item.optInt("status", 0);
        if (statusCode > 0) meta.append(statusCode).append(" · ");
        meta.append(item.optLong("latency_ms", 0)).append(" ms");
        row.addView(UI.mono(requireContext(), meta.toString(), Theme.p().body));
        return row;
    }

    private View diagnosticsCard() {
        JSONObject latest = latestRun();
        JSONArray diagnostics = latest == null ? null : latest.optJSONArray("diagnostics");
        if (diagnostics == null || diagnostics.length() == 0) return null;
        LinearLayout card = cardHead(I18n.text("m_7148c33da141"), I18n.text("m_fafb6ea98d14") + diagnostics.length()
                + " / " + latest.optInt("rejected", diagnostics.length()) + I18n.text("m_5954340a1d5b"));
        JSONObject probe = latest.optJSONObject("probe");
        String statuses = probe == null ? "" : probe.optString("statuses");
        if (probe != null) card.addView(UI.muted(requireContext(), I18n.text("m_0b33b09c41d2") + probe.optString("host")
                + probe.optString("path") + " · SNI " + probe.optString("sni") + I18n.text("m_8c59d98a4b33") + statuses));
        for (int index = 0; index < diagnostics.length(); index++) {
            JSONObject result = diagnostics.optJSONObject(index);
            if (result == null) continue;
            LinearLayout row = UI.column(requireContext());
            row.addView(UI.mono(requireContext(), result.optString("ip") + " · " + result.optLong("latency_ms") + " ms", Theme.p().ink));
            String problem = result.optString("error");
            String detail = !problem.isEmpty() ? I18n.text("m_2f43232ebb77") + problem
                    : "HTTP " + result.optInt("status") + I18n.text("m_54c086747683") + statuses + I18n.text("m_ce59ce17163a");
            row.addView(UI.muted(requireContext(), detail));
            UI.addRow(card, row, UI.SM);
        }
        return card;
    }

    private View missedCard() {
        List<JSONObject> missed = missedSegments();
        JSONObject latest = latestRun();
        int total = 0;
        if (latest != null) {
            JSONArray segments = latest.optJSONArray("segments");
            total = segments == null ? 0 : segments.length();
        }
        String summary = total == 0
                ? I18n.text("m_2fd51e19890c")
                : I18n.text("m_2428420e407f") + total + I18n.text("m_087c52b5de51") + missed.size() + I18n.text("m_0ca2d660e912")
                        + missedTotal(missed) + I18n.text("m_6ac17b291511");

        LinearLayout card = cardHead(I18n.text("m_f3cb5e38d5d7"), summary);

        if (missed.isEmpty()) {
            card.addView(emptyState(I18n.text("m_46dfefce779d")));
            return card;
        }

        for (int i = 0; i < missed.size(); i++) {
            JSONObject item = missed.get(i);
            LinearLayout row = UI.column(requireContext());
            row.setPadding(0, UI.dp(UI.SM), 0, UI.dp(UI.SM));
            LinearLayout top = UI.row(requireContext());
            top.addView(UI.mono(requireContext(), item.optString("target"), Theme.p().ink),
                    new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
            top.addView(UI.tag(requireContext(), I18n.text("m_822e6ca6c58a"), false));
            row.addView(top);
            TextView meta = UI.muted(requireContext(),
                    I18n.text("m_abcce5c1437a") + item.optInt("scanned", 0) + I18n.text("m_6015819aab22") + item.optInt("matched", 0));
            UI.margin(meta, 0, UI.XS, 0, 0);
            row.addView(meta);
            UI.addRow(card, row, i == 0 ? 0 : UI.SM);
        }

        LinearLayout actions = UI.row(requireContext());
        TextView copy = UI.button(requireContext(), I18n.text("m_72a7b6671168"), UI.BTN_SECONDARY);
        copy.setOnClickListener(v -> copyMissed(missed));
        UI.weight(copy, 1f);
        actions.addView(copy);
        TextView drop = UI.button(requireContext(), I18n.text("m_5aefa3f30318"), UI.BTN_DANGER);
        drop.setEnabled(!running && !busy);
        drop.setOnClickListener(v -> confirmDrop(missed));
        UI.weight(drop, 1f);
        UI.margin(drop, UI.SM, 0, 0, 0);
        actions.addView(drop);
        UI.addRow(card, actions, UI.MD);
        return card;
    }

    private View emptyState(String message) {
        TextView view = UI.muted(requireContext(), message);
        view.setGravity(Gravity.CENTER);
        view.setPadding(0, UI.dp(UI.LG), 0, UI.dp(UI.LG));
        return view;
    }

    private List<JSONObject> missedSegments() {
        List<JSONObject> out = new ArrayList<>();
        JSONObject latest = latestRun();
        if (latest == null) return out;
        JSONArray segments = latest.optJSONArray("segments");
        if (segments == null) return out;
        for (int i = 0; i < segments.length(); i++) {
            JSONObject item = segments.optJSONObject(i);
            if (item != null && item.optInt("matched", 0) == 0) out.add(item);
        }
        return out;
    }

    private int missedTotal(List<JSONObject> missed) {
        int total = 0;
        for (JSONObject item : missed) total += item.optInt("scanned", 0);
        return total;
    }

    private View historyCard() {
        LinearLayout card = cardHead(I18n.text("m_02451fd58cd3"), I18n.text("m_03bba0057f3d"));
        if (runs.length() == 0) {
            card.addView(emptyState(I18n.text("m_69c0d2ee422f")));
            return card;
        }
        for (int i = 0; i < runs.length(); i++) {
            JSONObject item = runs.optJSONObject(i);
            if (item == null) continue;
            LinearLayout row = UI.column(requireContext());
            row.setPadding(0, UI.dp(UI.SM), 0, UI.dp(UI.SM));

            LinearLayout top = UI.row(requireContext());
            top.addView(UI.body(requireContext(), formatTime(item.optLong("started_at", 0))),
                    new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
            boolean success = item.optBoolean("success", false);
            top.addView(UI.tag(requireContext(), success ? I18n.text("m_053461ce86d2") : I18n.text("m_28384d7afd2e"), success));
            row.addView(top);

            TextView meta = UI.muted(requireContext(),
                    I18n.text("m_221f3304a691") + item.optInt("scanned", 0) + I18n.text("m_3b0c274bd233") + item.optInt("matched", 0)
                            + " · DNS " + (item.optBoolean("dns_updated", false) ? I18n.text("m_434203a723b4") : I18n.text("m_bb44f79f1933")));
            UI.margin(meta, 0, UI.XS, 0, 0);
            row.addView(meta);

            String error = item.optString("error", "");
            if (!error.isEmpty()) {
                TextView errorView = UI.text(requireContext(), error, 13, Theme.p().error, Typeface.NORMAL);
                UI.margin(errorView, 0, UI.XS, 0, 0);
                row.addView(errorView);
            }
            UI.addRow(card, row, i == 0 ? 0 : UI.SM);
        }
        return card;
    }

    private static String formatTime(long seconds) {
        if (seconds <= 0) return "—";
        java.text.SimpleDateFormat format = new java.text.SimpleDateFormat("MM-dd HH:mm:ss",
                I18n.locale());
        return format.format(new java.util.Date(seconds * 1000L));
    }

    // ---------------------------------------------------------------- actions

    private void startRun() {
        capture();
        busy = true;
        banner = "";
        render();
        Api.async(() -> Api.post("/api/lab/ip-selector/run", new JSONObject()), result -> {
            if (!viewActive || !alive()) return;
            busy = false;
            running = true;
            banner = I18n.text("m_7f935d40a91e");
            bannerError = false;
            render();
            ui.removeCallbacks(pollTask);
            ui.postDelayed(pollTask, POLL_INTERVAL);
        }, failure -> {
            if (!viewActive || !alive()) return;
            busy = false;
            banner = I18n.text("m_a998931e9cab") + failure.getMessage();
            bannerError = true;
            render();
        });
    }

    private void save() {
        save(null);
    }

    private void save(Runnable afterSave) {
        if (busy || running) return;
        capture();
        busy = true;
        banner = "";
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("host", host);
            payload.put("sni", sni);
            payload.put("path", probePath);
            payload.put("statuses", statuses);
            payload.put("timeout", timeout);
            payload.put("workers", workers);
            payload.put("top", top);
            payload.put("ip_targets", ipTargets);
            payload.put("schedule", schedule);
            payload.put("interval_minutes", intervalMinutes);
            payload.put("update_dns", updateDns);
            payload.put("zone", zone);
            payload.put("zone_id", zoneId);
            payload.put("record", record);
            payload.put("ttl", ttl);
            payload.put("endpoint", endpoint);
            payload.put("access_key", accessKey);
            if (!secretDraft.isEmpty()) payload.put("secret_key", secretDraft);
            return Api.put("/api/lab/ip-selector", payload);
        }, result -> {
            if (!viewActive || !alive()) return;
            busy = false;
            banner = I18n.text("m_2890332be041");
            bannerError = false;
            applySettings(result);
            render();
            if (afterSave != null) afterSave.run();
            else loadStatus();
        }, failure -> {
            if (!viewActive || !alive()) return;
            busy = false;
            banner = I18n.text("m_6ade7baf8737") + failure.getMessage();
            bannerError = true;
            render();
        });
    }

    private void ownershipAction(String action) {
        if (busy) return;
        capture();
        String domain = authorizationDomain;
        busy = true;
        banner = "";
        render();
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            if ("/ownership/challenge".equals(action) || "/ownership/dns".equals(action)) payload.put("domain", domain);
            JSONObject operation = Api.post("/api/lab/ip-selector" + action, payload);
            return Api.get("/api/lab/ip-selector/status").put("operation", operation);
        }, result -> {
            if (!viewActive || !alive()) return;
            busy = false;
            applyStatus(result);
            if (!"/ownership/verify".equals(action)) schedule = false;
            banner = "/stop".equals(action) ? I18n.text("m_c1a4539fd21f") : "/ownership/challenge".equals(action)
                    ? I18n.text("m_d6390eeb1913") : I18n.text("m_31a1025984de");
            if ("/ownership/dns".equals(action)) {
                JSONObject operation = result.optJSONObject("operation");
                banner = operation == null ? I18n.text("m_5d43722902f7") : operation.optString("message");
            }
            bannerError = false;
            render();
            schedulePoll();
        }, failure -> {
            if (!viewActive || !alive()) return;
            busy = false;
            if ("/ownership/verify".equals(action)) schedule = false;
            banner = I18n.text("m_37cdfdb143a7") + failure.getMessage();
            bannerError = true;
            render();
            loadStatus();
        });
    }

    private void copyMissed(List<JSONObject> missed) {
        StringBuilder text = new StringBuilder();
        for (JSONObject item : missed) {
            if (text.length() > 0) text.append('\n');
            text.append(item.optString("target"));
        }
        ClipboardManager clipboard = (ClipboardManager) requireContext().getSystemService(Context.CLIPBOARD_SERVICE);
        if (clipboard == null) return;
        clipboard.setPrimaryClip(ClipData.newPlainText("missed-segments", text.toString()));
        banner = I18n.text("m_b364d30acd30");
        bannerError = false;
        render();
    }

    private void confirmDrop(List<JSONObject> missed) {
        Set<String> drop = new LinkedHashSet<>();
        for (JSONObject item : missed) drop.add(item.optString("target"));

        List<String> current = new ArrayList<>();
        for (String target : ipTargets.split("[,\\s]+")) {
            if (!target.isEmpty()) current.add(target);
        }
        List<String> kept = new ArrayList<>();
        for (String target : current) {
            if (!drop.contains(target)) kept.add(target);
        }
        int removed = current.size() - kept.size();
        if (removed == 0) {
            banner = I18n.text("m_3d3ce77e1cc8");
            bannerError = true;
            render();
            return;
        }
        capture();
        Modal.of(requireContext(), I18n.text("m_2410beea7b59"))
                .message(I18n.text("m_25dc865b15b9") + missed.size() + I18n.text("m_d3eccb0aa84e") + missedTotal(missed)
                        + I18n.text("m_b7d9a16af903"))
                .cancel(I18n.text("m_2cd0f3be8738"))
                .confirm(I18n.text("m_7aef3d7d5120"), true, () -> {
                    StringBuilder text = new StringBuilder();
                    for (String target : kept) {
                        if (text.length() > 0) text.append('\n');
                        text.append(target);
                    }
                    ipTargets = text.toString();
                    save();
                })
                .show();
    }
}
