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
            case "name": return "名称";
            case "zone_id": return "区域 ID";
            case "tunnel_id": return "隧道 ID";
            case "hostname": return "公开域名";
            case "service_url": return "源站地址";
            case "type": return "类型";
            case "content": return "记录值";
            case "ttl": return "TTL（1 为自动）";
            case "proxied": return "代理";
            case "priority": return "MX 优先级";
            case "interval_sec": return "间隔（秒）";
            case "monitor_id": return "监控项目 ID";
            case "url": return "探测地址";
            default: return key;
        }
    }
}
