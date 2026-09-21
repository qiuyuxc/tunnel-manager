package com.tunnelmanager.app;

import java.util.Locale;

final class TunnelFilter {
    enum State {
        HEALTHY("正常", "ok"), DEGRADED("降级", "warn"), DOWN("离线", "down"),
        INACTIVE("未连接", ""), UNKNOWN("未知", "");

        final String label;
        final String style;

        State(String label, String style) {
            this.label = label;
            this.style = style;
        }
    }

    private TunnelFilter() {}

    static State state(String status) {
        switch (normalized(status)) {
            case "healthy": return State.HEALTHY;
            case "degraded": return State.DEGRADED;
            case "down": return State.DOWN;
            case "inactive": return State.INACTIVE;
            default: return State.UNKNOWN;
        }
    }

    static boolean matches(String name, String id, String status, String query, int selected) {
        if (selected < 0 || selected > State.values().length) return false;
        if (selected > 0 && state(status).ordinal() != selected - 1) return false;
        String needle = normalized(query);
        return normalized(name).contains(needle) || normalized(id).contains(needle);
    }

    private static String normalized(String value) {
        return value == null ? "" : value.trim().toLowerCase(Locale.ROOT);
    }
}
