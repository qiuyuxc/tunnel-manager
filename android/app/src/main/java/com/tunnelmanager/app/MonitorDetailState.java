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
            error = failure.getMessage() == null ? I18n.text("m_caa47445ca65") : failure.getMessage();
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
                    formError = I18n.text("m_bf7bdc2c0f03");
                    changed();
                    return;
                }
                if (seconds < 30) {
                    formError = I18n.text("m_2fd839bd0851");
                    changed();
                    return;
                }
                payload.put("interval_sec", seconds);
                perform(I18n.text("m_c8550237ba70"), () -> Api.put(path(), payload), I18n.text("m_a839443abe79"), true);
                return;
            }
            if (name.trim().isEmpty() || url.trim().isEmpty()) {
                formError = I18n.text("m_4625525ea66d");
                changed();
                return;
            }
            payload.put("name", name.trim());
            payload.put("url", url.trim());
            if ("add".equals(form)) {
                payload.put("type", new String[]{"http", "tcp", "icmp"}[type]);
                if (type == 0) payload.put("method", method == 0 ? "GET" : "POST");
                perform(I18n.text("m_6f760cbd09bb"), () -> Api.post(path() + "/targets", payload), I18n.text("m_26b6635cb0cc"), false);
            } else {
                String targetPath = path() + "/targets/" + targetId;
                perform(I18n.text("m_467100deb296"), () -> Api.put(targetPath, payload), I18n.text("m_27a7be5b46ff"), false);
            }
        } catch (org.json.JSONException failure) {
            formError = I18n.text("m_2fb4b8c2ceb4");
            changed();
        }
    }

    void checkNow() {
        perform(I18n.text("m_14ca09ce1fd2"), () -> {
            JSONObject result = Api.post(path() + "/check", new JSONObject());
            return result.getJSONObject("monitor");
        }, I18n.text("m_37888055c912"), false);
    }

    void setPublish(boolean enabled) {
        perform(I18n.text("m_72c075860c03"), () -> Api.put(path(), new JSONObject().put("publish_enabled", enabled)),
                enabled ? I18n.text("m_99d49e881233") : I18n.text("m_3e95219eea69"), true);
    }

    void removeTarget() {
        String targetPath = path() + "/targets/" + targetId;
        perform(I18n.text("m_70f2105700a9"), () -> Api.delete(targetPath), I18n.text("m_dc29c275d718"), false);
    }

    void deleteMonitor() {
        perform(I18n.text("m_f6dc6a110d09"), () -> Api.delete(path()), I18n.text("m_99d16dfc942e"), false);
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
            deleted = I18n.text("m_f6dc6a110d09").equals(action);
            if (!deleted) monitor = updated;
            form = "";
            notice = success;
            changed();
            if (reload && !deleted) load();
        }, failure -> {
            if (cleared || requestId != generation) return;
            busy = false;
            String message = failure.getMessage() == null ? I18n.text("m_d8348d59c4bd") : failure.getMessage();
            if (I18n.text("m_6f760cbd09bb").equals(action)) message += I18n.text("m_af20806c4d4f");
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
