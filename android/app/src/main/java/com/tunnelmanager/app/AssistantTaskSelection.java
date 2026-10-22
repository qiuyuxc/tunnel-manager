package com.tunnelmanager.app;

import java.util.ArrayList;
import java.util.Collection;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Set;

final class AssistantTaskSelection {
    private String conversationID = "";
    private final Set<String> pending = new LinkedHashSet<>();
    private final Set<String> selected = new LinkedHashSet<>();

    void sync(String currentID, Collection<String> pendingIDs) {
        if (!conversationID.equals(currentID)) selected.clear();
        conversationID = currentID;
        pending.clear();
        pending.addAll(pendingIDs);
        selected.retainAll(pending);
    }

    void select(String taskID, boolean checked) {
        if (checked && pending.contains(taskID)) selected.add(taskID);
        else selected.remove(taskID);
    }

    boolean contains(String taskID) { return selected.contains(taskID); }
    int size() { return selected.size(); }
    void clear() { selected.clear(); }

    List<String> snapshot(String currentID, boolean busy) {
        if (busy || currentID.isEmpty() || !conversationID.equals(currentID)) return new ArrayList<>();
        return new ArrayList<>(selected);
    }

    static String argumentLabel(String key) {
        switch (key) {
            case "name": return I18n.text("m_d44e9b3d3b31");
            case "zone_id": return I18n.text("m_d8399be73059");
            case "tunnel_id": return I18n.text("m_17f261a9385c");
            case "hostname": return I18n.text("m_1ac79aa3aff0");
            case "service_url": return I18n.text("m_693f764b0b0b");
            case "type": return I18n.text("m_ba40014ff496");
            case "content": return I18n.text("m_7175aa5fa2ba");
            case "data": return I18n.text("m_98d11221a1cd");
            case "ttl": return I18n.text("m_5fada8da5e20");
            case "proxied": return I18n.text("m_5e84ea61e838");
            case "priority": return I18n.text("m_3b1f7c9f5b21");
            case "interval_sec": return I18n.text("m_45455dc4b517");
            case "monitor_id": return I18n.text("m_b57b23459372");
            case "url": return I18n.text("m_c935cb3a1ded");
            case "id": return I18n.text("m_b0719015e486");
            case "record_id": return I18n.text("m_79081a3c812e");
            case "target_id": return I18n.text("m_f621ef3eb10c");
            case "description": return I18n.text("m_9ddde2b92952");
            case "icon": return I18n.text("m_e220e37bbb93");
            case "landing_enabled": return I18n.text("m_6a654dc0b0c0");
            case "registration_enabled": return I18n.text("m_04f83ab519b2");
            case "invite_mode": return I18n.text("m_936bac542e64");
            case "experimental_features_enabled": return I18n.text("m_001a51a407db");
            case "value": return I18n.text("m_ff425d8a519d");
            case "items": return I18n.text("m_4a385e41088a");
            case "targets": return I18n.text("m_100d9e4baf2a");
            case "publish_enabled": return I18n.text("m_4521b52c08c9");
            case "alert_enabled": return I18n.text("m_6406d409c964");
            case "weight": return I18n.text("m_db18831a0457");
            case "port": return I18n.text("m_e71ac32b544b");
            case "target": return I18n.text("m_252e26071ac9");
            case "flags": return I18n.text("m_269635727321");
            case "tag": return I18n.text("m_1d0fd5f9336d");
            default: return key;
        }
    }
}
