package com.tunnelmanager.app;

import androidx.lifecycle.MutableLiveData;
import androidx.lifecycle.ViewModel;

import org.json.JSONObject;

import java.util.concurrent.Callable;

public final class MonitorDetailState extends ViewModel {
    final MutableLiveData<Integer> revision = new MutableLiveData<>(0);
    String monitorId = "";
    String selectedId = "";
    String error = "";
    String notice = "";
    String operation = "";
    String form = "";
    String formError = "";
    String targetId = "";
    String name = "";
    String url = "";
    String interval = "60";
    int type;
    int method;
    boolean loading;
    boolean busy;
    boolean deleted;
    boolean cleared;
    JSONObject monitor;
    private int generation;

    public MonitorDetailState() {}

    void changed() {
        if (!cleared) revision.setValue(revision.getValue() + 1);
    }

    void load() {
        if (busy || loading || cleared || deleted) return;
        int request = ++generation;
        loading = true;
        error = "";
        changed();
        Api.async(() -> Api.get("/api/monitors/" + monitorId), updated -> {
            if (cleared || request != generation) return;
            loading = false;
            monitor = updated;
            changed();
        }, failure -> {
            if (cleared || request != generation) return;
            loading = false;
            error = failure.getMessage() == null ? "加载失败，请重试" : failure.getMessage();
            changed();
        });
    }

    void openForm(String kind, JSONObject target) {
        if (busy) return;
        form = kind;
        formError = "";
        if ("edit".equals(kind) && target != null) {
            targetId = target.optString("id");
            name = target.optString("name");
            url = target.optString("url");
        } else if ("add".equals(kind)) {
            targetId = "";
            name = "";
            url = "";
            type = 0;
            method = 0;
        } else if ("interval".equals(kind) && monitor != null) {
            interval = String.valueOf(monitor.optInt("interval_sec", 60));
        }
        changed();
    }

    void submitForm() {
        if (busy || form.isEmpty()) return;
        try {
            JSONObject payload = new JSONObject();
            if ("interval".equals(form)) {
                int seconds;
                try {
                    seconds = Integer.parseInt(interval.trim());
                } catch (NumberFormatException failure) {
                    formError = "请输入有效的检测间隔";
                    changed();
                    return;
                }
                if (seconds < 30) {
                    formError = "检测间隔至少为 30 秒";
                    changed();
                    return;
                }
                payload.put("interval_sec", seconds);
                perform("保存设置", () -> Api.put(path(), payload), "检测间隔已保存", true);
                return;
            }
            if (name.trim().isEmpty() || url.trim().isEmpty()) {
                formError = "请填写服务名称和地址";
                changed();
                return;
            }
            payload.put("name", name.trim());
            payload.put("url", url.trim());
            if ("add".equals(form)) {
                payload.put("type", new String[]{"http", "tcp", "icmp"}[type]);
                if (type == 0) payload.put("method", method == 0 ? "GET" : "POST");
                perform("添加服务", () -> Api.post(path() + "/targets", payload), "服务已添加", false);
            } else {
                String targetPath = path() + "/targets/" + targetId;
                perform("保存服务", () -> Api.put(targetPath, payload), "服务已更新", false);
            }
        } catch (org.json.JSONException failure) {
            formError = "无法生成请求，请检查输入";
            changed();
        }
    }

    void checkNow() {
        perform("立即检测", () -> {
            JSONObject result = Api.post(path() + "/check", new JSONObject());
            return result.getJSONObject("monitor");
        }, "检测完成", false);
    }

    void setPublish(boolean enabled) {
        perform("更新公开设置", () -> Api.put(path(), new JSONObject().put("publish_enabled", enabled)),
                enabled ? "公开状态页已开启" : "公开状态页已关闭", true);
    }

    void removeTarget() {
        String targetPath = path() + "/targets/" + targetId;
        perform("移除服务", () -> Api.delete(targetPath), "服务已移除", false);
    }

    void deleteMonitor() {
        perform("删除项目", () -> Api.delete(path()), "监控项目已删除", false);
    }

    private String path() {
        return "/api/monitors/" + monitorId;
    }

    private void perform(String action, Callable<JSONObject> request, String success, boolean reload) {
        if (busy || cleared || deleted) return;
        int requestId = ++generation;
        busy = true;
        loading = false;
        operation = action;
        error = "";
        formError = "";
        changed();
        Api.async(request, updated -> {
            if (cleared || requestId != generation) return;
            busy = false;
            deleted = "删除项目".equals(action);
            if (!deleted) monitor = updated;
            form = "";
            notice = success;
            changed();
            if (reload && !deleted) load();
        }, failure -> {
            if (cleared || requestId != generation) return;
            busy = false;
            String message = failure.getMessage() == null ? "请求失败，请重试" : failure.getMessage();
            if ("添加服务".equals(action)) message += "。若网络中断，请先刷新核对，避免重复添加。";
            if (form.isEmpty()) error = message;
            else formError = message;
            changed();
        });
    }

    @Override protected void onCleared() {
        cleared = true;
        generation++;
    }
}
