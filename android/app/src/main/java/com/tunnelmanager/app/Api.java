package com.tunnelmanager.app;

import android.os.Handler;
import android.os.Looper;

import org.json.JSONArray;
import org.json.JSONObject;
import org.json.JSONTokener;

import java.io.ByteArrayOutputStream;
import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Callable;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.function.Consumer;

/**
 * The console API, one method per verb.
 *
 * Requests are synchronous; every caller goes through {@link #async} so nothing
 * touches the network on the UI thread. Failures carry the server's own message
 * where it has one, translated in the same places MainActivity already does.
 */
final class Api {

    private static final ExecutorService POOL = Executors.newFixedThreadPool(4);
    private static final Handler MAIN = new Handler(Looper.getMainLooper());
    private static final AtomicBoolean sessionLostFired = new AtomicBoolean();
    /** Set by the console shell while it is on screen. */
    private static volatile Runnable sessionLost;

    private Api() {
    }

    /** Registers the console's "the token is dead" handler. */
    static void watchSession(Runnable listener) {
        sessionLost = listener;
        sessionLostFired.set(false);
    }

    static void stopWatchingSession() {
        sessionLost = null;
    }

    /**
     * Reports a rejected token to the console shell, once.
     *
     * Every page can be in flight when a session dies, and the shell's answer is
     * to leave for the sign-in screen. Firing that twice is what made the app
     * bounce between the two, so the first rejection wins and the rest are
     * ignored until a new session is established.
     */
    private static void notifySessionLost() {
        Runnable listener = sessionLost;
        if (listener == null || !sessionLostFired.compareAndSet(false, true)) return;
        MAIN.post(listener);
    }

    private static void notifySessionLost(RequestScope.Snapshot expected) {
        Runnable listener = sessionLost;
        if (listener == null || !expected.matches(Session.snapshot())) return;
        MAIN.post(() -> {
            if (listener != sessionLost || !expected.matches(Session.snapshot())) return;
            if (sessionLostFired.compareAndSet(false, true)) listener.run();
        });
    }

    /** A request that failed. {@link #auth} means the session is gone. */
    static class Failure extends Exception {
        final int code;
        final boolean auth;

        Failure(int code, String message) {
            super(message);
            this.code = code;
            this.auth = code == 401 || code == 403;
        }
    }

    static JSONObject get(String path) throws Failure {
        return object(request("GET", path, null));
    }

    static JSONArray getArray(String path) throws Failure {
        Object value = parse(request("GET", path, null));
        if (value instanceof JSONArray) return (JSONArray) value;
        throw new Failure(0, I18n.text("m_d7c40bab50f2"));
    }

    static JSONObject post(String path, JSONObject body) throws Failure {
        return object(request("POST", path, body));
    }

    static JSONObject post(RequestScope scope, String path, JSONObject body) throws Failure {
        return object(request("POST", path, body, scope));
    }

    static JSONObject put(String path, JSONObject body) throws Failure {
        return object(request("PUT", path, body));
    }

    static JSONObject streamAssistant(RequestScope scope, String path, JSONObject body, Consumer<JSONObject> onEvent) throws Failure {
        scope.check();
        HttpURLConnection conn = null;
        try {
            conn = (HttpURLConnection) new URL(scope.session.server + path).openConnection();
            conn.setRequestMethod("POST"); conn.setConnectTimeout(10000); conn.setReadTimeout(180000);
            conn.setRequestProperty("Accept", "text/event-stream");
            conn.setRequestProperty("Accept-Language", I18n.languageTag());
            conn.setRequestProperty("Content-Type", "application/json; charset=utf-8");
            conn.setRequestProperty("X-Auth-Token", scope.session.token);
            conn.setDoOutput(true);
            byte[] payload = body.toString().getBytes(StandardCharsets.UTF_8);
            conn.setFixedLengthStreamingMode(payload.length);
            scope.check();
            try (OutputStream out = conn.getOutputStream()) { out.write(payload); }
            int code = conn.getResponseCode();
            if (code == 401) notifySessionLost(scope.session);
            if (code >= 400) throw new Failure(code, serverMessage(readAll(conn.getErrorStream()), code));
            if (conn.getContentType() == null || !conn.getContentType().contains("text/event-stream")) return object(readAll(conn.getInputStream()));
            try (BufferedReader reader = new BufferedReader(new InputStreamReader(conn.getInputStream(), StandardCharsets.UTF_8))) {
                String line, event = "message";
                StringBuilder data = new StringBuilder();
                int total = 0;
                while ((line = reader.readLine()) != null) {
                    scope.check();
                    total += line.length();
                    if (total > 4 * 1024 * 1024) throw new Failure(0, I18n.text("m_21d5010ed46d"));
                    if (line.isEmpty()) {
                        if (data.length() == 0) { event = "message"; continue; }
                        JSONObject value = new JSONObject(data.toString());
                        if ("error".equals(event)) throw new Failure(0, value.optString("error", I18n.text("m_b7b96f677d8e")));
                        if ("done".equals(event)) return value;
                        value.put("event", event);
                        MAIN.post(() -> { if (scope.session.matches(Session.snapshot())) onEvent.accept(value); });
                        data.setLength(0); event = "message";
                    } else if (line.startsWith("event:")) event = line.substring(6).trim();
                    else if (line.startsWith("data:")) { String value = line.substring(5); if (value.startsWith(" ")) value = value.substring(1); data.append(value).append('\n'); }
                }
            }
            throw new Failure(0, I18n.text("m_622be08aa01d"));
        } catch (Failure e) { throw e; }
        catch (Exception e) { throw new Failure(0, networkMessage(e)); }
        finally { if (conn != null) conn.disconnect(); }
    }

    static JSONObject delete(String path) throws Failure {
        return object(request("DELETE", path, null));
    }

    /** DELETE with a body — the ingress endpoint needs to name what it removes. */
    static JSONObject delete(String path, JSONObject body) throws Failure {
        return object(request("DELETE", path, body));
    }

    private static JSONObject object(String raw) throws Failure {
        Object value = parse(raw);
        if (value instanceof JSONObject) return (JSONObject) value;
        throw new Failure(0, I18n.text("m_d7c40bab50f2"));
    }

    /**
     * Uploads one file as {@code multipart/form-data}.
     *
     * The console's upload endpoints take a single "file" part rather than a
     * JSON body, so the two cannot share {@link #request}. The bytes are held in
     * memory because every caller has already read them (picker or camera) and
     * the server caps them at 4 MiB.
     */
    static JSONObject postFile(String path, byte[] data, String filename, String mime) throws Failure {
        String base = Session.server();
        if (base.isEmpty()) throw new Failure(0, I18n.text("m_104745b10ab6"));
        String boundary = "----tm" + System.nanoTime();
        HttpURLConnection conn = null;
        try {
            conn = (HttpURLConnection) new URL(base + path).openConnection();
            conn.setRequestMethod("POST");
            conn.setConnectTimeout(10000);
            conn.setReadTimeout(60000);
            conn.setDoOutput(true);
            conn.setRequestProperty("Accept", "application/json");
            conn.setRequestProperty("Content-Type", "multipart/form-data; boundary=" + boundary);
            conn.setRequestProperty("Accept-Language", I18n.languageTag());
            String token = Session.token();
            if (!token.isEmpty()) conn.setRequestProperty("X-Auth-Token", token);

            String safeName = filename == null || filename.isEmpty() ? "avatar" : filename;
            String safeMime = mime == null || mime.isEmpty() ? "application/octet-stream" : mime;
            ByteArrayOutputStream payload = new ByteArrayOutputStream();
            payload.write(("--" + boundary + "\r\n").getBytes(StandardCharsets.UTF_8));
            payload.write(("Content-Disposition: form-data; name=\"filename\"\r\n\r\n" + safeName + "\r\n")
                    .getBytes(StandardCharsets.UTF_8));
            payload.write(("--" + boundary + "\r\n").getBytes(StandardCharsets.UTF_8));
            payload.write(("Content-Disposition: form-data; name=\"file\"; filename=\"" + safeName + "\"\r\n")
                    .getBytes(StandardCharsets.UTF_8));
            payload.write(("Content-Type: " + safeMime + "\r\n\r\n").getBytes(StandardCharsets.UTF_8));
            payload.write(data);
            payload.write(("\r\n--" + boundary + "--\r\n").getBytes(StandardCharsets.UTF_8));

            byte[] body = payload.toByteArray();
            conn.setFixedLengthStreamingMode(body.length);
            OutputStream out = conn.getOutputStream();
            try {
                out.write(body);
            } finally {
                out.close();
            }

            int code = conn.getResponseCode();
            InputStream in = code >= 400 ? conn.getErrorStream() : conn.getInputStream();
            String raw = readAll(in);
            if (code == 401 && !token.isEmpty()) notifySessionLost();
            if (code >= 400) throw new Failure(code, serverMessage(raw, code));
            return object(raw);
        } catch (Failure e) {
            throw e;
        } catch (Exception e) {
            throw new Failure(0, networkMessage(e));
        } finally {
            if (conn != null) conn.disconnect();
        }
    }

    /** Fetches a same-server asset (an uploaded avatar or icon) as raw bytes. */
    static byte[] asset(String path) throws Failure {
        String base = Session.server();
        if (base.isEmpty()) throw new Failure(0, I18n.text("m_104745b10ab6"));
        // Branding can point at a CDN rather than this server; only a relative
        // path needs the console's address in front of it.
        String url = path.startsWith("http://") || path.startsWith("https://") ? path : base + path;
        HttpURLConnection conn = null;
        try {
            conn = (HttpURLConnection) new URL(url).openConnection();
            conn.setRequestMethod("GET");
            conn.setConnectTimeout(10000);
            conn.setReadTimeout(20000);
            conn.setRequestProperty("Accept", "image/*");
            String token = Session.token();
            if (!token.isEmpty()) conn.setRequestProperty("X-Auth-Token", token);
            int code = conn.getResponseCode();
            if (code >= 400) throw new Failure(code, I18n.text("m_572993f5820a"));
            return readBytes(conn.getInputStream());
        } catch (Failure e) {
            throw e;
        } catch (Exception e) {
            throw new Failure(0, networkMessage(e));
        } finally {
            if (conn != null) conn.disconnect();
        }
    }

    private static byte[] readBytes(InputStream in) throws Exception {
        if (in == null) return new byte[0];
        try {
            ByteArrayOutputStream buffer = new ByteArrayOutputStream();
            byte[] chunk = new byte[8192];
            int read;
            while ((read = in.read(chunk)) != -1) buffer.write(chunk, 0, read);
            return buffer.toByteArray();
        } finally {
            in.close();
        }
    }

    /**
     * A GET against an absolute URL, outside the console session.
     *
     * Used for the GitHub release feed on the about page, which has nothing to
     * do with the configured server and must not carry its token.
     */
    static String external(String url, String accept) throws Failure {
        HttpURLConnection conn = null;
        try {
            conn = (HttpURLConnection) new URL(url).openConnection();
            conn.setRequestMethod("GET");
            conn.setConnectTimeout(10000);
            conn.setReadTimeout(20000);
            if (accept != null && !accept.isEmpty()) conn.setRequestProperty("Accept", accept);
            conn.setRequestProperty("User-Agent", "TunnelManager-Android");
            int code = conn.getResponseCode();
            InputStream in = code >= 400 ? conn.getErrorStream() : conn.getInputStream();
            String raw = readAll(in);
            if (code >= 400) throw new Failure(code, I18n.text("m_99841da5202b") + code + "）");
            return raw;
        } catch (Failure e) {
            throw e;
        } catch (Exception e) {
            throw new Failure(0, networkMessage(e));
        } finally {
            if (conn != null) conn.disconnect();
        }
    }

    private static Object parse(String raw) throws Failure {
        if (raw == null || raw.isEmpty()) return new JSONObject();
        try {
            return new JSONTokener(raw).nextValue();
        } catch (Exception e) {
            throw new Failure(0, I18n.text("m_9a8a0b39f334"));
        }
    }

    private static String request(String method, String path, JSONObject body) throws Failure {
        return request(method, path, body, null);
    }

    private static String request(String method, String path, JSONObject body, RequestScope scope) throws Failure {
        if (scope != null) scope.check();
        String base = scope == null ? Session.server() : scope.session.server;
        if (base.isEmpty()) throw new Failure(0, I18n.text("m_104745b10ab6"));
        HttpURLConnection conn = null;
        try {
            conn = (HttpURLConnection) new URL(base + path).openConnection();
            conn.setRequestMethod(method);
            conn.setConnectTimeout(10000);
            conn.setReadTimeout(path.startsWith("/api/assistant/") ? 180000 : 25000);
            conn.setRequestProperty("Accept-Language", I18n.languageTag());
            conn.setRequestProperty("Accept", "application/json");
            String token = scope == null ? Session.token() : scope.session.token;
            if (!token.isEmpty()) conn.setRequestProperty("X-Auth-Token", token);

            if (body != null) {
                conn.setDoOutput(true);
                conn.setRequestProperty("Content-Type", "application/json; charset=utf-8");
                byte[] payload = body.toString().getBytes(StandardCharsets.UTF_8);
                conn.setFixedLengthStreamingMode(payload.length);
                if (scope != null) scope.check();
                OutputStream out = conn.getOutputStream();
                try {
                    out.write(payload);
                } finally {
                    out.close();
                }
            }

            if (body == null && scope != null) scope.check();
            int code = conn.getResponseCode();
            InputStream in = code >= 400 ? conn.getErrorStream() : conn.getInputStream();
            String raw = readAll(in);
            if (code == 401 && !token.isEmpty()) {
                if (scope == null) notifySessionLost();
                else notifySessionLost(scope.session);
            }
            if (code >= 400) throw new Failure(code, serverMessage(raw, code));
            return raw;
        } catch (Failure e) {
            throw e;
        } catch (RequestScope.Stopped stopped) {
            throw stopped;
        } catch (Exception e) {
            throw new Failure(0, networkMessage(e));
        } finally {
            if (conn != null) conn.disconnect();
        }
    }

    /** The API answers in English; show the operator something readable. */
    private static String serverMessage(String raw, int code) {
        String message = "";
        try {
            message = new JSONObject(raw).optString("error", "");
        } catch (Exception ignored) {
        }
        if (message.contains("invalid credentials")) return I18n.text("m_a9a6a83a7d22");
        if (message.contains("unauthorized")) return I18n.text("m_aa2943a75de2");
        if (message.contains("forbidden") || message.contains("permission")) return I18n.text("m_b7fcc405e49c");
        if (message.contains("not found")) return I18n.text("m_158780e04042");
        if (message.contains("required")) return I18n.text("m_760e540a81f9");
        if (!message.isEmpty()) return message;
        return I18n.text("m_99841da5202b") + code + "）";
    }

    private static String networkMessage(Exception e) {
        String message = e.getMessage() == null ? "" : e.getMessage();
        if (message.contains("ECONNREFUSED") || message.contains("Failed to connect")) {
            return I18n.text("m_ff8da88893d7");
        }
        if (message.contains("timed out") || message.contains("timeout")) return I18n.text("m_a3fceccd6229");
        if (message.contains("Unable to resolve host")) return I18n.text("m_417a14c8aae9");
        if (message.contains("CLEARTEXT")) return I18n.text("m_27d0f8469bb1");
        return I18n.text("m_4a5f658ed9ae") + (message.isEmpty() ? e.getClass().getSimpleName() : message);
    }

    private static String readAll(InputStream in) throws Exception {
        if (in == null) return "";
        try {
            ByteArrayOutputStream buffer = new ByteArrayOutputStream();
            byte[] chunk = new byte[4096];
            int read;
            while ((read = in.read(chunk)) != -1) buffer.write(chunk, 0, read);
            return new String(buffer.toByteArray(), StandardCharsets.UTF_8);
        } finally {
            in.close();
        }
    }

    // -------------------------------------------------------------- threading

    /** Runs work off the UI thread and delivers exactly one callback back on it. */
    static <T> void async(Callable<T> work, Consumer<T> onOk, Consumer<Failure> onErr) {
        POOL.execute(() -> {
            try {
                T value = work.call();
                MAIN.post(() -> onOk.accept(value));
            } catch (Failure f) {
                MAIN.post(() -> onErr.accept(f));
            } catch (Exception e) {
                MAIN.post(() -> onErr.accept(new Failure(0, networkMessage(e))));
            }
        });
    }
}
