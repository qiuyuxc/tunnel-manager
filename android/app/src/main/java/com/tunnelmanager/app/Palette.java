package com.tunnelmanager.app;

/**
 * Generated from frontend/src/styles.css by android/tools/gen-tokens.mjs.
 *
 * <p>Do not edit by hand: the web console owns the palette, and re-running the
 * generator is the only way to keep the two from drifting apart.
 */
final class Palette {

    /** Vercel 亮色 — :root */
    static final Palette ENTERPRISE = new Palette("enterprise", "Vercel 亮色", false);

    /** Vercel 暗色 — [data-theme="dark"] */
    static final Palette ENTERPRISE_DARK = new Palette("enterpriseDark", "Vercel 暗色", true);

    final String id;
    final String label;
    final boolean dark;

    int canvas;
    int canvasSoft;
    int canvasSoft2;
    int canvasRaised;
    int ink;
    int body;
    int mute;
    int hairline;
    int hairlineStrong;
    int link;
    int linkHover;
    int focus;
    int success;
    int error;
    int warning;
    int info;
    int sidebar;
    int sidebarHover;
    int sidebarActive;
    int sidebarActiveBg;
    int sidebarText;
    int sidebarTextActive;
    int sidebarHoverText;
    int sidebarDivider;
    int brandIcon;
    int headerBg;
    int headerBorder;
    int statusHealthyBg;
    int statusHealthyBorder;
    int statusHealthyText;
    int statusDownBg;
    int statusDownBorder;
    int statusDownText;
    int statusDegradedBg;
    int statusDegradedBorder;
    int statusDegradedText;
    int btnPrimaryBg;
    int btnPrimaryText;
    int btnPrimaryHover;
    int btnSecondaryBorder;
    int btnSecondaryHoverBorder;
    int btnGhostHover;
    int bannerWarningBg;
    int bannerWarningBorder;
    int bannerWarningText;
    int bannerInfoBg;
    int bannerInfoBorder;
    int bannerInfoText;
    int resultErrorBg;
    int resultErrorBorder;
    int resultErrorText;
    int resultSuccessBg;
    int resultSuccessBorder;
    int resultSuccessText;

    private Palette(String id, String label, boolean dark) {
        this.id = id;
        this.label = label;
        this.dark = dark;
    }

    static {
        Palette p = ENTERPRISE;
        p.canvas                  = 0xFFFFFFFF;
        p.canvasSoft              = 0xFFFAFAFA;
        p.canvasSoft2             = 0xFFF5F5F5;
        p.canvasRaised            = 0xFFFFFFFF;
        p.ink                     = 0xFF171717;
        p.body                    = 0xFF4D4D4D;
        p.mute                    = 0xFF6B6B6B;
        p.hairline                = 0xFFEBEBEB;
        p.hairlineStrong          = 0xFFA1A1A1;
        p.link                    = 0xFF0070F3;
        p.linkHover               = 0xFF0761D1;
        p.focus                   = 0xFF171717;
        p.success                 = 0xFF0E8A72;
        p.error                   = 0xFFEE0000;
        p.warning                 = 0xFFF5A623;
        p.info                    = 0xFF0070F3;
        p.sidebar                 = 0xFFFFFFFF;
        p.sidebarHover            = 0xFFF5F5F5;
        p.sidebarActive           = 0xFF171717;
        p.sidebarActiveBg         = 0xFFF5F5F5;
        p.sidebarText             = 0xFF4D4D4D;
        p.sidebarTextActive       = 0xFF171717;
        p.sidebarHoverText        = 0xFF171717;
        p.sidebarDivider          = 0xFFEBEBEB;
        p.brandIcon               = 0xFF171717;
        p.headerBg                = 0xFFFFFFFF;
        p.headerBorder            = 0xFFEBEBEB;
        p.statusHealthyBg         = 0xFFAAFFEC;
        p.statusHealthyBorder     = 0xFF50E3C2;
        p.statusHealthyText       = 0xFF0B6B57;
        p.statusDownBg            = 0xFFF7D4D6;
        p.statusDownBorder        = 0xFFF0B3B6;
        p.statusDownText          = 0xFFC50000;
        p.statusDegradedBg        = 0xFFFFEFCF;
        p.statusDegradedBorder    = 0xFFF4D79C;
        p.statusDegradedText      = 0xFFAB570A;
        p.btnPrimaryBg            = 0xFF171717;
        p.btnPrimaryText          = 0xFFFFFFFF;
        p.btnPrimaryHover         = 0xFF383838;
        p.btnSecondaryBorder      = 0xFFEBEBEB;
        p.btnSecondaryHoverBorder = 0xFFA1A1A1;
        p.btnGhostHover           = 0xFFF5F5F5;
        p.bannerWarningBg         = 0xFFFFEFCF;
        p.bannerWarningBorder     = 0xFFF4D79C;
        p.bannerWarningText       = 0xFFAB570A;
        p.bannerInfoBg            = 0xFFD3E5FF;
        p.bannerInfoBorder        = 0xFFA8CCFF;
        p.bannerInfoText          = 0xFF0761D1;
        p.resultErrorBg           = 0xFFF7D4D6;
        p.resultErrorBorder       = 0xFFF0B3B6;
        p.resultErrorText         = 0xFFC50000;
        p.resultSuccessBg         = 0xFFAAFFEC;
        p.resultSuccessBorder     = 0xFF50E3C2;
        p.resultSuccessText       = 0xFF0B6B57;
    }

    static {
        Palette p = ENTERPRISE_DARK;
        p.canvas                  = 0xFF0A0A0A;
        p.canvasSoft              = 0xFF000000;
        p.canvasSoft2             = 0xFF1A1A1A;
        p.canvasRaised            = 0xFF111111;
        p.ink                     = 0xFFEDEDED;
        p.body                    = 0xFFA1A1A1;
        p.mute                    = 0xFF9A9A9A;
        p.hairline                = 0xFF2A2A2A;
        p.hairlineStrong          = 0xFF444444;
        p.link                    = 0xFF3291FF;
        p.linkHover               = 0xFF52A8FF;
        p.focus                   = 0xFFEDEDED;
        p.success                 = 0xFF50E3C2;
        p.error                   = 0xFFFF6369;
        p.warning                 = 0xFFF5A623;
        p.info                    = 0xFF3291FF;
        p.sidebar                 = 0xFF000000;
        p.sidebarHover            = 0xFF1A1A1A;
        p.sidebarActive           = 0xFFEDEDED;
        p.sidebarActiveBg         = 0xFF1A1A1A;
        p.sidebarText             = 0xFFA1A1A1;
        p.sidebarTextActive       = 0xFFEDEDED;
        p.sidebarHoverText        = 0xFFEDEDED;
        p.sidebarDivider          = 0xFF2A2A2A;
        p.brandIcon               = 0xFFEDEDED;
        p.headerBg                = 0xFF0A0A0A;
        p.headerBorder            = 0xFF2A2A2A;
        p.statusHealthyBg         = 0xFF0C2B25;
        p.statusHealthyBorder     = 0xFF1F5C4D;
        p.statusHealthyText       = 0xFF7FE9D3;
        p.statusDownBg            = 0xFF2A1213;
        p.statusDownBorder        = 0xFF5C2528;
        p.statusDownText          = 0xFFFF8F92;
        p.statusDegradedBg        = 0xFF2B1F0C;
        p.statusDegradedBorder    = 0xFF5C4520;
        p.statusDegradedText      = 0xFFFFC978;
        p.btnPrimaryBg            = 0xFFEDEDED;
        p.btnPrimaryText          = 0xFF0A0A0A;
        p.btnPrimaryHover         = 0xFFFFFFFF;
        p.btnSecondaryBorder      = 0xFF2A2A2A;
        p.btnSecondaryHoverBorder = 0xFF444444;
        p.btnGhostHover           = 0xFF1A1A1A;
        p.bannerWarningBg         = 0xFF2B1F0C;
        p.bannerWarningBorder     = 0xFF5C4520;
        p.bannerWarningText       = 0xFFFFC978;
        p.bannerInfoBg            = 0xFF10233F;
        p.bannerInfoBorder        = 0xFF1F4D80;
        p.bannerInfoText          = 0xFF7CB8FF;
        p.resultErrorBg           = 0xFF2A1213;
        p.resultErrorBorder       = 0xFF5C2528;
        p.resultErrorText         = 0xFFFF8F92;
        p.resultSuccessBg         = 0xFF0C2B25;
        p.resultSuccessBorder     = 0xFF1F5C4D;
        p.resultSuccessText       = 0xFF7FE9D3;
    }

    static final Palette[] ALL = {
        ENTERPRISE,
        ENTERPRISE_DARK,
    };

    /** Looks a palette up by id, falling back to the Vercel light default. */
    static Palette byId(String id) {
        for (Palette p : ALL) {
            if (p.id.equals(id)) return p;
        }
        return ENTERPRISE;
    }
}
