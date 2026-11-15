package com.tunnelmanager.app;

import java.net.URI;
import java.util.Locale;

/** Canonical HTTP origin shared by navigation and every session script. */
final class WebOrigin {
    static String origin(String raw) {
        try {
            URI uri = new URI(raw);
            String scheme = uri.getScheme().toLowerCase(Locale.ROOT);
            String host = uri.getHost();
            if (!(scheme.equals("https") || scheme.equals("http")) || host == null
                    || uri.getRawUserInfo() != null) return "";
            int port = uri.getPort();
            if (port < -1 || port > 65535) return "";
            if (port == (scheme.equals("https") ? 443 : 80)) port = -1;
            return new URI(scheme, null, host.toLowerCase(Locale.ROOT), port, null, null, null).toASCIIString();
        } catch (Exception ignored) { return ""; }
    }

    static boolean trusted(String server, String page) {
        String expected = origin(server);
        return !expected.isEmpty() && expected.equals(origin(page));
    }
}
