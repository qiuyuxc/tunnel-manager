package com.tunnelmanager.app;

/** Small, platform-independent boundaries for authenticated HTTP and uploads. */
final class NativeRequestPolicy {
    static final int MAX_RESPONSE_BYTES = 8 * 1024 * 1024;

    static String authenticatedURL(String server, String path) {
        if (path == null || !path.startsWith("/") || path.startsWith("//") || path.indexOf('\\') >= 0
                || !WebOrigin.trusted(server, server + path)) {
            throw new IllegalArgumentException("Invalid panel request origin");
        }
        return server + path;
    }

    static String filename(String value) {
        String clean = value == null ? "" : value.replaceAll("[\\\\/\"\r\n]", "_");
        return clean.isEmpty() ? "avatar" : clean.substring(0, Math.min(clean.length(), 120));
    }

    static String mime(String value) {
        return value != null && value.matches("[a-zA-Z0-9!#$&^_.+-]+/[a-zA-Z0-9!#$&^_.+-]+")
                ? value : "application/octet-stream";
    }
}
