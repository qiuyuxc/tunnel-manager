package com.tunnelmanager.app;

import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Set;

final class DNSValues {
    static final String[] TYPES = {"A", "AAAA", "CNAME", "TXT", "MX", "NS", "SRV", "CAA", "PTR"};
    static final String[] BATCH_TYPES = {"A", "AAAA", "TXT", "MX", "NS", "PTR"};
    static final String[] SIMPLE_TYPES = {"A", "AAAA", "CNAME", "TXT", "MX", "NS", "PTR"};
    static final int LIMIT = 100;

    static boolean editable(String type) { return Arrays.asList(TYPES).contains(type); }
    static boolean structured(String type) { return "SRV".equals(type) || "CAA".equals(type); }
    static boolean proxy(String type) { return "A".equals(type) || "AAAA".equals(type) || "CNAME".equals(type); }
    static String trimDot(String text) { return text.endsWith(".") ? text.substring(0, text.length() - 1) : text; }

    static boolean ipv4(String value) {
        String[] parts = value.split("\\.", -1);
        if (parts.length != 4) return false;
        for (String part : parts) {
            if (!part.matches("(0|[1-9][0-9]{0,2})") || Integer.parseInt(part) > 255) return false;
        }
        return true;
    }

    static List<Integer> ipv6(String value) {
        if (!value.contains(":") || !value.matches("[0-9a-fA-F:.]+")) return null;
        if (value.contains(".")) {
            int last = value.lastIndexOf(':');
            String tail = value.substring(last + 1);
            if (!ipv4(tail)) return null;
            String[] octets = tail.split("\\.");
            value = value.substring(0, last + 1)
                    + Integer.toHexString(Integer.parseInt(octets[0]) * 256 + Integer.parseInt(octets[1])) + ":"
                    + Integer.toHexString(Integer.parseInt(octets[2]) * 256 + Integer.parseInt(octets[3]));
        }
        String[] halves = value.split("::", -1);
        if (halves.length > 2) return null;
        List<Integer> left = groups(halves[0]);
        List<Integer> right = halves.length == 2 ? groups(halves[1]) : new ArrayList<>();
        if (left == null || right == null) return null;
        int count = left.size() + right.size();
        if (halves.length == 1 ? count != 8 : count >= 8) return null;
        while (left.size() + right.size() < 8) left.add(0);
        left.addAll(right);
        return left;
    }

    private static List<Integer> groups(String value) {
        List<Integer> result = new ArrayList<>();
        if (value.isEmpty()) return result;
        for (String group : value.split(":", -1)) {
            if (!group.matches("[0-9a-fA-F]{1,4}")) return null;
            result.add(Integer.parseInt(group, 16));
        }
        return result;
    }

    static boolean hostname(String value) {
        value = trimDot(value);
        if (value.isEmpty() || value.length() > 253 || ipv4(value) || ipv6(value) != null) return false;
        for (String label : value.split("\\.", -1)) {
            if (label.length() > 63 || !label.matches("[A-Za-z0-9_](?:[A-Za-z0-9_-]*[A-Za-z0-9_])?")) return false;
        }
        return true;
    }

    static String contentError(String type, String raw) {
        String content = raw.trim();
        if (content.isEmpty()) return I18n.text("m_95ad513207d6");
        if (raw.getBytes(StandardCharsets.UTF_8).length > 4096 || raw.indexOf('\0') >= 0) return I18n.text("m_0d8414fecd29");
        switch (type) {
            case "A": return ipv4(content) ? null : I18n.text("m_658bc75675bb");
            case "AAAA": return ipv6(content) != null ? null : I18n.text("m_27651148a1a7");
            case "CNAME": return hostname(content) ? null : I18n.text("m_57a3d93cff1e");
            case "MX": return content.equals(".") || hostname(content) ? null : I18n.text("m_941a243f03d1");
            case "NS": case "PTR": return hostname(content) ? null : type + I18n.text("m_5f48b3165e32");
            case "TXT": return null;
            default: return I18n.text("m_dfa4a7b3892e");
        }
    }

    static int number(String text) {
        try { return text.trim().matches("[0-9]+") ? Integer.parseInt(text.trim()) : -1; }
        catch (NumberFormatException failure) { return -1; }
    }

    static final class Draft {
        String type = "A", name = "", content = "", ttl = "1", priority = "0";
        String weight = "0", port = "443", target = "", flags = "0", tag = "issue", value = "";
        boolean proxied;

        String validate() {
            if (!editable(type)) return I18n.text("m_7afbb9648b38");
            String recordName = trimDot(name.trim());
            if (!recordName.equals("@") && !recordName.equals("*") && !hostname(recordName.replaceFirst("^\\*\\.", ""))) return I18n.text("m_6a5b387c36db");
            int time = number(ttl);
            if (time != 1 && (time < 60 || time > 86400)) return I18n.text("m_9f70f743f754");
            if (type.equals("SRV")) {
                if (!recordName.matches("_[^.]+\\._[^.]+\\..+")) return I18n.text("m_459a6d6b5c52");
                for (String field : new String[]{priority, weight, port}) {
                    if (number(field) < 0 || number(field) > 65535) return I18n.text("m_a6dc72c28ce8");
                }
                if (!target.trim().equals(".") && !hostname(target.trim())) return I18n.text("m_a137ec3f71b5");
                return null;
            }
            if (type.equals("CAA")) {
                if (number(flags) < 0 || number(flags) > 255) return I18n.text("m_0ece41cc39ac");
                if (!tag.matches("[A-Za-z0-9]{1,15}")) return I18n.text("m_1c49e7ee8c85");
                if (value.getBytes(StandardCharsets.UTF_8).length > 4096 || value.indexOf('\0') >= 0 || value.contains("\n") || value.contains("\r")) return I18n.text("m_8f5aab64a61a");
                return null;
            }
            if (type.equals("MX") && (number(priority) < 0 || number(priority) > 65535 || (content.trim().equals(".") && number(priority) != 0))) return I18n.text("m_b19238527e95");
            return contentError(type, content);
        }
    }

    static String key(String type, String content) {
        if (type.equals("TXT")) return content;
        List<Integer> address = type.equals("AAAA") ? ipv6(content.trim()) : null;
        return address != null ? address.toString() : trimDot(content.trim()).toLowerCase(Locale.ROOT);
    }

    static final class Row {
        final int line;
        final String content, error, skipped;
        Row(int line, String content, String error, String skipped) {
            this.line = line; this.content = content; this.error = error; this.skipped = skipped;
        }
    }

    static final class Preview {
        final List<Row> rows = new ArrayList<>();
        String error;
        List<Row> pending() {
            List<Row> result = new ArrayList<>();
            for (Row row : rows) if (row.error == null && row.skipped == null) result.add(row);
            return result;
        }
    }

    static Preview parse(String type, String text, List<String> existing) {
        Preview result = new Preview();
        if (!Arrays.asList(BATCH_TYPES).contains(type)) { result.error = I18n.text("m_1bc27410630a"); return result; }
        if (text.length() > 512000) { result.error = I18n.text("m_7ba14fc9e64c"); return result; }
        Map<String, Integer> seen = new HashMap<>();
        Set<String> known = new HashSet<>();
        for (String value : existing) known.add(key(type, value));
        String[] lines = text.split("\\r?\\n", -1);
        for (int index = 0; index < lines.length; index++) {
            if (lines[index].trim().isEmpty()) continue;
            String content = type.equals("TXT") ? lines[index] : lines[index].trim();
            String error = contentError(type, content), identity = key(type, content);
            String skipped = known.contains(identity) ? I18n.text("m_24a0a2486445") : seen.containsKey(identity) ? I18n.text("m_28bd9e6f0951") + seen.get(identity) + I18n.text("m_304d882808a9") : null;
            result.rows.add(new Row(index + 1, content, error, skipped));
            if (error == null && skipped == null) seen.put(identity, index + 1);
            if (error != null) result.error = I18n.text("m_f32916472225");
        }
        if (result.rows.size() > LIMIT) result.error = I18n.text("m_6127fb2e9e4a");
        return result;
    }

    interface Sender { void send(Row row) throws Exception; }
    interface Progress { void accept(Result result); }
    static final class SendFailure extends Exception {
        final boolean uncertain;
        SendFailure(String message, boolean uncertain) { super(message); this.uncertain = uncertain; }
    }
    static final class Result {
        final Row row;
        final boolean success, uncertain, notSent;
        final String message;
        Result(Row row, boolean success, boolean uncertain, boolean notSent, String message) {
            this.row = row; this.success = success; this.uncertain = uncertain; this.notSent = notSent; this.message = message;
        }
    }

    static List<Result> send(List<Row> rows, Sender sender, Progress progress) {
        if (rows.size() > LIMIT) throw new IllegalArgumentException(I18n.text("m_422c2577fc23"));
        for (Row row : rows) if (row.error != null) throw new IllegalArgumentException(I18n.text("m_f3ada95156bd"));
        List<Result> results = new ArrayList<>();
        String stopped = null;
        for (Row row : rows) {
            if (row.skipped != null) continue;
            Result result;
            if (stopped != null) result = new Result(row, false, false, true, I18n.text("m_a3d2652b6a51") + stopped);
            else {
                try { sender.send(row); result = new Result(row, true, false, false, I18n.text("m_1bcd31e0174a")); }
                catch (RequestScope.Stopped failure) {
                    stopped = failure.getMessage();
                    result = new Result(row, false, false, true, I18n.text("m_a3d2652b6a51") + stopped);
                }
                catch (Exception failure) {
                    boolean uncertain = !(failure instanceof SendFailure) || ((SendFailure) failure).uncertain;
                    if (uncertain) stopped = I18n.text("m_f3e63077b975");
                    result = new Result(row, false, uncertain, false, (uncertain ? I18n.text("m_f983631f6687") : "") + failure.getMessage());
                }
            }
            results.add(result);
            progress.accept(result);
        }
        return results;
    }
}
