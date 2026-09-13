package com.tunnelmanager.app;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.app.Service;
import android.content.Context;
import android.content.Intent;
import android.content.SharedPreferences;
import android.content.pm.ServiceInfo;
import android.os.Build;
import android.os.IBinder;

import org.json.JSONArray;
import org.json.JSONObject;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Set;

/**
 * Polls the console's alert feed and raises system notifications.
 *
 * The web console can only notify while a tab is alive, which is exactly the
 * wrong guarantee for an uptime alert. This service asks the server what
 * changed instead: /api/alerts hands back the same rows the email channel
 * already produced, so nothing here re-derives state or can disagree with the
 * panel.
 *
 * It is started by hand from the notifications page — never on boot — and runs
 * as a foreground service so Android keeps it around.
 */
public class AlertService extends Service {

    /** Set by the console page toggle; doubles as the UI's source of truth. */
    static final String KEY_ENABLED = "notify_enabled";
    private static final String KEY_CURSOR = "alert_cursor";
    private static final String KEY_SEEN = "alert_seen";

    private static final int ID_SERVICE = 1001;
    private static final int ID_SESSION = 1002;
    private static final int ID_TEST = 1003;

    private static final long POLL_OK = 60_000L;
    private static final long POLL_MAX = 15 * 60_000L;
    private static final int SEEN_CAP = 120;

    private volatile boolean running;
    private Thread worker;
    private SharedPreferences prefs;
    private final Set<String> seen = new LinkedHashSet<>();

    // ---------------------------------------------------------------- lifecycle

    @Override
    public void onCreate() {
        super.onCreate();
        prefs = getSharedPreferences(MainActivity.PREFS, MODE_PRIVATE);
        createChannels(this);
        loadSeen();
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        if (MainActivity.ACTION_STOP_ALERTS.equals(intent == null ? null : intent.getAction())) {
            stopSelf();
            return START_NOT_STICKY;
        }
        startInForeground();
        prefs.edit().putBoolean(KEY_ENABLED, true).apply();
        if (!running) {
            running = true;
            worker = new Thread(this::loop, "tm-alerts");
            worker.setDaemon(true);
            worker.start();
        }
        return START_STICKY;
    }

    @Override
    public void onDestroy() {
        running = false;
        if (worker != null) {
            worker.interrupt();
            worker = null;
        }
        prefs.edit().putBoolean(KEY_ENABLED, false).apply();
        super.onDestroy();
    }

    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }

    private void startInForeground() {
        Notification.Builder b = new Notification.Builder(this, MainActivity.CH_SERVICE)
                .setSmallIcon(R.drawable.ic_stat_tm)
                .setContentTitle("告警推送已开启")
                .setContentText("正在等待监控状态变化")
                .setOngoing(true)
                .setShowWhen(false)
                .setContentIntent(openConsole(null));
        if (Build.VERSION.SDK_INT >= 34) {
            // The manifest declares specialUse; passing it explicitly is what
            // Android 14+ actually enforces.
            startForeground(ID_SERVICE, b.build(), ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE);
        } else {
            startForeground(ID_SERVICE, b.build());
        }
    }

    // -------------------------------------------------------------------- poll

    private void loop() {
        long delay = POLL_OK;
        while (running) {
            try {
                pollOnce();
                delay = POLL_OK;
            } catch (SessionGone e) {
                postSessionExpired();
                stopSelf();
                return;
            } catch (Exception e) {
                delay = Math.min(delay * 2, POLL_MAX);
            } catch (Throwable ignored) {
                delay = POLL_MAX;
            }
            try {
                Thread.sleep(delay);
            } catch (InterruptedException e) {
                return;
            }
        }
    }

    private void pollOnce() throws Exception {
        String server = prefs.getString(MainActivity.KEY_SERVER, "");
        String token = prefs.getString(MainActivity.KEY_TOKEN, "");
        if (server.isEmpty() || token.isEmpty()) throw new SessionGone();

        long since = Math.max(0L, cursor() - 1L);
        HttpURLConnection conn = (HttpURLConnection) new URL(server + "/api/alerts?since=" + since).openConnection();
        try {
            conn.setRequestMethod("GET");
            conn.setConnectTimeout(10000);
            conn.setReadTimeout(20000);
            conn.setRequestProperty("Accept", "application/json");
            conn.setRequestProperty("X-Auth-Token", token);
            int code = conn.getResponseCode();
            if (code == 401 || code == 403) throw new SessionGone();
            InputStream in = code >= 400 ? conn.getErrorStream() : conn.getInputStream();
            String body = readAll(in);
            if (code >= 400) throw new IOException("HTTP " + code);

            JSONObject root = new JSONObject(body);
            long next = root.optLong("cursor", cursor());
            JSONArray alerts = root.optJSONArray("alerts");
            for (int i = 0; alerts != null && i < alerts.length(); i++) {
                JSONObject alert = alerts.optJSONObject(i);
                if (alert == null) continue;
                if (!remember(keyOf(alert))) continue;
                postAlert(alert);
            }
            // Re-reading the last second on every poll is what keeps two alerts
            // in the same wall-clock second from being skipped; `seen` is what
            // stops that overlap from notifying twice.
            prefs.edit().putLong(KEY_CURSOR, Math.max(next, cursor())).apply();
            persistSeen(next);
        } finally {
            conn.disconnect();
        }
    }

    private long cursor() {
        return prefs.getLong(KEY_CURSOR, 0L);
    }

    private static String keyOf(JSONObject alert) {
        return alert.optLong("created_at", 0)
                + "|" + alert.optString("monitor_id", "")
                + "|" + alert.optString("target_id", "");
    }

    /** @return true the first time a key is handed over. */
    private boolean remember(String key) {
        synchronized (seen) {
            return seen.add(key);
        }
    }

    private void loadSeen() {
        String raw = prefs.getString(KEY_SEEN, "");
        if (raw.isEmpty()) return;
        synchronized (seen) {
            for (String line : raw.split("\n")) {
                if (!line.isEmpty()) seen.add(line);
            }
        }
    }

    /** Keeps the dedupe set from growing forever without losing the overlap. */
    private void persistSeen(long cursor) {
        List<String> kept = new ArrayList<>();
        synchronized (seen) {
            for (String line : seen) {
                int cut = line.indexOf('|');
                long stamp = cut > 0 ? parseLong(line.substring(0, cut)) : 0L;
                if (stamp >= cursor - 1L) kept.add(line);
            }
            if (kept.size() > SEEN_CAP) {
                kept = kept.subList(kept.size() - SEEN_CAP, kept.size());
            }
            seen.clear();
            seen.addAll(kept);
        }
        prefs.edit().putString(KEY_SEEN, String.join("\n", kept)).apply();
    }

    private static long parseLong(String value) {
        try {
            return Long.parseLong(value);
        } catch (Exception e) {
            return 0L;
        }
    }

    // ----------------------------------------------------------- notifications

    @Override
    public void onTimeout(int startId) {
        // Hook point for the foreground-service budget on newer Androids; the
        // specialUse type we declare is not supposed to be capped, so treat
        // this as a plain stop rather than leaving a half-dead service.
        stopSelf();
    }

    private void postAlert(JSONObject alert) {
        boolean recovered = "ok".equals(alert.optString("state", ""));
        String monitor = alert.optString("monitor_name", "");
        String target = alert.optString("target_name", "");
        if (monitor.isEmpty()) monitor = "监控项目";

        String detail = alert.optString("error", "");
        if (detail.isEmpty()) {
            int httpCode = alert.optInt("http_code", 0);
            if (httpCode > 0) detail = "HTTP " + httpCode;
        }

        String title = recovered ? monitor + " 已恢复" : monitor + " 状态异常";
        String text = target.isEmpty() ? detail : target + (detail.isEmpty() ? "" : " · " + detail);
        if (text.isEmpty()) text = recovered ? "探测已恢复正常" : "探测失败";

        Notification.Builder b = new Notification.Builder(this, recovered ? MainActivity.CH_UP : MainActivity.CH_DOWN)
                .setSmallIcon(R.drawable.ic_stat_tm)
                .setContentTitle(title)
                .setContentText(text)
                .setStyle(new Notification.BigTextStyle().bigText(text))
                .setAutoCancel(true)
                .setContentIntent(openConsole("/monitors/" + alert.optString("monitor_id", "")));

        String extra = alert.optString("detail", "");
        if (!extra.isEmpty() && !extra.equals(text)) {
            b.setSubText(extra);
        }
        // Keyed off the whole alert, not just its timestamp: two targets can
        // change state in the same second and must not overwrite each other.
        notify(keyOf(alert).hashCode(), b.build());
    }

    private void postSessionExpired() {
        Notification n = new Notification.Builder(this, MainActivity.CH_UP)
                .setSmallIcon(R.drawable.ic_stat_tm)
                .setContentTitle("登录已失效，告警推送已停止")
                .setContentText("点按回到 App 重新登录后，再到通知设置里开启")
                .setStyle(new Notification.BigTextStyle().bigText("登录已失效，告警推送已停止。点按回到 App 重新登录后，再到通知设置里开启。"))
                .setAutoCancel(true)
                .setContentIntent(openConsole(null))
                .build();
        notify(ID_SESSION, n);
    }

    /** Posted by the console's "发送测试通知" button, service running or not. */
    static void postTest(Context ctx) {
        createChannels(ctx);
        Notification n = new Notification.Builder(ctx, MainActivity.CH_UP)
                .setSmallIcon(R.drawable.ic_stat_tm)
                .setContentTitle("测试通知")
                .setContentText("这条通知来自本机 App，与浏览器无关")
                .setStyle(new Notification.BigTextStyle().bigText("测试通知：这条通知来自本机 App，与浏览器无关。"))
                .setAutoCancel(true)
                .setContentIntent(openConsole(ctx, null))
                .build();
        NotificationManager nm = ctx.getSystemService(NotificationManager.class);
        if (nm != null) nm.notify(ID_TEST, n);
    }

    private PendingIntent openConsole(String path) {
        return openConsole(this, path);
    }

    private static PendingIntent openConsole(Context ctx, String path) {
        Intent intent = new Intent(ctx, MainActivity.class);
        intent.setFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_SINGLE_TOP);
        if (path != null) intent.putExtra(MainActivity.EXTRA_PATH, path);
        int flags = PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE;
        return PendingIntent.getActivity(ctx, path == null ? 0 : path.hashCode(), intent, flags);
    }

    private void notify(int id, Notification n) {
        NotificationManager nm = getSystemService(NotificationManager.class);
        if (nm != null) nm.notify(id, n);
    }

    private static void createChannels(Context ctx) {
        NotificationManager nm = ctx.getSystemService(NotificationManager.class);
        if (nm == null) return;
        NotificationChannel service = new NotificationChannel(
                MainActivity.CH_SERVICE, ctx.getString(R.string.channel_service), NotificationManager.IMPORTANCE_MIN);
        service.setDescription(ctx.getString(R.string.channel_service_desc));
        service.setShowBadge(false);
        NotificationChannel down = new NotificationChannel(
                MainActivity.CH_DOWN, ctx.getString(R.string.channel_down), NotificationManager.IMPORTANCE_HIGH);
        down.setDescription(ctx.getString(R.string.channel_down_desc));
        NotificationChannel up = new NotificationChannel(
                MainActivity.CH_UP, ctx.getString(R.string.channel_up), NotificationManager.IMPORTANCE_DEFAULT);
        up.setDescription(ctx.getString(R.string.channel_up_desc));
        nm.createNotificationChannel(service);
        nm.createNotificationChannel(down);
        nm.createNotificationChannel(up);
    }

    private static String readAll(InputStream in) throws Exception {
        if (in == null) return "";
        try {
            ByteArrayOutputStream buffer = new ByteArrayOutputStream();
            byte[] chunk = new byte[4096];
            int read;
            while ((read = in.read(chunk)) != -1) {
                buffer.write(chunk, 0, read);
            }
            return new String(buffer.toByteArray(), StandardCharsets.UTF_8);
        } finally {
            in.close();
        }
    }

    /** The stored session no longer works; polling cannot continue. */
    private static class SessionGone extends Exception {
    }
}
