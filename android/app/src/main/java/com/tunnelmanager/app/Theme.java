package com.tunnelmanager.app;

import android.content.Context;
import android.content.SharedPreferences;

/**
 * Whether the shell is painted with the Vercel light or dark palette.
 *
 * The choice is per-install rather than per-account: it is a display
 * preference, and the web console already keeps its own in localStorage.
 */
final class Theme {

    private static final String KEY_DARK = "dark";

    private static Palette current;
    private static boolean dark;

    private Theme() {
    }

    /** Safe to call from any activity; the first call seeds it from prefs. */
    static void init(Context ctx) {
        if (current != null) return;
        SharedPreferences prefs = ctx.getSharedPreferences(MainActivity.PREFS, Context.MODE_PRIVATE);
        dark = prefs.getBoolean(KEY_DARK, false);
        current = resolve();
    }

    static Palette p() {
        return current != null ? current : Palette.ENTERPRISE;
    }

    static boolean isDark() {
        return dark;
    }

    static void setDark(Context ctx, boolean value) {
        if (dark != value) WebPageFragment.resetPalette();
        dark = value;
        current = resolve();
        ctx.getSharedPreferences(MainActivity.PREFS, Context.MODE_PRIVATE)
                .edit().putBoolean(KEY_DARK, dark).apply();
    }

    private static Palette resolve() {
        return dark ? Palette.ENTERPRISE_DARK : Palette.ENTERPRISE;
    }
}
