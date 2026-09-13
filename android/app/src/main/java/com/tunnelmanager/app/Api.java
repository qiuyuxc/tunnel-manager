package com.tunnelmanager.app;

import android.os.Handler;
import android.os.Looper;

import org.json.JSONArray;
import org.json.JSONObject;
import org.json.JSONTokener;

import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Callable;
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

    private Api() {
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
        throw new Failure(0, "响应格式不正确");
    }

    static JSONObject post(String path, JSONObject body) throws Failure {
        return object(request("POST", path, body));
    }

    static JSONObject put(String path, JSONObject body) throws Failure {
        return object(request("PUT", path, body));
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
        throw new Failure(0, "响应格式不正确");
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
        if (base.isEmpty()) throw new Failure(0, "未配置服务器地址");
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
        if (base.isEmpty()) throw new Failure(0, "未配置服务器地址");
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
            if (code >= 400) throw new Failure(code, "图片加载失败");
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
            if (code >= 400) throw new Failure(code, "请求失败（HTTP " + code + "）");
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
            throw new Failure(0, "响应解析失败");
        }
    }

    private static String request(String method, String path, JSONObject body) throws Failure {
        String base = Session.server();
        if (base.isEmpty()) throw new Failure(0, "未配置服务器地址");
        HttpURLConnection conn = null;
        try {
            conn = (HttpURLConnection) new URL(base + path).openConnection();
            conn.setRequestMethod(method);
            conn.setConnectTimeout(10000);
            conn.setReadTimeout(25000);
            conn.setRequestProperty("Accept", "application/json");
            String token = Session.token();
            if (!token.isEmpty()) conn.setRequestProperty("X-Auth-Token", token);

            if (body != null) {
                conn.setDoOutput(true);
                conn.setRequestProperty("Content-Type", "application/json; charset=utf-8");
                byte[] payload = body.toString().getBytes(StandardCharsets.UTF_8);
                conn.setFixedLengthStreamingMode(payload.length);
                OutputStream out = conn.getOutputStream();
                try {
                    out.write(payload);
                } finally {
                    out.close();
                }
            }

            int code = conn.getResponseCode();
            InputStream in = code >= 400 ? conn.getErrorStream() : conn.getInputStream();
            String raw = readAll(in);
            if (code >= 400) throw new Failure(code, serverMessage(raw, code));
            return raw;
        } catch (Failure e) {
            throw e;
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
        if (message.contains("invalid credentials")) return "用户名或密码错误";
        if (message.contains("unauthorized")) return "登录状态已失效，请重新登录";
        if (message.contains("forbidden") || message.contains("permission")) return "没有权限执行该操作";
        if (message.contains("not found")) return "对象不存在或已被删除";
        if (message.contains("required")) return "请填写必填项";
        if (!message.isEmpty()) return message;
        return "请求失败（HTTP " + code + "）";
    }

    private static String networkMessage(Exception e) {
        String message = e.getMessage() == null ? "" : e.getMessage();
        if (message.contains("ECONNREFUSED") || message.contains("Failed to connect")) {
            return "连不上服务器，请检查地址和端口";
        }
        if (message.contains("timed out") || message.contains("timeout")) return "连接超时，请检查网络";
        if (message.contains("Unable to resolve host")) return "域名解析失败，请检查地址";
        if (message.contains("CLEARTEXT")) return "该地址不允许明文 HTTP 访问";
        return "网络错误：" + (message.isEmpty() ? e.getClass().getSimpleName() : message);
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
