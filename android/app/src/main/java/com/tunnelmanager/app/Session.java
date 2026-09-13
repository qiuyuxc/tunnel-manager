package com.tunnelmanager.app;

import android.content.Context;
import android.content.SharedPreferences;

import org.json.JSONArray;
import org.json.JSONObject;

import java.util.Collections;
import java.util.HashSet;
import java.util.Set;

/**
 * Who is signed in, and where the console lives.
 *
 * Mirrors the four keys MainActivity already writes so the login screen and the
 * console never disagree about the session. The permission set is fetched once
 * per launch and gates the navigation, the same way the web sidebar does.
 */
final class Session {

    private static SharedPreferences prefs;
    private static final Set<String> permissions = new HashSet<>();
    private static String nickname = "";
    private static String avatar = "";
    private static String siteName = "";
    private static boolean experimental = false;

    private Session() {
    }

    static void init(Context ctx) {
        if (prefs == null) {
            prefs = ctx.getApplicationContext().getSharedPreferences(MainActivity.PREFS, Context.MODE_PRIVATE);
        }
        Theme.init(ctx);
    }

    static String server() {
        return prefs == null ? "" : prefs.getString(MainActivity.KEY_SERVER, "");
    }

    static String token() {
        return prefs == null ? "" : prefs.getString(MainActivity.KEY_TOKEN, "");
    }

    static String username() {
        return prefs == null ? "" : prefs.getString("username", "");
    }

    static String role() {
        return prefs == null ? "" : prefs.getString("role", "");
    }

    static String nickname() {
        return nickname.isEmpty() ? username() : nickname;
    }

    /** Uploaded avatar path, or "" when the account has not set one. */
    static String avatar() {
        return avatar;
    }

    static String siteName() {
        return siteName.isEmpty() ? "Tunnel Manager" : siteName;
    }

    static boolean isAdmin() {
        return "admin".equals(role());
    }

    static boolean experimental() {
        return experimental;
    }

    static boolean hasPerm(String perm) {
        return isAdmin() || permissions.contains(perm);
    }

    static void save(Context ctx, String server, String token, String user, String userRole) {
        prefs = ctx.getApplicationContext().getSharedPreferences(MainActivity.PREFS, Context.MODE_PRIVATE);
        prefs.edit()
                .putString(MainActivity.KEY_SERVER, server)
                .putString(MainActivity.KEY_TOKEN, token)
                .putString("username", user)
                .putString("role", userRole)
                .apply();
    }

    static void logout(Context ctx) {
        if (prefs != null) prefs.edit().remove(MainActivity.KEY_TOKEN).apply();
        permissions.clear();
    }

    /** Applies the identity payload from /api/me or /api/config. */
    static void applyConfig(JSONObject me) {
        if (me == null) return;
        // The two endpoints this is fed from carry different halves of the
        // picture, so only keys actually present are allowed to overwrite.
        if (me.has("nickname")) nickname = me.optString("nickname", "");
        if (me.has("avatar")) avatar = me.optString("avatar", "");
        String site = me.has("site_name") ? me.optString("site_name", "") : me.optString("name", "");
        if (!site.isEmpty()) siteName = site;
        if (me.has("experimental_features_enabled")) {
            experimental = me.optBoolean("experimental_features_enabled", false);
        } else if (me.has("experimental_features")) {
            experimental = me.optBoolean("experimental_features", false);
        }
        if (me.has("username")) {
            prefs.edit().putString("username", me.optString("username", "")).apply();
        }
        if (me.has("role")) {
            prefs.edit().putString("role", me.optString("role", "")).apply();
        }
        JSONArray list = me.optJSONArray("permissions");
        if (list != null) {
            permissions.clear();
            for (int i = 0; i < list.length(); i++) {
                permissions.add(list.optString(i));
            }
        }
    }

    static Set<String> permissions() {
        return Collections.unmodifiableSet(permissions);
    }
}
