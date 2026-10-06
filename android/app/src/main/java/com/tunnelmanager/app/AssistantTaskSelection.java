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
            case "data": return "记录数据（SRV / CAA）";
            case "ttl": return "TTL（1 为自动）";
            case "proxied": return "代理";
            case "priority": return "MX 优先级";
            case "interval_sec": return "间隔（秒）";
            case "monitor_id": return "监控项目 ID";
            case "url": return "探测地址";
            case "id": return "对象 ID";
            case "record_id": return "DNS 记录 ID";
            case "target_id": return "探测目标 ID";
            case "description": return "站点描述";
            case "icon": return "站点图标";
            case "landing_enabled": return "落地页";
            case "registration_enabled": return "开放注册";
            case "invite_mode": return "邀请码模式";
            case "experimental_features_enabled": return "实验功能";
            case "value": return "优选 CNAME";
            case "items": return "优选 CNAME 预设";
            case "targets": return "探测目标";
            case "publish_enabled": return "公开监控";
            case "alert_enabled": return "监控告警";
            case "weight": return "权重";
            case "port": return "端口";
            case "target": return "目标主机";
            case "flags": return "标记";
            case "tag": return "标签";
            default: return key;
        }
    }
}
