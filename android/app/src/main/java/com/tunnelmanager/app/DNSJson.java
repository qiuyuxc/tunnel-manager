package com.tunnelmanager.app;

import org.json.JSONObject;
import java.util.Arrays;
import java.util.Iterator;
import java.util.List;

final class DNSJson {
    static DNSValues.Draft draft(JSONObject record) {
        DNSValues.Draft draft = new DNSValues.Draft();
        if (record == null) return draft;
        draft.type = record.optString("type"); draft.name = record.optString("name");
        draft.ttl = String.valueOf(record.optInt("ttl", 1)); draft.proxied = record.optBoolean("proxied");
        draft.content = record.optString("content"); draft.priority = String.valueOf(record.optInt("priority", 0));
        JSONObject data = record.optJSONObject("data");
        if (data != null && draft.type.equals("SRV")) {
            draft.priority = data.optString("priority", ""); draft.weight = data.optString("weight", "");
            draft.port = data.optString("port", ""); draft.target = data.optString("target", "");
        }
        if (data != null && draft.type.equals("CAA")) {
            draft.flags = data.optString("flags", ""); draft.tag = data.optString("tag", ""); draft.value = data.optString("value", "");
        }
        return draft;
    }

    static boolean editable(JSONObject record) {
        String type = record.optString("type");
        if (!DNSValues.editable(type)) return false;
        JSONObject data = record.optJSONObject("data");
        if (!DNSValues.structured(type)) return data == null || data.length() == 0;
        if (data == null) return false;
        List<String> allowed = type.equals("SRV") ? Arrays.asList("priority", "weight", "port", "target", "name", "service", "proto") : Arrays.asList("flags", "tag", "value");
        for (Iterator<String> keys = data.keys(); keys.hasNext();) if (!allowed.contains(keys.next())) return false;
        for (String key : type.equals("SRV") ? new String[]{"priority", "weight", "port", "target"} : new String[]{"flags", "tag", "value"}) {
            if (!data.has(key) || data.isNull(key)) return false;
        }
        return draft(record).validate() == null;
    }

    static JSONObject payload(DNSValues.Draft draft, JSONObject original) throws Exception {
        String error = draft.validate();
        if (error != null) throw new IllegalArgumentException(error);
        JSONObject result = new JSONObject();
        result.put("type", draft.type); result.put("name", DNSValues.trimDot(draft.name.trim()));
        result.put("ttl", DNSValues.number(draft.ttl)); result.put("proxied", DNSValues.proxy(draft.type) && draft.proxied);
        if (DNSValues.structured(draft.type)) {
            JSONObject data = new JSONObject();
            if (draft.type.equals("SRV")) {
                data.put("priority", DNSValues.number(draft.priority)); data.put("weight", DNSValues.number(draft.weight));
                data.put("port", DNSValues.number(draft.port)); data.put("target", draft.target.trim());
                JSONObject oldData = original == null || !draft.type.equals(original.optString("type")) ? null : original.optJSONObject("data");
                if (oldData != null) {
                    String[] labels = result.getString("name").split("\\.", 3);
                    if (oldData.has("service")) data.put("service", labels[0]);
                    if (oldData.has("proto")) data.put("proto", labels[1]);
                    if (oldData.has("name")) data.put("name", labels[2]);
                }
            } else {
                data.put("flags", DNSValues.number(draft.flags)); data.put("tag", draft.tag); data.put("value", draft.value);
            }
            result.put("data", data);
        } else {
            result.put("content", draft.type.equals("TXT") ? draft.content : draft.content.trim());
            if (draft.type.equals("MX")) result.put("priority", DNSValues.number(draft.priority));
        }
        return result;
    }
}
