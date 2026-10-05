package com.tunnelmanager.app;

import android.app.Activity;
import android.content.Context;
import android.content.ContextWrapper;
import android.os.Handler;
import android.os.Looper;
import android.text.Editable;
import android.text.InputType;
import android.text.TextWatcher;
import android.view.View;
import android.widget.EditText;
import android.widget.LinearLayout;
import android.widget.PopupMenu;
import android.widget.TextView;
import androidx.lifecycle.Lifecycle;
import androidx.lifecycle.LifecycleEventObserver;
import androidx.lifecycle.LifecycleOwner;
import org.json.JSONArray;
import org.json.JSONObject;
import java.util.ArrayList;
import java.util.List;
import java.util.Locale;

final class DNSBatchCreate {
    private final Context context;
    private final String zone, zoneName;
    private final JSONArray records;
    private final Runnable saved;
    private final RequestScope scope = new RequestScope(Session::snapshot);
    private final Handler main = new Handler(Looper.getMainLooper());
    private final DNSValues.Draft draft = new DNSValues.Draft();
    private final List<String> successful = new ArrayList<>();
    private EditText name, values, ttl, priority;
    private TextView type, proxy, previewText, error, resultsText, submit, acknowledge;
    private View priorityField, proxyField;
    private Sheet.Builder sheet;
    private Lifecycle lifecycle;
    private LifecycleEventObserver observer;
    private boolean running, locked, reviewRequired, reviewed, disposed;

    DNSBatchCreate(Context context, String zone, String zoneName, JSONArray records, Runnable saved) {
        this.context = context; this.zone = zone; this.zoneName = zoneName; this.records = records; this.saved = saved;
    }

    void show() {
        Context owner = context;
        while (owner instanceof ContextWrapper && !(owner instanceof Activity)) owner = ((ContextWrapper) owner).getBaseContext();
        if (!(owner instanceof Activity) || !(owner instanceof LifecycleOwner)
                || ((Activity) owner).isFinishing() || ((Activity) owner).isDestroyed()) {
            dispose();
            return;
        }
        lifecycle = ((LifecycleOwner) owner).getLifecycle();
        if (lifecycle.getCurrentState() == Lifecycle.State.DESTROYED) { dispose(); return; }
        LinearLayout form = UI.column(context);
        form.addView(UI.muted(context, "区域：" + zoneName + "\n每批最多 100 条非空行，逐条串行新增。支持 A / AAAA / TXT / MX / NS / PTR。CNAME、SRV、CAA 请单条添加。"));
        type = UI.button(context, draft.type, UI.BTN_SECONDARY);
        type.setOnClickListener(view -> {
            PopupMenu menu = new PopupMenu(context, type);
            for (String choice : DNSValues.BATCH_TYPES) menu.getMenu().add(choice);
            menu.setOnMenuItemClickListener(item -> {
                draft.type = item.getTitle().toString(); draft.proxied = false; type.setText(draft.type); sync(); return true;
            });
            menu.show();
        });
        form.addView(UI.field(context, "记录类型", type, UI.MD));
        name = UI.input(context, "www 或 example.com"); form.addView(UI.field(context, "名称", name, UI.MD));
        ttl = UI.input(context, "1"); ttl.setInputType(InputType.TYPE_CLASS_NUMBER); ttl.setText("1");
        form.addView(UI.field(context, "TTL（1 为自动）", ttl, UI.MD));
        priority = UI.input(context, "0–65535"); priority.setInputType(InputType.TYPE_CLASS_NUMBER); priority.setText("0");
        priorityField = UI.field(context, "MX 优先级（共用）", priority, UI.MD); form.addView(priorityField);
        proxy = UI.button(context, "仅 DNS", UI.BTN_SECONDARY);
        proxy.setOnClickListener(view -> { draft.proxied = !draft.proxied; sync(); });
        proxyField = UI.field(context, "代理状态", proxy, UI.MD); form.addView(proxyField);
        values = UI.textArea(context, "每行一个解析值"); values.setMinLines(4);
        form.addView(UI.field(context, "解析值，每行一条", values, UI.MD));
        form.addView(UI.muted(context, "空行忽略，重复和已加载的同名同类型值跳过。TXT 保留空格，不按逗号拆分；跨行 TXT 请单条添加。"));
        previewText = UI.mono(context, "", Theme.p().body); form.addView(previewText);
        error = UI.muted(context, ""); error.setTextColor(Theme.p().error); form.addView(error);
        resultsText = UI.mono(context, "", Theme.p().body); form.addView(resultsText);
        acknowledge = UI.button(context, "核对后点击：我已移除实际创建的行", UI.BTN_SECONDARY);
        acknowledge.setVisibility(View.GONE); UI.fill(acknowledge);
        acknowledge.setOnClickListener(view -> {
            reviewed = !reviewed;
            acknowledge.setText(reviewed ? "已核对并移除实际创建的行 ✓" : "核对后点击：我已移除实际创建的行");
            refreshPreview();
        });
        form.addView(acknowledge);
        submit = UI.button(context, "批量新增", UI.BTN_PRIMARY); UI.fill(submit); UI.margin(submit, 0, UI.MD, 0, 0);
        submit.setOnClickListener(view -> send()); form.addView(submit);
        for (EditText input : new EditText[]{name, values, ttl, priority}) input.addTextChangedListener(new TextWatcher() {
            public void beforeTextChanged(CharSequence text, int start, int count, int after) {}
            public void onTextChanged(CharSequence text, int start, int before, int count) {}
            public void afterTextChanged(Editable text) { if (!running) refreshPreview(); }
        });
        sheet = Sheet.of(context, "批量新增 DNS 记录").content(form).onDismiss(this::dispose);
        observer = (source, event) -> { if (event == Lifecycle.Event.ON_DESTROY) dispose(); };
        lifecycle.addObserver(observer);
        sync(); sheet.show();
    }

    private void dispose() {
        if (disposed) return;
        disposed = true;
        scope.stop("窗口已关闭或宿主已销毁，队列已停止");
        if (lifecycle != null && observer != null) lifecycle.removeObserver(observer);
        if (sheet != null && sheet.isShowing()) sheet.dismiss();
    }

    private void sync() {
        priorityField.setVisibility(draft.type.equals("MX") ? View.VISIBLE : View.GONE);
        proxyField.setVisibility(DNSValues.proxy(draft.type) ? View.VISIBLE : View.GONE);
        proxy.setText(draft.proxied ? "已代理" : "仅 DNS"); refreshPreview();
    }

    private DNSValues.Preview preview() {
        draft.name = name.getText().toString(); draft.ttl = ttl.getText().toString(); draft.priority = priority.getText().toString();
        String fullName = DNSValues.trimDot(draft.name.trim()).toLowerCase(Locale.ROOT);
        String suffix = zoneName.toLowerCase(Locale.ROOT);
        if (fullName.equals("@")) fullName = suffix;
        else if (!fullName.equals(suffix) && !fullName.endsWith("." + suffix)) fullName += "." + suffix;
        List<String> existing = new ArrayList<>(successful);
        for (int index = 0; index < records.length(); index++) {
            JSONObject record = records.optJSONObject(index);
            if (record != null && record.optString("type").equals(draft.type) && DNSValues.trimDot(record.optString("name")).equalsIgnoreCase(fullName)) existing.add(record.optString("content"));
        }
        DNSValues.Preview result = DNSValues.parse(draft.type, values.getText().toString(), existing);
        for (int index = 0; index < result.rows.size(); index++) {
            DNSValues.Row row = result.rows.get(index);
            if (row.error != null || row.skipped != null) continue;
            draft.content = row.content;
            String problem = draft.validate();
            if (problem != null) {
                result.rows.set(index, new DNSValues.Row(row.line, row.content, problem, row.skipped));
                result.error = "请先修正标注行，整个批次尚未提交";
            }
        }
        return result;
    }

    private void refreshPreview() {
        DNSValues.Preview result = preview();
        StringBuilder text = new StringBuilder("预览：" + result.pending().size() + " 条待新增\n");
        for (DNSValues.Row row : result.rows.subList(0, Math.min(result.rows.size(), DNSValues.LIMIT))) {
            text.append("第 ").append(row.line).append(" 行 · ").append(row.content).append("\n")
                    .append(row.error != null ? row.error : row.skipped != null ? row.skipped : "待新增").append("\n");
        }
        String stopped = scope.reason();
        previewText.setText(text); error.setText(stopped != null ? stopped : result.error == null ? "" : result.error);
        submit.setEnabled(!disposed && stopped == null && !running && result.error == null && !result.pending().isEmpty() && (!reviewRequired || reviewed));
        submit.setText(locked ? "重试剩余项" : "新增 " + result.pending().size() + " 条记录");
    }

    private void send() {
        if (disposed || running || (reviewRequired && !reviewed)) return;
        if (scope.reason() != null) { refreshPreview(); return; }
        DNSValues.Preview parsed = preview();
        if (parsed.error != null || parsed.pending().isEmpty()) { refreshPreview(); return; }
        List<DNSValues.Row> rows = parsed.pending();
        running = true; locked = true; reviewRequired = false; reviewed = false;
        acknowledge.setText("核对后点击：我已移除实际创建的行"); acknowledge.setVisibility(View.GONE); setControls(); resultsText.setText("");
        StringBuilder report = new StringBuilder();
        Api.async(() -> DNSValues.send(rows, row -> {
            scope.check();
            draft.content = row.content;
            try {
                JSONObject created = Api.post(scope, "/api/zones/" + zone + "/dns-records", DNSJson.payload(draft, null));
                if (created.optString("id").isEmpty()) throw new DNSValues.SendFailure("缺少创建记录确认", true);
            }
            catch (Api.Failure failure) {
                String detail = String.valueOf(failure.getMessage());
                boolean uncertain = failure.code == 0 || failure.code == 408 || failure.code >= 500
                        || detail.matches("(?is).*(request failed|read response failed|parse response failed|timeout|deadline|connection|network|EOF).*");
                throw new DNSValues.SendFailure(detail, uncertain);
            }
        }, result -> main.post(() -> {
            if (disposed) return;
            if (result.uncertain) { reviewRequired = true; acknowledge.setVisibility(View.VISIBLE); }
            report.append("第 ").append(result.row.line).append(" 行 · ").append(result.row.content).append("\n").append(result.message).append("\n");
            resultsText.setText(report);
        })), completed -> {
            StringBuilder failures = new StringBuilder();
            int count = 0, unknown = 0, notSent = 0;
            for (DNSValues.Result result : completed) {
                if (result.success) { count++; successful.add(result.row.content); }
                else { if (failures.length() > 0) failures.append('\n'); failures.append(result.row.content); }
                if (result.uncertain) unknown++;
                if (result.notSent) notSent++;
            }
            running = false;
            if (disposed) return;
            values.setText(failures);
            report.append("成功 ").append(count).append("，失败 ").append(completed.size() - count - unknown - notSent)
                    .append("，未知 ").append(unknown).append("，未发送 ").append(notSent)
                    .append("。只保留未完成项，绝不自动重发成功行。未发送行未执行请求。");
            if (unknown > 0) report.append("未知请求可能已写入，请核对并移除实际创建的行后再重试。");
            report.append("关闭窗口会清空本批输入。");
            resultsText.setText(report); setControls(); refreshPreview();
            if (scope.reason() == null) saved.run();
        }, failure -> {
            running = false;
            if (disposed) return;
            setControls(); error.setText(failure.getMessage());
        });
    }

    private void setControls() {
        boolean enabled = !disposed && !running && scope.reason() == null;
        sheet.setDismissible(!running); values.setEnabled(enabled);
        acknowledge.setEnabled(enabled);
        for (View control : new View[]{name, ttl, priority, type, proxy}) control.setEnabled(enabled && !locked);
        submit.setEnabled(enabled); if (running) submit.setText("正在逐条新增…");
    }
}
