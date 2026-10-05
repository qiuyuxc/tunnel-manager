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
                payload.put("dns", new JSONObject().put("available", false).put("message", "暂时无法检查已绑定账户，可手动添加 TXT。"));
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
            banner = "加载失败：" + failure.getMessage();
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
                    banner = running ? "任务运行中" : latest == null ? "任务已停止" : latest.optString("error", "");
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
            if (authorizationSummary != null) authorizationSummary.setText("状态更新失败：" + failure.getMessage());
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
        String text = (authorized() ? "域名已授权" : "请先验证域名") + " · UTC " + current.optString("budget_day", "")
                + "\n已预留 " + current.optInt("budget_used") + " / " + current.optInt("budget_limit", 100000)
                + " · " + current.optInt("requests_per_second", 10) + " 请求/秒 · 最多 " + current.optInt("max_workers", 32) + " 并发";
        long cooldown = current.optLong("next_allowed_at") - System.currentTimeMillis() / 1000;
        if (cooldown > 0) text += "\n启动冷却剩余 " + cooldown + " 秒";
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

        body.addView(UI.pageTitle(requireContext(), "IP 优选实验室"));
        TextView subtitle = UI.muted(requireContext(),
                "HTTP 探测、延迟排序与 DNS 更新");
        UI.margin(subtitle, 0, UI.XS, 0, UI.MD);
        body.addView(subtitle);

        if (loading) {
            body.addView(UI.muted(requireContext(), "加载中…"));
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
        LinearLayout card = cardHead("DNS TXT 域名验证", "与探测 Host/SNI 独立。验证后请保留 DNS 中的 TXT。");
        authorizationInput = UI.input(requireContext(), "example.com");
        authorizationInput.setText(authorizationDomain);
        card.addView(UI.field(requireContext(), "验证域名", authorizationInput));
        authorizationSummary = UI.body(requireContext(), authorizationText());
        UI.addRow(card, authorizationSummary, UI.MD);
        if (dnsAvailability != null) {
            String dnsInfo = dnsAvailability.optString("message", "未绑定 DNS 账户，请手动添加 TXT。");
            String accountName = dnsAvailability.optString("account_name", "");
            if (!accountName.isEmpty()) dnsInfo += " 当前账户：" + accountName;
            UI.addRow(card, UI.muted(requireContext(), dnsInfo), UI.MD);
            if (dnsAvailability.optBoolean("available", false)) {
                TextView automatic = UI.button(requireContext(), "一键填写并验证", UI.BTN_PRIMARY);
                automatic.setEnabled(!busy && !running);
                automatic.setOnClickListener(view -> {
                    schedule = false;
                    ownershipAction("/ownership/dns");
                });
                UI.addRow(card, automatic, UI.MD);
            }
        }
        TextView help = UI.button(requireContext(), "使用说明", UI.BTN_GHOST);
        help.setOnClickListener(view -> new AlertDialog.Builder(requireContext()).setTitle("使用说明")
                .setMessage("预算由管理员设置，按候选 IP 预留，取消不退，UTC 次日重置；修改从下一轮生效，超出新并发上限时按上限运行。这不是云平台用量统计。\n\n一键只添加本次 TXT，不覆盖其他记录，也不启动扫描。验证成功后收起记录值；运行前仍会复查，请保留 DNS 中的 TXT。\n\n只探测自建或已获授权的 Host/SNI。TXT 验证不代表探针免计费，请向托管平台确认。")
                .setPositiveButton("知道了", null).show());
        UI.addRow(card, help, UI.MD);
        TextView challenge = UI.button(requireContext(), "生成 TXT", UI.BTN_SECONDARY);
        challenge.setEnabled(!busy && !running);
        challenge.setOnClickListener(view -> {
            capture();
            Runnable generate = () -> ownershipAction("/ownership/challenge");
            if (verification().optString("token").isEmpty()) generate.run();
            else new AlertDialog.Builder(requireContext()).setMessage("生成新记录会撤销旧授权并关闭定时任务，是否继续？")
                    .setNegativeButton("取消", null).setPositiveButton("继续", (dialog, which) -> generate.run()).show();
        });
        UI.addRow(card, challenge, UI.MD);
        JSONObject verified = verification();
        if (!verified.optString("token").isEmpty()) {
            if (authorized()) {
                UI.addRow(card, UI.muted(requireContext(), "已验证：" + verified.optString("domain")), UI.MD);
            } else for (String field : new String[]{"record_name", "token"}) {
                String text = verified.optString(field);
                TextView recordText = UI.body(requireContext(), ("record_name".equals(field) ? "TXT 完整记录名\n" : "TXT 记录值\n") + text);
                recordText.setTextIsSelectable(true);
                UI.addRow(card, recordText, UI.MD);
                TextView copy = UI.button(requireContext(), "record_name".equals(field) ? "复制记录名" : "复制记录值", UI.BTN_GHOST);
                copy.setOnClickListener(view -> {
                    ClipboardManager clipboard = (ClipboardManager) requireContext().getSystemService(Context.CLIPBOARD_SERVICE);
                    if (clipboard != null) clipboard.setPrimaryClip(ClipData.newPlainText("DNS TXT", text));
                });
                card.addView(copy);
            }
            TextView verify = UI.button(requireContext(), "检查 TXT 记录", UI.BTN_PRIMARY);
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
                        "已扫描 " + value.optInt("scanned", 0) + " / " + total),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        meta.addView(UI.muted(requireContext(), "匹配 " + value.optInt("matched", 0)));
        UI.margin(meta, 0, UI.SM, 0, 0);
        card.addView(meta);

        UI.addRow(progressSlot, card, 0);
    }

    private View toolbar() {
        LinearLayout card = UI.card(requireContext());
        LinearLayout head = UI.row(requireContext());
        TextView pill = running ? UI.tag(requireContext(), "任务运行中", false) : UI.tag(requireContext(), "空闲", true);
        if (running) {
            pill.setTextColor(Theme.p().statusDegradedText);
            pill.setBackground(UI.roundedStroke(Theme.p().statusDegradedBg, UI.RADIUS_PILL,
                    Theme.p().statusDegradedBorder, 1));
        }
        LinearLayout statusBox = UI.row(requireContext());
        statusBox.setGravity(Gravity.START);
        statusBox.addView(pill);
        head.addView(statusBox, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        TextView refresh = UI.button(requireContext(), "刷新状态", UI.BTN_GHOST);
        refresh.setEnabled(!busy);
        refresh.setOnClickListener(v -> loadStatus());
        head.addView(refresh);
        card.addView(head);

        TextView run = UI.button(requireContext(), running ? "运行中…" : "保存并执行", UI.BTN_PRIMARY);
        run.setEnabled(!running && !busy && authorized()
                && state().optLong("next_allowed_at") <= System.currentTimeMillis() / 1000
                && state().optInt("budget_used") < state().optInt("budget_limit", 100000));
        run.setOnClickListener(v -> save(this::startRun));
        UI.margin(run, 0, UI.MD, 0, 0);
        card.addView(run);
        if (running || schedule) {
            TextView stop = UI.button(requireContext(), "停止并关闭定时", UI.BTN_SECONDARY);
            stop.setEnabled(!busy);
            stop.setOnClickListener(view -> ownershipAction("/stop"));
            UI.addRow(card, stop, UI.MD);
        }
        return card;
    }

    private String phaseText() {
        String phase = phase();
        if ("preparing".equals(phase)) return "准备扫描…";
        if ("scanning".equals(phase)) return "正在探测 IP…";
        if ("updating_dns".equals(phase)) return "扫描完成，正在更新华为云 DNS…";
        if ("completed".equals(phase)) return "任务已完成";
        if ("failed".equals(phase)) return "任务未完成，请检查错误信息";
        return "等待执行";
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
        LinearLayout card = cardHead("探测配置",
                "Host 与 SNI 可指向托管在 Cloudflare 或其他平台的域名；SNI 留空时使用 Host。");

        hostInput = UI.input(requireContext(), "cdn.example.com");
        hostInput.setText(host);
        card.addView(UI.field(requireContext(), "Host（HTTP Host Header）", hostInput));

        sniInput = UI.input(requireContext(), "留空使用 Host");
        sniInput.setText(sni);
        card.addView(UI.field(requireContext(), "SNI（TLS Server Name）", sniInput, UI.MD));

        pathInput = UI.input(requireContext(), "/healthz");
        pathInput.setText(probePath);
        card.addView(UI.field(requireContext(), "探测路径", pathInput, UI.MD));

        statusesInput = UI.input(requireContext(), "200,204");
        statusesInput.setText(statuses);
        card.addView(UI.field(requireContext(), "接受状态码", statusesInput, UI.MD));

        timeoutInput = numberInput("2", timeout);
        workersInput = numberInput("32", workers);
        UI.addRow(card, pair(
                UI.field(requireContext(), "超时（秒）", timeoutInput),
                UI.field(requireContext(), "并发数（1 至 " + state().optInt("max_workers", 32) + "）", workersInput)), UI.MD);

        topInput = numberInput("10", top);
        card.addView(UI.field(requireContext(), "保留 Top N", topInput, UI.MD));

        targetsInput = UI.textArea(requireContext(),
                "1.2.3.4\n1.2.4.0/24\n1.2.5.10-1.2.5.20");
        targetsInput.setMinLines(5);
        // Without a cap the field grows to fit every target and pushes the rest
        // of the page off-screen; eight lines scrolls instead.
        targetsInput.setMaxLines(8);
        targetsInput.setText(ipTargets);
        card.addView(UI.field(requireContext(),
                "IP 段（支持单 IP、CIDR、范围，逗号或换行分隔）", targetsInput, UI.MD));
        return card;
    }

    private View autoCard() {
        LinearLayout card = cardHead("自动化与华为云 DNS",
                "密钥使用应用加密密钥加密保存；关闭「自动更新 DNS」时仅扫描并展示结果。");

        card.addView(toggleRow("定时执行", schedule, next -> {
            capture();
            if (next && !authorized()) {
                banner = "请先完成当前管理员的域名验证";
                bannerError = true;
                render();
                return;
            }
            schedule = next;
            render();
        }));

        if (schedule) {
            intervalInput = numberInput("30", intervalMinutes);
            card.addView(UI.field(requireContext(), "间隔（分钟）", intervalInput, UI.MD));
        }

        UI.addRow(card, toggleRow("自动更新华为云 DNS", updateDns, next -> {
            capture();
            updateDns = next;
            render();
        }), UI.MD);

        if (updateDns) {
            zoneInput = UI.input(requireContext(), "example.com");
            zoneInput.setText(zone);
            card.addView(UI.field(requireContext(), "Zone 名称", zoneInput, UI.MD));

            zoneIdInput = UI.input(requireContext(), "填写后跳过 Zone 查询");
            zoneIdInput.setText(zoneId);
            card.addView(UI.field(requireContext(), "Zone ID（可选）", zoneIdInput, UI.MD));

            recordInput = UI.input(requireContext(), "cdn.example.com");
            recordInput.setText(record);
            card.addView(UI.field(requireContext(), "A 记录", recordInput, UI.MD));

            ttlInput = numberInput("300", ttl);
            card.addView(UI.field(requireContext(), "TTL", ttlInput, UI.MD));

            endpointInput = UI.input(requireContext(), "https://dns.myhuaweicloud.com");
            endpointInput.setInputType(InputType.TYPE_TEXT_VARIATION_URI);
            endpointInput.setText(endpoint);
            card.addView(UI.field(requireContext(), "API Endpoint", endpointInput, UI.MD));

            accessKeyInput = UI.input(requireContext(), "Access Key");
            accessKeyInput.setText(accessKey);
            card.addView(UI.field(requireContext(), "Access Key", accessKeyInput, UI.MD));

            secretInput = passwordInput(hasSecretKey ? "留空保持现有密钥" : "输入 Secret Access Key");
            secretInput.setText(secretDraft);
            LinearLayout secretField = UI.field(requireContext(), "Secret Key", secretInput, UI.MD);
            TextView state = UI.tag(requireContext(), hasSecretKey ? "密钥已设置" : "密钥未设置", hasSecretKey);
            UI.margin(state, 0, UI.SM, 0, 0);
            LinearLayout.LayoutParams slp = new LinearLayout.LayoutParams(
                    ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);
            state.setLayoutParams(slp);
            secretField.addView(state);
            card.addView(secretField);
        }

        TextView save = UI.button(requireContext(), busy ? "保存中…" : "保存配置", UI.BTN_PRIMARY);
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
        String summary = "尚未执行。";
        if (latest != null) {
            summary = formatTime(latest.optLong("started_at", 0))
                    + " · 扫描 " + latest.optInt("scanned", 0) + " 个，匹配 " + latest.optInt("matched", 0) + " 个";
            JSONArray selected = latest.optJSONArray("selected_ips");
            if (selected != null && selected.length() > 0) {
                summary += " · 选中 " + selected.length() + " 个";
            }
        }
        LinearLayout card = cardHead("最近结果", summary);

        JSONArray results = latest == null ? null : latest.optJSONArray("results");
        if (results == null || results.length() == 0) {
            card.addView(emptyState(latest == null ? "暂无探测结果，请保存配置后立即执行。" : "本轮没有命中任何 IP。"));
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
        LinearLayout card = cardHead("未命中原因", "未命中不等于 IP 完全不可用。显示 " + diagnostics.length()
                + " / " + latest.optInt("rejected", diagnostics.length()) + " 条，最多保留 100 条；可单独复测一个 IP。");
        JSONObject probe = latest.optJSONObject("probe");
        String statuses = probe == null ? "" : probe.optString("statuses");
        if (probe != null) card.addView(UI.muted(requireContext(), "本次探针：" + probe.optString("host")
                + probe.optString("path") + " · SNI " + probe.optString("sni") + " · 接受 " + statuses));
        for (int index = 0; index < diagnostics.length(); index++) {
            JSONObject result = diagnostics.optJSONObject(index);
            if (result == null) continue;
            LinearLayout row = UI.column(requireContext());
            row.addView(UI.mono(requireContext(), result.optString("ip") + " · " + result.optLong("latency_ms") + " ms", Theme.p().ink));
            String problem = result.optString("error");
            String detail = !problem.isEmpty() ? "探测失败：" + problem
                    : "HTTP " + result.optInt("status") + "，不在本次接受状态码 " + statuses + " 中";
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
                ? "本轮暂无分段统计。"
                : "共 " + total + " 个输入段，其中 " + missed.size() + " 段 0 命中，涉及 "
                        + missedTotal(missed) + " 个 IP。请先核对探针响应与未命中原因，再决定是否剔除。";

        LinearLayout card = cardHead("未命中 IP 段", summary);

        if (missed.isEmpty()) {
            card.addView(emptyState("暂无未命中的 IP 段。"));
            return card;
        }

        for (int i = 0; i < missed.size(); i++) {
            JSONObject item = missed.get(i);
            LinearLayout row = UI.column(requireContext());
            row.setPadding(0, UI.dp(UI.SM), 0, UI.dp(UI.SM));
            LinearLayout top = UI.row(requireContext());
            top.addView(UI.mono(requireContext(), item.optString("target"), Theme.p().ink),
                    new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
            top.addView(UI.tag(requireContext(), "可剔除", false));
            row.addView(top);
            TextView meta = UI.muted(requireContext(),
                    "已探测 " + item.optInt("scanned", 0) + " · 命中 " + item.optInt("matched", 0));
            UI.margin(meta, 0, UI.XS, 0, 0);
            row.addView(meta);
            UI.addRow(card, row, i == 0 ? 0 : UI.SM);
        }

        LinearLayout actions = UI.row(requireContext());
        TextView copy = UI.button(requireContext(), "复制未命中段", UI.BTN_SECONDARY);
        copy.setOnClickListener(v -> copyMissed(missed));
        UI.weight(copy, 1f);
        actions.addView(copy);
        TextView drop = UI.button(requireContext(), "一键剔除并保存", UI.BTN_DANGER);
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
        LinearLayout card = cardHead("执行历史", "最多保留 20 条记录。");
        if (runs.length() == 0) {
            card.addView(emptyState("暂无执行历史。"));
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
            top.addView(UI.tag(requireContext(), success ? "成功" : "失败", success));
            row.addView(top);

            TextView meta = UI.muted(requireContext(),
                    "扫描 " + item.optInt("scanned", 0) + " / 匹配 " + item.optInt("matched", 0)
                            + " · DNS " + (item.optBoolean("dns_updated", false) ? "已更新" : "未更新"));
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
                java.util.Locale.getDefault());
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
            banner = "任务已开始执行";
            bannerError = false;
            render();
            ui.removeCallbacks(pollTask);
            ui.postDelayed(pollTask, POLL_INTERVAL);
        }, failure -> {
            if (!viewActive || !alive()) return;
            busy = false;
            banner = "执行失败：" + failure.getMessage();
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
            banner = "实验功能配置已保存";
            bannerError = false;
            applySettings(result);
            render();
            if (afterSave != null) afterSave.run();
            else loadStatus();
        }, failure -> {
            if (!viewActive || !alive()) return;
            busy = false;
            banner = "保存失败：" + failure.getMessage();
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
            banner = "/stop".equals(action) ? "已请求停止，并关闭定时任务" : "/ownership/challenge".equals(action)
                    ? "请添加 TXT 记录，然后点击检查" : "域名验证通过；定时任务需手动开启";
            if ("/ownership/dns".equals(action)) {
                JSONObject operation = result.optJSONObject("operation");
                banner = operation == null ? "TXT 已提交，请检查解析结果" : operation.optString("message");
            }
            bannerError = false;
            render();
            schedulePoll();
        }, failure -> {
            if (!viewActive || !alive()) return;
            busy = false;
            if ("/ownership/verify".equals(action)) schedule = false;
            banner = "操作失败：" + failure.getMessage();
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
        banner = "未命中 IP 段已复制";
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
            banner = "当前输入中没有找到与最近一轮完全一致的未命中段";
            bannerError = true;
            render();
            return;
        }
        capture();
        Modal.of(requireContext(), "剔除未命中 IP 段")
                .message("确定剔除 " + missed.size() + " 个未命中段吗？将减少 " + missedTotal(missed)
                        + " 个 IP 的后续探测压力。")
                .cancel("取消")
                .confirm("剔除并保存", true, () -> {
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
