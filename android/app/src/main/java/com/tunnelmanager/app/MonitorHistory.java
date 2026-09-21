package com.tunnelmanager.app;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.TreeMap;

final class MonitorHistory {
    static final class Sample {
        final long time;
        final String state;
        final double latency;

        Sample(long time, String state, double latency) {
            this.time = time;
            this.state = normalize(state);
            this.latency = latency;
        }

        boolean hasLatency() {
            return ("ok".equals(state) || "warn".equals(state))
                    && Double.isFinite(latency) && latency >= 0;
        }
    }

    static final class Counts {
        int total;
        int ok;
        int warn;
        int down;
        int unknown;

        void add(String state) {
            total++;
            switch (normalize(state)) {
                case "ok": ok++; break;
                case "warn": warn++; break;
                case "down": down++; break;
                default: unknown++; break;
            }
        }

        String state() {
            if (down > 0) return "down";
            if (warn > 0) return "warn";
            if (unknown > 0 || total == 0) return "unknown";
            return "ok";
        }
    }

    final List<Sample> samples;

    MonitorHistory(List<Sample> source) {
        TreeMap<Long, Sample> ordered = new TreeMap<>();
        for (Sample sample : source) {
            if (sample != null && sample.time > 0) ordered.put(sample.time, sample);
        }
        samples = Collections.unmodifiableList(new ArrayList<>(ordered.values()));
    }

    static String normalize(String state) {
        return "ok".equals(state) || "warn".equals(state) || "down".equals(state) ? state : "unknown";
    }

    long firstTime() {
        return samples.isEmpty() ? 0 : samples.get(0).time;
    }

    long lastTime() {
        return samples.isEmpty() ? 0 : samples.get(samples.size() - 1).time;
    }

    boolean hasRecentSamples(long now, long window) {
        for (Sample sample : samples) {
            if (sample.time <= now && sample.time >= now - window) return true;
        }
        return false;
    }

    double fraction(int index) {
        long duration = lastTime() - firstTime();
        return duration == 0 ? 0.5 : (double) (samples.get(index).time - firstTime()) / duration;
    }

    double maxLatency() {
        double maximum = 0;
        for (Sample sample : samples) {
            if (sample.hasLatency()) maximum = Math.max(maximum, sample.latency);
        }
        return maximum;
    }

    boolean connects(int index, long intervalMillis) {
        if (index <= 0 || index >= samples.size() || intervalMillis <= 0) return false;
        Sample previous = samples.get(index - 1);
        Sample current = samples.get(index);
        return previous.hasLatency() && current.hasLatency() && previous.state.equals(current.state)
                && current.time - previous.time <= intervalMillis * 1.75;
    }
}
