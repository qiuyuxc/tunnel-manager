package com.tunnelmanager.app;

import java.util.ArrayList;
import java.util.List;

/**
 * The navigation table, mirroring frontend/src/navigation.ts.
 *
 * Same order, same labels, same permission gates — the sidebar and the phone
 * tab bar both read this list, which is what stopped them drifting apart on the
 * web side and is worth repeating here.
 */
final class Nav {

    enum Group { NETWORK, SYSTEM, PERSONAL }

    static final class Item {
        final String path;
        final String label;
        /** Shorter phone wording, when the sidebar label does not fit a tab. */
        final String tabLabel;
        final int icon;
        final Group group;
        final String perm;
        final boolean admin;
        final boolean experimental;
        /** Set for pages the native shell can already render itself. */
        final boolean nativePage;

        Item(String path, String label, String tabLabel, int icon, Group group,
             String perm, boolean admin, boolean experimental, boolean nativePage) {
            this.path = path;
            this.label = label;
            this.tabLabel = tabLabel == null ? label : tabLabel;
            this.icon = icon;
            this.group = group;
            this.perm = perm;
            this.admin = admin;
            this.experimental = experimental;
            this.nativePage = nativePage;
        }
    }

    static final List<Item> ALL = new ArrayList<>();

    static {
        // `nativePage` is the porting switchboard: flip one to true as its
        // native page lands, and everything still false keeps opening the web
        // console, so the app is never half-broken.
        ALL.add(new Item("/dashboard", "控制面板", "概览", R.drawable.ic_nav_dashboard, Group.NETWORK, null, false, false, true));
        ALL.add(new Item("/tunnels", "隧道管理", null, R.drawable.ic_nav_tunnels, Group.NETWORK, "tunnels", false, false, true));
        ALL.add(new Item("/monitors", "服务监控", "监控", R.drawable.ic_nav_monitor, Group.NETWORK, "monitors", false, false, true));
        ALL.add(new Item("/domain", "域名绑定", null, R.drawable.ic_nav_domain, Group.NETWORK, "domain_bind", false, false, true));
        ALL.add(new Item("/dns", "DNS 管理", null, R.drawable.ic_nav_dns, Group.NETWORK, "dns", false, false, true));
        ALL.add(new Item("/lab/ip-selector", "IP 优选实验室", null, R.drawable.ic_nav_lab, Group.NETWORK, null, true, true, true));
        ALL.add(new Item("/settings", "全局设置", null, R.drawable.ic_nav_settings, Group.SYSTEM, null, true, false, true));
        ALL.add(new Item("/telegram", "TG 机器人", null, R.drawable.ic_nav_telegram, Group.SYSTEM, null, false, false, true));
        ALL.add(new Item("/admin", "管理后台", null, R.drawable.ic_nav_admin, Group.SYSTEM, null, true, false, true));
        ALL.add(new Item("/notifications", "通知", null, R.drawable.ic_nav_bell, Group.SYSTEM, null, false, false, true));
        ALL.add(new Item("/account", "账户", null, R.drawable.ic_nav_account, Group.PERSONAL, null, false, false, true));
        ALL.add(new Item("/about", "关于", null, R.drawable.ic_nav_about, Group.PERSONAL, null, false, false, true));
    }

    /** Phone tab bar order; the create button sits between the second and third. */
    static final String[] TAB_PATHS = {"/dashboard", "/monitors", "/dns"};

    private Nav() {
    }

    static List<Item> visible() {
        List<Item> out = new ArrayList<>();
        for (Item item : ALL) {
            if (item.experimental && !Session.experimental()) continue;
            if (item.admin && !Session.isAdmin()) continue;
            if (item.perm != null && !Session.hasPerm(item.perm)) continue;
            out.add(item);
        }
        return out;
    }

    static Item byPath(String path) {
        for (Item item : ALL) {
            if (item.path.equals(path)) return item;
        }
        return null;
    }

    /**
     * `/dashboard` is exact so it does not swallow every other route. The
     * current path is null for the moment between the shell being built and the
     * first page landing, so nothing is active yet.
     */
    static boolean isActive(String current, String candidate) {
        if (current == null) return false;
        if ("/dashboard".equals(candidate)) return candidate.equals(current);
        return candidate.equals(current) || current.startsWith(candidate + "/");
    }
}
