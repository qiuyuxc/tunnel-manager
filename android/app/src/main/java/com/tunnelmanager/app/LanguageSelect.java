package com.tunnelmanager.app;

import android.content.Context;

/** The sign-in and console menus share the app's compact, fixed-height choice list. */
final class LanguageSelect {
    private static final String[] VALUES = {"system", "zh-CN", "en"};

    static String currentLabel() {
        String preference = I18n.preference();
        return "zh-CN".equals(preference) ? "简体中文" : "en".equals(preference) ? "English" : I18n.text("language.system");
    }

    static void show(Context context, Runnable changed) {
        String current = I18n.preference();
        String[] labels = {I18n.text("language.system"), "简体中文", "English"};
        Sheet.Builder sheet = Sheet.of(context, I18n.text("language.switch"), SheetSizing.Tier.COMPACT);
        for (int index = 0; index < VALUES.length; index++) {
            String value = VALUES[index];
            sheet.choice(labels[index], value.equals(current), () -> {
                if (value.equals(I18n.preference())) return;
                I18n.setLanguage(value);
                if (changed != null) changed.run();
            });
        }
        sheet.show();
    }

    private LanguageSelect() {}
}
