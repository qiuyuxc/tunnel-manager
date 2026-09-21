package com.tunnelmanager.app;

import java.util.Arrays;
import java.util.Collections;

public final class MonitorHistoryTest {
    public static void main(String[] args) {
        switch (args[0]) {
            case "empty": {
                MonitorHistory history = new MonitorHistory(Collections.emptyList());
                check(history.samples.isEmpty(), "No synthetic successful samples");
                check(history.firstTime() == 0 && history.lastTime() == 0, "No fake timestamps");
                check(history.maxLatency() == 0, "Empty scale baseline");
                check(!history.connects(0, 60000), "Empty history has no edges");
                check(!history.hasRecentSamples(1000, 1000), "Empty history is not recent");
                break;
            }
            case "ordering": {
                MonitorHistory history = history(sample(3000, "ok", 4), sample(1000, "ok", 2), null,
                        sample(-1, "ok", 2), sample(0, "ok", 2), sample(1000, "down", 0));
                check(history.samples.size() == 2, "Reject invalid times and deduplicate");
                check(history.firstTime() == 1000 && history.lastTime() == 3000, "Order by timestamp");
                check("down".equals(history.samples.get(0).state), "Last duplicate wins");
                try {
                    history.samples.clear();
                    throw new AssertionError("History must be immutable");
                } catch (UnsupportedOperationException expected) {}
                break;
            }
            case "states": {
                MonitorHistory.Counts counts = new MonitorHistory.Counts();
                check("unknown".equals(counts.state()), "Empty projects are not healthy");
                counts.add("ok");
                check("ok".equals(counts.state()), "All known healthy");
                counts.add("future-state");
                counts.add(null);
                check(counts.unknown == 2 && "unknown".equals(counts.state()), "Unknown is not success");
                counts.add("warn");
                check("warn".equals(counts.state()), "Degraded is distinct");
                counts.add("down");
                check("down".equals(counts.state()) && counts.total == 5, "Failures have priority");
                break;
            }
            case "time-axis": {
                MonitorHistory history = history(sample(1000, "ok", 1), sample(2000, "ok", 2), sample(11000, "ok", 3));
                check(history.fraction(0) == 0 && history.fraction(2) == 1, "Use actual endpoints");
                check(Math.abs(history.fraction(1) - 0.1) < 0.000001, "Irregular times are not equally spaced");
                check(history(sample(1000, "ok", 0)).fraction(0) == 0.5, "Single sample is centered");
                break;
            }
            case "gaps": {
                MonitorHistory history = history(sample(1000, "ok", 1), sample(61000, "ok", 2),
                        sample(121000, "down", 0), sample(181000, "ok", 2), sample(241000, "unknown", 0),
                        sample(301000, "ok", 2), sample(901000, "ok", 3), sample(961000, "warn", 3));
                check(history.connects(1, 60000), "Neighboring healthy checks connect");
                for (int index = 2; index < history.samples.size(); index++) {
                    check(!history.connects(index, 60000), "Do not bridge failures, unknowns, gaps or state transitions");
                }
                check(!history.connects(1, 0) && !history.connects(20, 60000), "Invalid connection requests");
                break;
            }
            case "latency": {
                check(sample(1000, "ok", 0).hasLatency(), "Successful sub-millisecond latency can round to zero");
                check(!sample(1000, "down", 5000).hasLatency(), "Timeout is not a successful response");
                check(!sample(1000, "unknown", 1).hasLatency(), "Unknown has no measured response");
                check(!sample(1000, "ok", -1).hasLatency(), "Reject negative latency");
                check(!sample(1000, "ok", Double.NaN).hasLatency(), "Reject missing values");
                check(!sample(1000, "ok", Double.POSITIVE_INFINITY).hasLatency(), "Reject nonfinite values");
                MonitorHistory history = history(sample(1000, "ok", 12), sample(2000, "down", 9999), sample(3000, "warn", 100));
                check(history.maxLatency() == 100, "Scale only measured latency");
                break;
            }
            case "recent": {
                MonitorHistory history = history(sample(1000, "ok", 1), sample(9000, "down", 0), sample(11000, "ok", 1));
                check(history.hasRecentSamples(10000, 1000), "Failure counts as a sample");
                check(!history.hasRecentSamples(10000, 500), "Future samples do not prove recent availability");
                check(!history.hasRecentSamples(100000, 1000), "Old samples cannot prove a current 24h statistic");
                break;
            }
            default: throw new AssertionError("Unknown scenario");
        }
    }

    private static MonitorHistory.Sample sample(long time, String state, double latency) {
        return new MonitorHistory.Sample(time, state, latency);
    }

    private static MonitorHistory history(MonitorHistory.Sample... samples) {
        return new MonitorHistory(Arrays.asList(samples));
    }

    private static void check(boolean condition, String message) {
        if (!condition) throw new AssertionError(message);
    }
}
