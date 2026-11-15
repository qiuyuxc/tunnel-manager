package com.tunnelmanager.app;

final class WebOriginTest {
    public static void main(String[] args) {
        String server = "https://panel.example.com";
        for (String url : new String[]{server, server + "/account?code=1", "https://PANEL.example.com:443/account"}) {
            if (!WebOrigin.trusted(server, url)) throw new AssertionError("rejected " + url);
        }
        for (String url : new String[]{"http://panel.example.com", "https://panel.example.com:8443", "https://panel.example.com.evil.test", "https://evil.test/?host=panel.example.com", "https://panel.example.com@evil.test", "https://user@panel.example.com", "file:///panel.example.com", "javascript:alert(1)", "data:text/html,hi", "about:blank", null}) {
            if (WebOrigin.trusted(server, url)) throw new AssertionError("trusted " + url);
        }
        if (!WebOrigin.trusted("http://[::1]:8080", "http://[::1]:8080/account")) throw new AssertionError("IPv6 origin");
        if (WebOrigin.trusted("", "")) throw new AssertionError("empty origin");
    }
}
