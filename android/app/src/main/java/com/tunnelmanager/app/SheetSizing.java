package com.tunnelmanager.app;

/** A scene has one fixed screen-relative height, capped only by available space. */
final class SheetSizing {
    enum Tier {
        COMPACT(.30f), MEDIUM(.60f), EXPANDED(.80f);
        final float ratio;
        Tier(float ratio) { this.ratio = ratio; }
    }

    static final class Geometry {
        private final int screenHeight, availableHeight;
        Geometry(int screenHeight, int availableHeight) {
            this.screenHeight = screenHeight;
            this.availableHeight = availableHeight;
        }
        int height(Tier scene) {
            if (scene == null) throw new IllegalArgumentException("Sheet scene is required");
            return Math.min(availableHeight, Math.round(screenHeight * scene.ratio));
        }
    }

    static Geometry geometry(int screenHeight, int availableHeight) {
        if (screenHeight <= 0 || availableHeight <= 0) throw new IllegalArgumentException("Sheet bounds must be positive");
        return new Geometry(screenHeight, availableHeight);
    }
}
