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
        if (content.isEmpty()) return "解析值不能为空";
        if (raw.getBytes(StandardCharsets.UTF_8).length > 4096 || raw.indexOf('\0') >= 0) return "解析值最多 4096 字节，不能包含空字符";
        switch (type) {
            case "A": return ipv4(content) ? null : "A 记录必须填写有效的 IPv4 地址";
            case "AAAA": return ipv6(content) != null ? null : "AAAA 记录必须填写有效的 IPv6 地址";
            case "CNAME": return hostname(content) ? null : "CNAME 记录必须填写域名目标，不能填写 IP 地址";
            case "MX": return content.equals(".") || hostname(content) ? null : "MX 记录必须填写邮件服务器域名";
            case "NS": case "PTR": return hostname(content) ? null : type + " 记录必须填写域名目标";
            case "TXT": return null;
            default: return "此类型请使用结构化字段";
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
            if (!editable(type)) return "不支持的记录类型，只读保护";
            String recordName = trimDot(name.trim());
            if (!recordName.equals("@") && !recordName.equals("*") && !hostname(recordName.replaceFirst("^\\*\\.", ""))) return "请输入有效记录名称（国际域名用 Punycode）";
            int time = number(ttl);
            if (time != 1 && (time < 60 || time > 86400)) return "TTL 必须为 1（自动）或 60–86400 秒";
            if (type.equals("SRV")) {
                if (!recordName.matches("_[^.]+\\._[^.]+\\..+")) return "SRV 名称应为 _服务._协议.域名";
                for (String field : new String[]{priority, weight, port}) {
                    if (number(field) < 0 || number(field) > 65535) return "SRV 优先级、权重和端口必须为 0–65535 的整数";
                }
                if (!target.trim().equals(".") && !hostname(target.trim())) return "SRV 目标必须为域名或 .";
                return null;
            }
            if (type.equals("CAA")) {
                if (number(flags) < 0 || number(flags) > 255) return "CAA flags 必须为 0–255 的整数";
                if (!tag.matches("[A-Za-z0-9]{1,15}")) return "CAA tag 必须为 1–15 个字母或数字";
                if (value.getBytes(StandardCharsets.UTF_8).length > 4096 || value.indexOf('\0') >= 0 || value.contains("\n") || value.contains("\r")) return "CAA value 必须为单行文本，最多 4096 字节";
                return null;
            }
            if (type.equals("MX") && (number(priority) < 0 || number(priority) > 65535 || (content.trim().equals(".") && number(priority) != 0))) return "MX 优先级必须为 0–65535 的整数（空 MX 为 0）";
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
        if (!Arrays.asList(BATCH_TYPES).contains(type)) { result.error = "批量新增仅支持 A / AAAA / TXT / MX / NS / PTR"; return result; }
        if (text.length() > 512000) { result.error = "粘贴内容过大，请拆分批次"; return result; }
        Map<String, Integer> seen = new HashMap<>();
        Set<String> known = new HashSet<>();
        for (String value : existing) known.add(key(type, value));
        String[] lines = text.split("\\r?\\n", -1);
        for (int index = 0; index < lines.length; index++) {
            if (lines[index].trim().isEmpty()) continue;
            String content = type.equals("TXT") ? lines[index] : lines[index].trim();
            String error = contentError(type, content), identity = key(type, content);
            String skipped = known.contains(identity) ? "已存在，跳过" : seen.containsKey(identity) ? "与第 " + seen.get(identity) + " 行重复，跳过" : null;
            result.rows.add(new Row(index + 1, content, error, skipped));
            if (error == null && skipped == null) seen.put(identity, index + 1);
            if (error != null) result.error = "请先修正标注行，整个批次尚未提交";
        }
        if (result.rows.size() > LIMIT) result.error = "每批最多 100 条非空行（含重复行）";
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
        if (rows.size() > LIMIT) throw new IllegalArgumentException("超过数量上限");
        for (Row row : rows) if (row.error != null) throw new IllegalArgumentException("请先修正校验错误");
        List<Result> results = new ArrayList<>();
        String stopped = null;
        for (Row row : rows) {
            if (row.skipped != null) continue;
            Result result;
            if (stopped != null) result = new Result(row, false, false, true, "未发送：" + stopped);
            else {
                try { sender.send(row); result = new Result(row, true, false, false, "已新增"); }
                catch (RequestScope.Stopped failure) {
                    stopped = failure.getMessage();
                    result = new Result(row, false, false, true, "未发送：" + stopped);
                }
                catch (Exception failure) {
                    boolean uncertain = !(failure instanceof SendFailure) || ((SendFailure) failure).uncertain;
                    if (uncertain) stopped = "前一请求结果未知，队列已停止";
                    result = new Result(row, false, uncertain, false, (uncertain ? "结果未知，请核对后再重试：" : "") + failure.getMessage());
                }
            }
            results.add(result);
            progress.accept(result);
        }
        return results;
    }
}
