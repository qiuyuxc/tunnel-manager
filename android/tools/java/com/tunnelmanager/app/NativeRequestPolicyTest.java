package com.tunnelmanager.app;

final class NativeRequestPolicyTest {
    public static void main(String[] args) {
        String server = "https://panel.example";
        if (!NativeRequestPolicy.authenticatedURL(server, "/api/tunnels?name=a").equals(server + "/api/tunnels?name=a")) throw new AssertionError("valid path");
        for (String path : new String[]{"https://evil.test/api", "//evil.test/api", "\\evil.test", "api/tunnels", null}) {
            try { NativeRequestPolicy.authenticatedURL(server, path); throw new AssertionError("accepted path " + path); }
            catch (IllegalArgumentException expected) { }
        }
        try { NativeRequestPolicy.authenticatedURL("https://user@panel.example", "/api/me"); throw new AssertionError("accepted credentials"); }
        catch (IllegalArgumentException expected) { }
        if (!NativeRequestPolicy.filename("path/ava\"tar\r\nX-Auth-Token: injected.png").equals("path_ava_tar__X-Auth-Token: injected.png")) throw new AssertionError("multipart injection");
        if (!NativeRequestPolicy.filename(null).equals("avatar")) throw new AssertionError("filename fallback");
        if (NativeRequestPolicy.filename("a".repeat(1000)).length() != 120) throw new AssertionError("filename cap");
        if (!NativeRequestPolicy.mime("image/png").equals("image/png")) throw new AssertionError("image MIME");
        if (!NativeRequestPolicy.mime("image/png\r\nX-Auth-Token: leak").equals("application/octet-stream")) throw new AssertionError("MIME injection");
    }
}
