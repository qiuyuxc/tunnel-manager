package com.tunnelmanager.app;

import java.util.Locale;

final class TunnelFilter {
    enum State {
        HEALTHY("m_296de0e31f8c", "ok"), DEGRADED("m_9f22b41cf5d7", "warn"), DOWN("m_be1b4f3c6c1c", "down"),
        INACTIVE("m_3d52574ce150", ""), UNKNOWN("m_4d8c1c5b4283", "");

        final String labelKey;
        String label() { return I18n.text(labelKey); }
        final String style;

        State(String label, String style) {
            this.labelKey = label;
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
