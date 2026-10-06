package com.tunnelmanager.app;

import android.content.Context;
import android.text.Editable;
import android.text.InputType;
import android.text.TextWatcher;
import android.widget.CheckBox;
import android.widget.EditText;
import android.widget.LinearLayout;
import android.widget.TextView;
import android.view.View;
import android.view.ViewGroup;
import org.json.JSONArray;
import org.json.JSONObject;
import java.util.Iterator;

/** Edits a private copy; the server validates it before it becomes confirmable. */
final class AssistantTaskEditor {
    final JSONObject draft;
    final LinearLayout form;
    private final Context context;
    private final String tool;

    AssistantTaskEditor(Context context, JSONObject task) throws Exception {
        this.context = context;
        this.tool = task.optString("tool");
        draft = new JSONObject(task.getJSONObject("arguments").toString());
        form = UI.column(context);
        render();
    }

    private interface Setter { void set(Object value) throws Exception; }

    void setEnabled(boolean enabled) { setTreeEnabled(form, enabled); }
    private void setTreeEnabled(View view, boolean enabled) {
        view.setEnabled(enabled);
        if (view instanceof ViewGroup) {
            ViewGroup group = (ViewGroup) view;
            for (int index = 0; index < group.getChildCount(); index++) setTreeEnabled(group.getChildAt(index), enabled);
        }
    }

    private void render() {
        form.removeAllViews();
        form.addView(UI.muted(context, "修改后保存，再勾选确认执行。保存不会执行任务。"));
        Iterator<String> keys = draft.keys();
        while (keys.hasNext()) {
            String key = keys.next();
            if ("update_dns_record".equals(tool) && ("zone_id".equals(key) || "record_id".equals(key))) {
                form.addView(UI.muted(context, AssistantTaskSelection.argumentLabel(key) + "：" + draft.optString(key)));
            } else if ("type".equals(key) && tool.contains("dns_record")) {
                TextView choose = UI.button(context, "记录类型：" + draft.optString(key), UI.BTN_SECONDARY);
                choose.setOnClickListener(view -> chooseDNSType());
                form.addView(choose);
            } else addField(form, key, draft.opt(key), value -> draft.put(key, value));
        }
    }

    private void chooseDNSType() {
        Sheet.Builder sheet = Sheet.of(context, "选择记录类型");
        for (String type : new String[]{"A", "AAAA", "CNAME", "TXT", "MX", "NS", "SRV", "CAA", "PTR"}) {
            sheet.action(R.drawable.ic_nav_assistant, type, "切换后请核对记录值", () -> {
                if (type.equals(draft.optString("type"))) return;
                try {
                    draft.put("type", type); draft.remove("data"); draft.remove("priority"); draft.remove("content");
                    if (!"A".equals(type) && !"AAAA".equals(type) && !"CNAME".equals(type)) draft.put("proxied", false);
                    if ("SRV".equals(type)) draft.put("data", new JSONObject().put("priority", 0).put("weight", 0).put("port", 0).put("target", ""));
                    else if ("CAA".equals(type)) draft.put("data", new JSONObject().put("flags", 0).put("tag", "issue").put("value", ""));
                    else { draft.put("content", ""); if ("MX".equals(type)) draft.put("priority", 0); }
                    render();
                } catch (Exception ignored) { }
            });
        }
        sheet.show();
    }

    private void addField(LinearLayout parent, String key, Object value, Setter setter) {
        String label = AssistantTaskSelection.argumentLabel(key);
        if (value instanceof JSONObject) {
            parent.addView(UI.label(context, label));
            JSONObject object = (JSONObject) value;
            Iterator<String> keys = object.keys();
            while (keys.hasNext()) {
                String child = keys.next();
                addField(parent, child, object.opt(child), updated -> object.put(child, updated));
            }
        } else if (value instanceof JSONArray) {
            parent.addView(UI.label(context, label));
            JSONArray array = (JSONArray) value;
            for (int i = 0; i < array.length(); i++) {
                final int index = i;
                addField(parent, "第 " + (i + 1) + " 项", array.opt(i), updated -> array.put(index, updated));
                TextView remove = UI.button(context, "移除这项", UI.BTN_GHOST);
                remove.setOnClickListener(view -> { array.remove(index); render(); });
                parent.addView(remove);
            }
            TextView add = UI.button(context, "添加预设", UI.BTN_SECONDARY);
            add.setOnClickListener(view -> { try { array.put(new JSONObject().put("name", "").put("value", "")); render(); } catch (Exception ignored) { } });
            parent.addView(add);
        } else if (value instanceof Boolean) {
            CheckBox check = new CheckBox(context);
            check.setText(label); check.setTextColor(Theme.p().ink); check.setChecked((Boolean) value);
            check.setOnCheckedChangeListener((button, checked) -> { try { setter.set(checked); } catch (Exception ignored) { } });
            parent.addView(check);
        } else if ("invite_mode".equals(key)) {
            parent.addView(UI.label(context, label));
            String[] modes = {"off", "optional", "required"};
            int selected = "required".equals(value) ? 2 : "optional".equals(value) ? 1 : 0;
            parent.addView(UI.segmented(context, new String[]{"不使用", "可选", "必须"}, selected, index -> { try { setter.set(modes[index]); } catch (Exception ignored) { } }));
        } else {
            EditText input = UI.input(context, label);
            boolean number = value instanceof Number;
            if (number) input.setInputType(InputType.TYPE_CLASS_NUMBER | InputType.TYPE_NUMBER_FLAG_SIGNED);
            else input.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_FLAG_MULTI_LINE);
            input.setText(String.valueOf(value));
            input.addTextChangedListener(new TextWatcher() {
                public void beforeTextChanged(CharSequence text, int start, int count, int after) { }
                public void onTextChanged(CharSequence text, int start, int before, int count) { }
                public void afterTextChanged(Editable text) {
                    Object updated = text.toString();
                    if (number) { try { updated = Long.parseLong(text.toString()); } catch (NumberFormatException ignored) { } }
                    try { setter.set(updated); } catch (Exception ignored) { }
                }
            });
            parent.addView(UI.field(context, label, input));
        }
    }
}
