package com.tunnelmanager.app;

import android.content.Context;
import android.text.InputType;
import android.view.View;
import android.widget.EditText;
import android.widget.LinearLayout;
import android.widget.PopupMenu;
import android.widget.TextView;
import org.json.JSONObject;
import java.util.LinkedHashMap;
import java.util.Map;

final class DNSRecordEditor {
    private final Context context;
    private final String zone;
    private final JSONObject original;
    private final Runnable saved;
    private final DNSValues.Draft draft;
    private final Map<String, EditText> inputs = new LinkedHashMap<>();
    private final Map<String, View> fields = new LinkedHashMap<>();
    private LinearLayout form;
    private TextView type, proxy, error, save;
    private Sheet.Builder sheet;
    private boolean saving;

    DNSRecordEditor(Context context, String zone, JSONObject record, Runnable saved) {
        this.context = context; this.zone = zone; this.original = record; this.saved = saved;
        draft = DNSJson.draft(record);
    }

    void show() {
        form = UI.column(context);
        type = UI.button(context, draft.type, UI.BTN_SECONDARY);
        type.setOnClickListener(view -> {
            PopupMenu menu = new PopupMenu(context, type);
            for (String choice : DNSValues.TYPES) menu.getMenu().add(choice);
            menu.setOnMenuItemClickListener(item -> {
                draft.type = item.getTitle().toString(); draft.proxied = false;
                inputs.get("content").setText("");
                sync(); return true;
            });
            menu.show();
        });
        form.addView(UI.field(context, I18n.text("m_8f6363220575"), type, UI.MD));
        field("name", I18n.text("m_ae45cd6fbacf"), draft.name, false);
        field("content", I18n.text("m_a76ff9e03742"), draft.content, false);
        field("priority", I18n.text("m_fefb7dd1c2dd"), draft.priority, true);
        field("weight", I18n.text("m_8e12951d4f9b"), draft.weight, true);
        field("port", I18n.text("m_b6315eb5f488"), draft.port, true);
        field("target", I18n.text("m_db4ee932a0d3"), draft.target, false);
        field("flags", "CAA flags（0–255）", draft.flags, true);
        field("tag", "CAA tag（issue / issuewild / iodef）", draft.tag, false);
        field("value", "CAA value", draft.value, false);
        field("ttl", I18n.text("m_1005edfc97dd"), draft.ttl, true);
        proxy = UI.button(context, "", UI.BTN_SECONDARY);
        proxy.setOnClickListener(view -> { draft.proxied = !draft.proxied; sync(); });
        form.addView(UI.field(context, I18n.text("m_30efb7690431"), proxy, UI.MD));
        error = UI.muted(context, ""); error.setTextColor(Theme.p().error); form.addView(error);
        save = UI.button(context, original == null ? I18n.text("m_42845d86b969") : I18n.text("m_ecf0ea01192d"), UI.BTN_PRIMARY);
        UI.fill(save); UI.margin(save, 0, UI.MD, 0, 0); form.addView(save);
        save.setOnClickListener(view -> submit());
        sheet = Sheet.of(context, original == null ? I18n.text("m_b6e583370f98") : I18n.text("m_aa4d3660ca67")).content(form);
        sync(); sheet.show();
    }

    private void field(String key, String label, String value, boolean numeric) {
        EditText input = key.equals("content") ? UI.textArea(context, I18n.text("m_5632a6a4e542")) : UI.input(context, label);
        if (numeric) input.setInputType(InputType.TYPE_CLASS_NUMBER);
        input.setText(value);
        View field = UI.field(context, label, input, UI.MD);
        inputs.put(key, input); fields.put(key, field); form.addView(field);
    }

    private void sync() {
        type.setText(draft.type);
        fields.get("content").setVisibility(DNSValues.structured(draft.type) ? View.GONE : View.VISIBLE);
        fields.get("priority").setVisibility(draft.type.equals("MX") || draft.type.equals("SRV") ? View.VISIBLE : View.GONE);
        for (String key : new String[]{"weight", "port", "target"}) fields.get(key).setVisibility(draft.type.equals("SRV") ? View.VISIBLE : View.GONE);
        for (String key : new String[]{"flags", "tag", "value"}) fields.get(key).setVisibility(draft.type.equals("CAA") ? View.VISIBLE : View.GONE);
        ((View) proxy.getParent()).setVisibility(DNSValues.proxy(draft.type) ? View.VISIBLE : View.GONE);
        proxy.setText(draft.proxied ? I18n.text("m_5c7bbf98286f") : I18n.text("m_80848ec9f9a4"));
    }

    private String text(String key) { return inputs.get(key).getText().toString(); }

    private void submit() {
        if (saving) return;
        draft.name = text("name"); draft.content = text("content"); draft.ttl = text("ttl");
        draft.priority = text("priority"); draft.weight = text("weight"); draft.port = text("port"); draft.target = text("target");
        draft.flags = text("flags"); draft.tag = text("tag"); draft.value = text("value");
        String problem = draft.validate();
        if (problem != null) { error.setText(problem); return; }
        final JSONObject payload;
        try { payload = DNSJson.payload(draft, original); }
        catch (Exception failure) { error.setText(failure.getMessage()); return; }
        setSaving(true); error.setText("");
        Api.async(() -> original == null ? Api.post("/api/zones/" + zone + "/dns-records", payload)
                : Api.put("/api/zones/" + zone + "/dns-records/" + original.optString("id"), payload), result -> {
            setSaving(false); sheet.dismiss(); saved.run();
        }, failure -> { setSaving(false); error.setText(failure.getMessage()); });
    }

    private void setSaving(boolean value) {
        saving = value; sheet.setDismissible(!value);
        for (EditText input : inputs.values()) input.setEnabled(!value);
        type.setEnabled(!value); proxy.setEnabled(!value); save.setEnabled(!value);
        save.setText(value ? I18n.text("m_6bdb4435095e") : original == null ? I18n.text("m_42845d86b969") : I18n.text("m_ecf0ea01192d"));
    }
}
