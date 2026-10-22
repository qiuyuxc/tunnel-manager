package com.tunnelmanager.app;

import android.content.Context;
import android.content.res.Configuration;
import androidx.appcompat.app.AppCompatDelegate;
import androidx.core.os.LocaleListCompat;
import java.util.Locale;

final class I18n {
    private static Context context;
    private static final String PREFERENCE = "language";

    static void init(Context value) {
        context = value.getApplicationContext();
        String preference = context.getSharedPreferences(MainActivity.PREFS, Context.MODE_PRIVATE)
                .getString(PREFERENCE, "system");
        AppCompatDelegate.setApplicationLocales(LocaleListCompat.forLanguageTags(
                "system".equals(preference) ? "" : preference));
    }

    static String preference() {
        return context == null ? "system" : context.getSharedPreferences(MainActivity.PREFS, Context.MODE_PRIVATE)
                .getString(PREFERENCE, "system");
    }

    static void setLanguage(String value) {
        if (context == null || !("system".equals(value) || "zh-CN".equals(value) || "en".equals(value))) return;
        context.getSharedPreferences(MainActivity.PREFS, Context.MODE_PRIVATE).edit().putString(PREFERENCE, value).apply();
        AppCompatDelegate.setApplicationLocales(LocaleListCompat.forLanguageTags("system".equals(value) ? "" : value));
    }

    static Locale locale() {
        LocaleListCompat selected = AppCompatDelegate.getApplicationLocales();
        if (!selected.isEmpty()) return selected.get(0);
        android.os.LocaleList system = android.content.res.Resources.getSystem().getConfiguration().getLocales();
        for (int index = 0; index < system.size(); index++) {
            Locale candidate = system.get(index);
            if ("zh".equals(candidate.getLanguage())) return Locale.SIMPLIFIED_CHINESE;
            if ("en".equals(candidate.getLanguage())) return candidate;
        }
        return Locale.ENGLISH;
    }

    static String languageTag() { return locale().getLanguage().equals("zh") ? "zh-CN" : "en"; }

    static String text(String id) {
        if (context == null) return id;
        Configuration configuration = new Configuration(context.getResources().getConfiguration());
        configuration.setLocale(locale());
        Context localized = context.createConfigurationContext(configuration);
        int resource = localized.getResources().getIdentifier(id.replace('.', '_'), "string", context.getPackageName());
        return resource == 0 ? id : localized.getString(resource);
    }

    private I18n() {}
}
