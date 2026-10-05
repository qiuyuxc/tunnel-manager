package com.tunnelmanager.app;

import java.util.function.Supplier;

final class RequestScope {
    static final class Snapshot {
        final String server, token, username;
        final long revision;

        Snapshot(String server, String token, String username, long revision) {
            this.server = server; this.token = token; this.username = username; this.revision = revision;
        }

        boolean matches(Snapshot other) {
            return other != null && revision == other.revision && server.equals(other.server)
                    && token.equals(other.token) && username.equals(other.username);
        }
    }

    static final class Stopped extends RuntimeException {
        Stopped(String reason) { super(reason); }
    }

    final Snapshot session;
    private final Supplier<Snapshot> current;
    private String stopped;

    RequestScope(Supplier<Snapshot> current) {
        this.current = current;
        session = current.get();
    }

    synchronized void stop(String reason) {
        if (stopped == null) stopped = reason;
    }

    synchronized String reason() {
        if (stopped == null && (session.server.isEmpty() || session.token.isEmpty() || !session.matches(current.get()))) {
            stopped = "用户会话或服务器已切换，请关闭窗口后重新创建批次";
        }
        return stopped;
    }

    void check() {
        String reason = reason();
        if (reason != null) throw new Stopped(reason);
    }
}
