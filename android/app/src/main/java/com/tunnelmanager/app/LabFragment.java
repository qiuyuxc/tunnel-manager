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
    private final Handler ui = new Handler(Looper.getMainLooper());

    private JSONObject status = new JSONObject();
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
            if (!isAdded()) return;
            loadStatus();
        }
    };

    @Override
    String route() {
        return "/lab/ip-selector";
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
            payload.put("settings", Api.get("/api/lab/ip-selector"));
            payload.put("status", Api.get("/api/lab/ip-selector/status"));
            return payload;
        }, payload -> {
            loading = false;
            banner = "";
            applySettings(payload.optJSONObject("settings"));
            applyStatus(payload.optJSONObject("status"));
            render();
            if (running) {
                ui.removeCallbacks(pollTask);
                ui.postDelayed(pollTask, POLL_INTERVAL);
            }
        }, failure -> {
            loading = false;
            banner = "加载失败：" + failure.getMessage();
            bannerError = true;
            render();
        });
    }

    private void loadStatus() {
        Api.async(() -> Api.get("/api/lab/ip-selector/status"), payload -> {
            boolean wasRunning = running;
            applyStatus(payload);
            if (running) {
                // A repaint in place keeps whatever the operator is typing alive.
                if (wasRunning) {
                    fillProgress();
                } else {
                    render();
                }
                ui.removeCallbacks(pollTask);
                ui.postDelayed(pollTask, POLL_INTERVAL);
            } else {
                ui.removeCallbacks(pollTask);
                if (wasRunning) {
                    banner = "任务已完成";
                    bannerError = false;
                    render();
                }
            }
        }, failure -> {
            ui.removeCallbacks(pollTask);
        });
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
                "直连探测指定 IP 段，按延迟挑选可用 IP，并可自动更新华为云 DNS 的 A 记录");
        UI.margin(subtitle, 0, UI.XS, 0, UI.MD);
        body.addView(subtitle);

        if (loading) {
            body.addView(UI.muted(requireContext(), "加载中…"));
            return;
        }

        body.addView(toolbar());
        UI.addRow(body, progressSlot = UI.column(requireContext()), UI.MD);
        fillProgress();
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(probeCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(autoCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(resultsCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(missedCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(historyCard());

        if (!banner.isEmpty()) {
            TextView view = UI.banner(requireContext(), banner, bannerError);
            UI.margin(view, 0, UI.MD, 0, 0);
            body.addView(view);
        }
        UI.restoreScroll(scroll, keepScroll);
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
        refresh.setOnClickListener(v -> load());
        head.addView(refresh);
        card.addView(head);

        TextView run = UI.button(requireContext(), running ? "运行中…" : "立即执行", UI.BTN_PRIMARY);
        run.setEnabled(!running && !busy);
        run.setOnClickListener(v -> startRun());
        UI.margin(run, 0, UI.MD, 0, 0);
        card.addView(run);
        return card;
    }

    private String phaseText() {
        String phase = phase();
        if ("preparing".equals(phase)) return "准备扫描…";
        if ("scanning".equals(phase)) return "正在探测 IP…";
        if ("updating_dns".equals(phase)) return "扫描完成，正在更新华为云 DNS…";
        if ("completed".equals(phase)) return "任务已完成";
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
                UI.field(requireContext(), "并发数", workersInput)), UI.MD);

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
                : "共 " + total + " 个输入段，其中 " + missed.size() + " 段 0 命中，可减少 "
                        + missedTotal(missed) + " 个 IP 的后续探测压力。";

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
        busy = true;
        banner = "";
        render();
        Api.async(() -> Api.post("/api/lab/ip-selector/run", new JSONObject()), result -> {
            busy = false;
            running = true;
            banner = "任务已开始执行";
            bannerError = false;
            render();
            ui.removeCallbacks(pollTask);
            ui.postDelayed(pollTask, POLL_INTERVAL);
        }, failure -> {
            busy = false;
            banner = "执行失败：" + failure.getMessage();
            bannerError = true;
            render();
        });
    }

    private void save() {
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
            busy = false;
            banner = "实验功能配置已保存";
            bannerError = false;
            load();
        }, failure -> {
            busy = false;
            banner = "保存失败：" + failure.getMessage();
            bannerError = true;
            render();
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
