package com.tunnelmanager.app;

import android.app.Application;

public final class TunnelApplication extends Application {
    @Override public void onCreate() {
        super.onCreate();
        I18n.init(this);
    }
}
