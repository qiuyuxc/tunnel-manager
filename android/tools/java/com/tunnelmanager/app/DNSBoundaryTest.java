package com.tunnelmanager.app;

import android.content.Context;
import android.content.SharedPreferences;
import android.os.Looper;
import org.json.JSONObject;
import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.net.URLConnection;
import java.net.URLStreamHandler;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Collections;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicReference;

public final class DNSBoundaryTest {
    private static final Preferences prefs = new Preferences();
    private static final Context context = new Context() {
        public SharedPreferences getSharedPreferences(String name, int mode) { return prefs; }
    };
    private static final List<Connection> requests = new ArrayList<>();
    private static Runnable opened = () -> {};
    private static Runnable responded = () -> {};
    private static Runnable writing = () -> {};
    private static int status = 200;

    public static void main(String[] args) throws Exception {
        URL.setURLStreamHandlerFactory(protocol -> new URLStreamHandler() {
            protected URLConnection openConnection(URL url) {
                opened.run();
                return new Connection(url);
            }
        });
        Session.init(context);
        login("old");
        switch (args[0]) {
            case "api-snapshot":
                RequestScope bound = new RequestScope(Session::snapshot);
                writing = () -> login("new");
                Api.post(bound, "/dns-test", new JSONObject());
                check(requests.size() == 1, "one in-flight request");
                check(requests.get(0).getURL().getHost().equals("old.invalid"), "original server");
                check(requests.get(0).getRequestProperty("X-Auth-Token").equals("old-token"),
                        "must not mix old server with new token");
                break;
            case "api-stale-401":
                int[] notices = {0};
                Api.watchSession(() -> notices[0]++);
                status = 401;
                expectUnauthorized();
                login("new");
                Looper.drain();
                check(notices[0] == 0, "queued old 401 must not invalidate new session");
                expectUnauthorized();
                Looper.drain();
                check(notices[0] == 1, "current 401 still invalidates current session");
                break;
            case "api-old-response-401":
                int[] oldNotices = {0};
                Api.watchSession(() -> oldNotices[0]++);
                status = 401;
                responded = () -> login("new");
                expectUnauthorized();
                Looper.drain();
                check(oldNotices[0] == 0, "401 received after switch must not invalidate new session");
                break;
            case "api-unsent":
                RequestScope unsent = new RequestScope(Session::snapshot);
                opened = () -> login("new");
                List<DNSValues.Result> stopped = send(unsent);
                check(requests.isEmpty(), "final preflight must stop request before output stream opens");
                check(stopped.stream().allMatch(result -> result.notSent && !result.uncertain), "preflight is unsent, never unknown");
                break;
            case "api-inflight-switch":
            case "api-inflight-destroy":
            case "api-send-race":
                concurrentBoundary(args[0]);
                break;
            case "session-aba":
                RequestScope previous = new RequestScope(Session::snapshot);
                prefs.edit().putString("role", "user").apply();
                previous.check();
                prefs.edit().putString("token", "replacement").apply();
                prefs.edit().putString("token", "old-token").apply();
                check(previous.reason() != null, "A-B-A token changes must invalidate original batch");
                RequestScope beforeLogout = new RequestScope(Session::snapshot);
                Session.logout(context);
                login("old");
                check(beforeLogout.reason() != null, "logout and identical login must not revive batch");
                RequestScope beforeUser = new RequestScope(Session::snapshot);
                prefs.edit().putString("username", "other").apply();
                check(beforeUser.reason() != null, "user switch invalidates batch even with same token and server");
                RequestScope beforeServer = new RequestScope(Session::snapshot);
                prefs.edit().putString("server", "https://other.invalid").apply();
                check(beforeServer.reason() != null, "server switch invalidates batch even with same token");
                break;
            case "session-snapshot-atomic":
                atomicSnapshots();
                break;
            case "api-compatible":
                Api.post("/dns-test", new JSONObject());
                check(requests.size() == 1 && requests.get(0).getRequestProperty("X-Auth-Token").equals("old-token"),
                        "ordinary post signature and authentication preserved");
                break;
            default: throw new AssertionError(args[0]);
        }
    }

    private static void expectUnauthorized() throws Exception {
        try { Api.post(new RequestScope(Session::snapshot), "/dns-test", new JSONObject()); throw new AssertionError("expected 401"); }
        catch (Api.Failure failure) { check(failure.code == 401, "401 preserved"); }
    }

    private static void login(String identity) {
        Session.save(context, "https://" + identity + ".invalid", identity + "-token", identity, "admin");
    }

    private static List<DNSValues.Result> send(RequestScope scope) {
        return DNSValues.send(DNSValues.parse("A", "192.0.2.1\n192.0.2.2\n192.0.2.3", Collections.emptyList()).pending(), row -> {
            scope.check();
            Api.post(scope, "/dns-test", new JSONObject());
        }, result -> {});
    }

    private static void concurrentBoundary(String scenario) throws Exception {
        RequestScope scope = new RequestScope(Session::snapshot);
        CountDownLatch reached = new CountDownLatch(1), release = new CountDownLatch(1);
        Runnable pause = () -> { reached.countDown(); await(release); };
        if (scenario.equals("api-send-race")) opened = pause; else responded = pause;
        AtomicReference<List<DNSValues.Result>> completed = new AtomicReference<>();
        Thread worker = new Thread(() -> completed.set(send(scope)));
        worker.setDaemon(true);
        worker.start();
        try {
            await(reached);
            if (scenario.equals("api-inflight-destroy")) scope.stop("宿主已销毁");
            else login("new");
        } finally {
            release.countDown();
        }
        worker.join(5000);
        check(!worker.isAlive(), "batch drains without deadlocking session change or destruction");
        List<DNSValues.Result> results = completed.get();
        check(results != null && results.size() == 3, "all row outcomes collected");
        if (scenario.equals("api-send-race")) {
            check(requests.isEmpty(), "no dispatch when session changes during request preparation");
            check(results.get(0).notSent && !results.get(0).uncertain, "prepared but unsent row is not unknown");
        } else {
            check(requests.size() == 1 && results.get(0).success, "in-flight success collected after stop");
            check(requests.get(0).getURL().getHost().equals("old.invalid")
                    && requests.get(0).getRequestProperty("X-Auth-Token").equals("old-token"), "in-flight credentials remain paired");
        }
        check(results.get(1).notSent && !results.get(1).uncertain && results.get(2).notSent, "remaining rows are unsent");
    }

    private static void atomicSnapshots() throws Exception {
        CountDownLatch start = new CountDownLatch(1);
        Thread writer = new Thread(() -> {
            await(start);
            for (int index = 0; index < 2000; index++) {
                String identity = index % 2 == 0 ? "new" : "old";
                prefs.edit().putString("server", "https://" + identity + ".invalid")
                        .putString("token", identity + "-token").putString("username", identity).apply();
            }
        });
        writer.setDaemon(true);
        writer.start();
        start.countDown();
        for (int index = 0; index < 2000; index++) {
            RequestScope.Snapshot snapshot = Session.snapshot();
            check(snapshot.server.equals("https://" + snapshot.username + ".invalid")
                    && snapshot.token.equals(snapshot.username + "-token"), "snapshot cannot mix concurrent credentials");
        }
        writer.join(5000);
        check(!writer.isAlive(), "preference listener and snapshot have no lock inversion");
    }

    private static void await(CountDownLatch latch) {
        try { check(latch.await(5, TimeUnit.SECONDS), "latch timeout"); }
        catch (InterruptedException failure) { Thread.currentThread().interrupt(); throw new AssertionError(failure); }
    }

    private static void check(boolean value, String message) {
        if (!value) throw new AssertionError(message);
    }

    private static final class Connection extends HttpURLConnection {
        Connection(URL url) { super(url); }
        public void connect() {}
        public void disconnect() {}
        public boolean usingProxy() { return false; }
        public OutputStream getOutputStream() {
            requests.add(this);
            writing.run();
            return new ByteArrayOutputStream();
        }
        public int getResponseCode() { responded.run(); return status; }
        public InputStream getInputStream() { return new ByteArrayInputStream("{}".getBytes()); }
        public InputStream getErrorStream() { return getInputStream(); }
    }

    private static final class Preferences implements SharedPreferences {
        private final Map<String, String> values = new HashMap<>();
        private final List<OnSharedPreferenceChangeListener> listeners = new ArrayList<>();
        public synchronized Map<String, ?> getAll() { return new HashMap<>(values); }
        public synchronized String getString(String key, String fallback) { return values.getOrDefault(key, fallback); }
        public synchronized void registerOnSharedPreferenceChangeListener(OnSharedPreferenceChangeListener listener) { listeners.add(listener); }
        public synchronized void unregisterOnSharedPreferenceChangeListener(OnSharedPreferenceChangeListener listener) { listeners.remove(listener); }
        public Editor edit() {
            return new Editor() {
                private final Map<String, String> updates = new HashMap<>();
                public Editor putString(String key, String value) { updates.put(key, value); return this; }
                public Editor remove(String key) { updates.put(key, null); return this; }
                public void apply() {
                    List<String> changed = new ArrayList<>();
                    List<OnSharedPreferenceChangeListener> callbacks;
                    synchronized (Preferences.this) {
                        updates.forEach((key, value) -> {
                            if (!Objects.equals(values.get(key), value)) changed.add(key);
                            if (value == null) values.remove(key); else values.put(key, value);
                        });
                        callbacks = new ArrayList<>(listeners);
                    }
                    for (String key : changed) for (OnSharedPreferenceChangeListener listener : callbacks) {
                        listener.onSharedPreferenceChanged(Preferences.this, key);
                    }
                }
            };
        }
    }
}
