package com.tunnelmanager.app;

import java.util.Locale;

public final class TunnelFilterTest {
    public static void main(String[] args) {
        switch (args[0]) {
            case "status":
                check(TunnelFilter.state("healthy") == TunnelFilter.State.HEALTHY, "Healthy state");
                check(TunnelFilter.state("degraded") == TunnelFilter.State.DEGRADED, "Degraded state");
                check(TunnelFilter.state(" DOWN ") == TunnelFilter.State.DOWN, "Normalized state");
                check(TunnelFilter.state("inactive") == TunnelFilter.State.INACTIVE, "Inactive is not down");
                check(TunnelFilter.state(null) == TunnelFilter.State.UNKNOWN, "Missing state is unknown");
                check(TunnelFilter.state("new-server-state") == TunnelFilter.State.UNKNOWN, "Future state is unknown");
                check("未连接".equals(TunnelFilter.State.INACTIVE.label), "Inactive is not paused");
                check(TunnelFilter.State.UNKNOWN.style.isEmpty(), "Unknown is not an error");
                break;
            case "search":
                check(TunnelFilter.matches("家庭实验室", "AbC-123", "healthy", "实验", 0), "Chinese names");
                check(TunnelFilter.matches("Studio", "AbC-123", "healthy", " abc-123 ", 0), "Case-insensitive IDs");
                check(TunnelFilter.matches("Studio", "id", "healthy", "STUDIO", 0), "Case-insensitive names");
                check(TunnelFilter.matches(null, null, null, null, 0), "Empty query includes unnamed entries");
                check(!TunnelFilter.matches("home", "id", "healthy", "missing", 0), "No false match");
                break;
            case "combined":
                check(TunnelFilter.matches("home", "id", "healthy", "home", 1), "Matching text and state");
                check(!TunnelFilter.matches("home", "id", "down", "home", 1), "State must match");
                check(!TunnelFilter.matches("home", "id", "healthy", "office", 1), "Text must match");
                check(TunnelFilter.matches("home", "id", "degraded", "", 2), "Degraded filter");
                check(TunnelFilter.matches("home", "id", "down", "", 3), "Down filter");
                check(TunnelFilter.matches("home", "id", "inactive", "", 4), "Inactive filter");
                check(TunnelFilter.matches("home", "id", "unexpected", "", 5), "Unknown filter");
                check(!TunnelFilter.matches("home", "id", "healthy", "", -1), "Invalid negative filter");
                check(!TunnelFilter.matches("home", "id", "healthy", "", 6), "Invalid high filter");
                break;
            case "locale":
                Locale previous = Locale.getDefault();
                try {
                    Locale.setDefault(Locale.forLanguageTag("tr-TR"));
                    check(TunnelFilter.matches("INDEX", "ID", "INACTIVE", "index", 4), "Search is independent of device locale");
                } finally {
                    Locale.setDefault(previous);
                }
                break;
            default:
                throw new AssertionError("Unknown case: " + args[0]);
        }
    }

    private static void check(boolean condition, String message) {
        if (!condition) throw new AssertionError(message);
    }
}
