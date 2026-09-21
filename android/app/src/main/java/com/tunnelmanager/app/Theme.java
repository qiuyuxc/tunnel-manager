package com.tunnelmanager.app;

import android.content.Context;
import android.content.SharedPreferences;
import android.animation.ValueAnimator;

final class Theme {

    private static final String KEY_DARK = "dark";
    private static final String KEY_REDUCED_EFFECTS = "reduced_effects";

    private static Palette current;
    private static boolean dark;
    private static boolean reducedEffects;

    private Theme() {
    }

    /** Safe to call from any activity; the first call seeds it from prefs. */
    static void init(Context ctx) {
        if (current != null) return;
        SharedPreferences prefs = ctx.getSharedPreferences(MainActivity.PREFS, Context.MODE_PRIVATE);
        dark = prefs.getBoolean(KEY_DARK, true);
        reducedEffects = prefs.getBoolean(KEY_REDUCED_EFFECTS, false);
        current = resolve();
    }

    static Palette p() {
        return current != null ? current : Palette.LIQUID_DARK;
    }

    static boolean isDark() {
        return dark;
    }

    static boolean reducedEffects() {
        return reducedEffects;
    }

    static boolean motionEnabled() {
        return !reducedEffects && ValueAnimator.areAnimatorsEnabled();
    }

    static void setReducedEffects(Context ctx, boolean value) {
        reducedEffects = value;
        ctx.getSharedPreferences(MainActivity.PREFS, Context.MODE_PRIVATE)
                .edit().putBoolean(KEY_REDUCED_EFFECTS, value).apply();
    }

    static void setDark(Context ctx, boolean value) {
        if (dark != value) WebPageFragment.resetPalette();
        dark = value;
        current = resolve();
        ctx.getSharedPreferences(MainActivity.PREFS, Context.MODE_PRIVATE)
                .edit().putBoolean(KEY_DARK, dark).apply();
    }

    private static Palette resolve() {
        return dark ? Palette.LIQUID_DARK : Palette.LIQUID_LIGHT;
    }
}
