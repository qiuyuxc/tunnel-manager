package com.tunnelmanager.app;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.List;
import java.util.concurrent.atomic.AtomicReference;

public final class DNSValuesTest {
    public static void main(String[] args) {
        switch (args[0]) {
            case "types":
                for (String type : DNSValues.TYPES) check(DNSValues.editable(type), type);
                check(DNSValues.TYPES.length == 9, "nine types only");
                check(!DNSValues.editable("HTTPS"), "unknown read-only");
                check(DNSValues.proxy("CNAME") && !DNSValues.proxy("NS"), "proxy types");
                check(DNSValues.contentError("NS", "ns.example.com") == null, "NS target");
                check(DNSValues.contentError("PTR", "host.example.com") == null, "PTR target");
                check(DNSValues.contentError("MX", ".") == null, "null MX");
                break;
            case "addresses":
                for (String value : new String[]{"::", "::1", "2001:db8::", "2001:db8:0:0:0:0:0:1", "::ffff:192.0.2.1"}) check(DNSValues.contentError("AAAA", value) == null, value);
                for (String value : new String[]{"1:", "1::2::3", "1:2:3", "1:2:3:4:5:6:7:8:9", "fe80::1%eth0", "localhost", "1.2.3.4"}) check(DNSValues.contentError("AAAA", value) != null, value);
                check(DNSValues.contentError("A", "01.2.3.4") != null, "leading zeros");
                check(DNSValues.contentError("NS", "1.2.3.4") != null, "NS cannot be IP");
                check(DNSValues.key("AAAA", "2001:db8::1").equals(DNSValues.key("AAAA", "2001:0db8:0:0:0:0:0:1")), "canonical IPv6");
                break;
            case "structured":
                DNSValues.Draft draft = new DNSValues.Draft();
                draft.name = "_sip._tcp.example.com"; draft.type = "SRV"; draft.target = "sip.example.com";
                check(draft.validate() == null, "SRV");
                draft.weight = "65536"; check(draft.validate() != null, "SRV range");
                draft.weight = ""; check(draft.validate() != null, "SRV missing");
                draft.type = "CAA"; draft.name = "example.com"; draft.flags = "128"; draft.value = "mailto:security@example.com"; draft.tag = "iodef";
                check(draft.validate() == null, "CAA");
                draft.flags = "256"; check(draft.validate() != null, "flags range");
                draft.flags = "0"; draft.tag = "invalid tag"; check(draft.validate() != null, "tag syntax");
                draft.type = "MX"; draft.content = "mail.example.com"; draft.priority = "-1"; check(draft.validate() != null, "MX priority");
                draft.priority = "0"; draft.ttl = "0"; check(draft.validate() != null, "TTL zero rejected");
                break;
            case "preview":
                DNSValues.Preview preview = DNSValues.parse("A", "192.0.2.1\r\n\r\nbad\n192.0.2.1\n192.0.2.2", Arrays.asList("192.0.2.2"));
                check(preview.rows.get(1).line == 3 && preview.rows.get(1).error != null, "line number validation");
                check(preview.rows.get(2).skipped != null && preview.rows.get(3).skipped != null, "duplicate and existing");
                check(DNSValues.parse("NS", "NS.EXAMPLE.COM\nns.example.com.", Collections.emptyList()).pending().size() == 1, "hostname duplicate");
                check(DNSValues.parse("TXT", "  raw, TEXT  \nraw, TEXT", Collections.emptyList()).pending().get(0).content.equals("  raw, TEXT  "), "TXT intact");
                check(DNSValues.parse("CAA", "x", Collections.emptyList()).error != null, "structured cannot bulk");
                check(DNSValues.parse("TXT", String.join("\n", Collections.nCopies(101, "x")), Collections.emptyList()).error != null, "limit includes duplicates");
                break;
            case "retry":
                List<String> calls = new ArrayList<>();
                List<DNSValues.Result> results = DNSValues.send(DNSValues.parse("A", "192.0.2.1\n192.0.2.2\n192.0.2.1\n192.0.2.3", Collections.emptyList()).rows, row -> {
                    calls.add(row.content);
                    if (row.content.endsWith(".2")) throw new DNSValues.SendFailure("mock rejection", false);
                }, result -> {});
                List<DNSValues.Row> failed = new ArrayList<>();
                for (DNSValues.Result result : results) if (!result.success) failed.add(result.row);
                DNSValues.send(failed, row -> calls.add(row.content), result -> {});
                check(calls.equals(Arrays.asList("192.0.2.1", "192.0.2.2", "192.0.2.3", "192.0.2.2")), "only failed retried");
                calls.clear();
                List<DNSValues.Result> unknown = DNSValues.send(DNSValues.parse("A", "192.0.2.1\n192.0.2.2\n192.0.2.3", Collections.emptyList()).rows, row -> {
                    calls.add(row.content);
                    if (row.content.endsWith(".2")) throw new Exception("disconnected");
                }, result -> {});
                check(calls.size() == 2 && unknown.get(1).uncertain && unknown.get(2).notSent, "unknown stops queue");
                break;
            case "session-boundary":
            case "host-boundary":
            case "inflight-unknown":
                boundary(args[0]);
                break;
            default: throw new AssertionError(args[0]);
        }
    }

    private static void boundary(String scenario) {
        RequestScope.Snapshot original = new RequestScope.Snapshot("old", "old-token", "alice", 1);
        AtomicReference<RequestScope.Snapshot> current = new AtomicReference<>(original);
        RequestScope scope = new RequestScope(current::get);
        List<String> sent = new ArrayList<>();
        List<DNSValues.Row> rows = DNSValues.parse("A", "192.0.2.1\n192.0.2.2\n192.0.2.3", Collections.emptyList()).pending();
        List<DNSValues.Result> completed = DNSValues.send(rows, row -> {
            scope.check();
            sent.add(row.content);
            if (scenario.equals("session-boundary")) {
                current.set(new RequestScope.Snapshot("new", "new-token", "bob", 2));
            } else {
                scope.stop("宿主已销毁");
            }
            if (scenario.equals("inflight-unknown")) throw new DNSValues.SendFailure("response lost", true);
        }, result -> {});
        check(sent.size() == 1, "only in-flight row can reach sender");
        check(completed.get(0).success != scenario.equals("inflight-unknown"), "in-flight result preserved");
        check(completed.get(0).uncertain == scenario.equals("inflight-unknown"), "only dispatched failure can be unknown");
        check(completed.get(1).notSent && !completed.get(1).uncertain, "cancelled row must be unsent, not unknown");
        check(completed.get(2).notSent && !completed.get(2).uncertain, "remaining queue is unsent");
        current.set(original);
        List<DNSValues.Row> remaining = new ArrayList<>();
        for (DNSValues.Result result : completed) if (!result.success) remaining.add(result.row);
        List<DNSValues.Result> retried = DNSValues.send(remaining, row -> {
            scope.check();
            sent.add(row.content);
        }, result -> {});
        check(sent.size() == 1, "stopped batch cannot resume when original session returns");
        check(retried.stream().allMatch(result -> result.notSent && !result.uncertain), "retry remains explicitly unsent");
    }

    private static void check(boolean value, String message) { if (!value) throw new AssertionError(message); }
}
