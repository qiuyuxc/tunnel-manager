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
        form.addView(UI.muted(context, I18n.text("m_b385ac5af51e")));
        Iterator<String> keys = draft.keys();
        while (keys.hasNext()) {
            String key = keys.next();
            if ("update_dns_record".equals(tool) && ("zone_id".equals(key) || "record_id".equals(key))) {
                form.addView(UI.muted(context, AssistantTaskSelection.argumentLabel(key) + "：" + draft.optString(key)));
            } else if ("type".equals(key) && tool.contains("dns_record")) {
                TextView choose = UI.button(context, I18n.text("m_a86b7b28351b") + draft.optString(key), UI.BTN_SECONDARY);
                choose.setOnClickListener(view -> chooseDNSType());
                form.addView(choose);
            } else addField(form, key, draft.opt(key), value -> draft.put(key, value));
        }
    }

    private void chooseDNSType() {
        Sheet.Builder sheet = Sheet.of(context, I18n.text("m_e17cf46a4083"));
        for (String type : new String[]{"A", "AAAA", "CNAME", "TXT", "MX", "NS", "SRV", "CAA", "PTR"}) {
            sheet.action(R.drawable.ic_nav_assistant, type, I18n.text("m_4c24d8c28433"), () -> {
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
                addField(parent, I18n.text("m_9552fe3bd366") + (i + 1) + I18n.text("m_f9d529eacdf3"), array.opt(i), updated -> array.put(index, updated));
                TextView remove = UI.button(context, I18n.text("m_b80f8ec9d893"), UI.BTN_GHOST);
                remove.setOnClickListener(view -> { array.remove(index); render(); });
                parent.addView(remove);
            }
            TextView add = UI.button(context, I18n.text("m_1073a088d1a0"), UI.BTN_SECONDARY);
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
            parent.addView(UI.segmented(context, new String[]{I18n.text("m_dd21dd9bdd9a"), I18n.text("m_537352400c49"), I18n.text("m_2c08cddfb13b")}, selected, index -> { try { setter.set(modes[index]); } catch (Exception ignored) { } }));
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
