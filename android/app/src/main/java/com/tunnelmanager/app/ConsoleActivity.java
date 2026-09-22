package com.tunnelmanager.app;

import android.annotation.SuppressLint;
import android.content.Intent;
import android.graphics.Bitmap;
import android.graphics.BitmapFactory;
import android.graphics.Outline;
import android.graphics.drawable.GradientDrawable;
import android.graphics.Typeface;
import android.os.Bundle;
import android.view.Gravity;
import android.view.View;
import android.view.ViewGroup;
import android.view.ViewOutlineProvider;
import android.widget.FrameLayout;
import android.widget.ImageView;
import android.widget.LinearLayout;
import android.widget.PopupWindow;
import android.widget.ScrollView;
import android.widget.TextView;

import androidx.activity.OnBackPressedCallback;
import androidx.annotation.NonNull;
import androidx.appcompat.app.AppCompatActivity;
import androidx.fragment.app.Fragment;
import androidx.fragment.app.FragmentManager;

import org.json.JSONObject;

import java.util.function.Consumer;
import java.util.ArrayList;
import java.util.List;
import android.text.TextUtils;

/**
 * The native console shell.
 *
 * Owns everything that has to stay put while pages come and go: the app bar,
 * the navigation (a sidebar when there is room, a tab bar when there is not),
 * and the fragment back stack. Pages themselves are {@link PageFragment}s.
 *
 */
public class ConsoleActivity extends AppCompatActivity {

    private static final String STATE_PATH = "path";
    /** Console route to open on launch, set by a tapped notification. */
    static final String EXTRA_PATH = "route";

    private FrameLayout content;
    private LinearLayout bottomBar;
    private View shell;
    private ImageView themeButton;
    private ImageView navigationButton;
    private final List<String> bottomPaths = new ArrayList<>();
    /** The scrolling sidebar, or null on narrow screens. */
    private ScrollView sidebar;
    /** Open identity popover, or null. Held so a palette change can close it. */
    private PopupWindow accountMenu;
    /** Top-bar avatar slot, repainted once the account image finishes loading. */
    private FrameLayout avatarSlot;
    private Bitmap avatarBitmap;
    private String avatarUrl = "";
    private boolean wide;
    /** Null until the first page lands, so the first {@link #open} always runs. */
    private String currentPath;
    private boolean restoring;
    /** Set once a dead session has been acted on, so it is acted on once. */
    private boolean sessionLostHandled;
    private TunnelCreateSheet tunnelCreateSheet;

    @Override
    protected void onCreate(Bundle saved) {
        super.onCreate(saved);
        Session.init(this);
        if (Session.token().isEmpty()) {
            backToLogin(null);
            return;
        }
        // Any 401 from here on means the session died under us; the shell owns
        // the only sane answer to that, so it registers itself as the handler.
        Api.watchSession(this::onSessionLost);
        wide = getResources().getConfiguration().screenWidthDp >= UI.WIDE_DP;
        if (saved != null) {
            currentPath = saved.getString(STATE_PATH, currentPath);
            restoring = true;
        }
        applySystemBars();
        shell = buildShell();
        setContentView(shell);
        tunnelCreateSheet = new TunnelCreateSheet(this, saved);
        installBackHandler();
        loadIdentity();
        if (saved == null) open(initialRoute(getIntent()));
    }

    @Override
    protected void onNewIntent(@NonNull Intent intent) {
        super.onNewIntent(intent);
        setIntent(intent);
        String route = intent.getStringExtra(EXTRA_PATH);
        if (route != null && !route.isEmpty()) open(route);
    }

    @Override
    protected void onDestroy() {
        if (tunnelCreateSheet != null) tunnelCreateSheet.dispose();
        Api.stopWatchingSession();
        super.onDestroy();
    }

    /**
     * The server stopped accepting the stored token.
     *
     * The token has to go before the sign-in screen comes up: leaving it behind
     * is what made the app relaunch the console straight from the login page,
     * take another 401 and bounce back, flashing the launch screen forever.
     *
     * A request's own failure callback usually arrives right after the shell
     * was already told, so only the first one leaves.
     */
    private void onSessionLost() {
        if (sessionLostHandled || isFinishing() || isDestroyed()) return;
        sessionLostHandled = true;
        Session.logout(this);
        backToLogin(MainActivity.NOTICE_EXPIRED);
    }

    private String initialRoute(Intent intent) {
        String route = intent == null ? null : intent.getStringExtra(EXTRA_PATH);
        return route == null || route.isEmpty() ? "/dashboard" : route;
    }

    @Override
    protected void onSaveInstanceState(@NonNull Bundle outState) {
        if (tunnelCreateSheet != null) tunnelCreateSheet.save(outState);
        outState.putString(STATE_PATH, currentPath == null ? "/dashboard" : currentPath);
        super.onSaveInstanceState(outState);
    }

    // ----------------------------------------------------------------- chrome

    private View buildShell() {
        Palette p = Theme.p();
        LinearLayout root = UI.column(this);
        root.setBackgroundColor(p.canvas);
        root.addView(buildAppBar());
        FrameLayout stage = new FrameLayout(this);
        root.addView(stage, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, 0, 1f));
        LinearLayout body = UI.row(this);
        if (wide) {
            sidebar = buildSidebar();
            body.addView(sidebar, new LinearLayout.LayoutParams(UI.dp(UI.SIDEBAR_W), ViewGroup.LayoutParams.MATCH_PARENT));
        }
        content = new FrameLayout(this);
        content.setId(R.id.tm_content);
        body.addView(content, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.MATCH_PARENT, 1f));
        stage.addView(body, new FrameLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT));

        if (!wide) {
            stage.addView(buildBottomArea());
        }
        return root;
    }

    private View buildBottomArea() {
        FrameLayout area = new FrameLayout(this);
        area.setLayoutParams(new FrameLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, UI.dp(UI.TABBAR_H + 24), Gravity.BOTTOM));
        bottomBar = buildBottomBar();
        FrameLayout.LayoutParams barParams = new FrameLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, UI.dp(UI.TABBAR_H), Gravity.BOTTOM);
        barParams.setMargins(UI.dp(20), 0, UI.dp(20), UI.dp(12));
        area.addView(bottomBar, barParams);
        return area;
    }

    /** A 1dp rule, for the seams the design draws between chrome and page. */
    private View hairline(int color) {
        View v = new View(this);
        v.setBackgroundColor(color);
        v.setLayoutParams(new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, Math.max(1, UI.dp(1))));
        return v;
    }

    private View buildAppBar() {
        Palette p = Theme.p();
        LinearLayout bar = UI.row(this);
        bar.setLayoutParams(new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, UI.dp(UI.HEADER_H)));
        bar.setBackgroundColor(p.headerBg);
        bar.setPadding(UI.dp(12), 0, UI.dp(UI.SM), 0);
        navigationButton = (ImageView) iconButton(R.drawable.ic_nav_tunnels, p.success, view -> {
            if (isDetailRoute()) getOnBackPressedDispatcher().onBackPressed();
            else open("/dashboard");
        });
        navigationButton.setContentDescription("控制面板");
        bar.addView(navigationButton);
        TextView brand = UI.text(this, Session.siteName(), 18, p.ink, Typeface.BOLD);
        brand.setSingleLine(true);
        brand.setEllipsize(TextUtils.TruncateAt.END);
        bar.addView(brand, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));

        // Day/night is one tap. There is a single visual theme, so a palette
        // menu would have exactly two entries and no reason to exist; going
        // back is the system gesture, so the bar carries no back arrow either.
        themeButton = (ImageView) iconButton(Theme.isDark() ? R.drawable.ic_nav_sun : R.drawable.ic_nav_moon,
                p.body, v -> toggleDark());
        themeButton.setContentDescription("切换深浅主题");
        bar.addView(themeButton);
        bar.addView(avatarButton());
        return bar;
    }

    private ScrollView buildSidebar() {
        Palette p = Theme.p();
        ScrollView scroll = new ScrollView(this);
        scroll.setBackgroundColor(p.sidebar);
        LinearLayout column = UI.column(this);
        int pad = UI.dp(UI.SM);
        column.setPadding(pad, UI.dp(UI.SM), pad, UI.dp(UI.SM));
        scroll.addView(column, new FrameLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));
        fillSidebar(column);
        return scroll;
    }

    private void fillSidebar(LinearLayout column) {
        Nav.Group lastGroup = null;
        for (Nav.Item item : Nav.visible()) {
            if (item.group != lastGroup) {
                lastGroup = item.group;
                TextView heading = UI.label(this, groupLabel(item.group));
                UI.margin(heading, UI.SM, UI.MD, 0, UI.XS);
                column.addView(heading);
            }
            column.addView(sidebarRow(item));
        }
    }

    private View sidebarRow(Nav.Item item) {
        Palette p = Theme.p();
        boolean active = Nav.isActive(currentPath, item.path);
        LinearLayout row = UI.row(this);
        row.setPadding(UI.dp(UI.SM), UI.dp(9), UI.dp(UI.SM), UI.dp(9));

        ImageView icon = new ImageView(this);
        icon.setImageResource(item.icon);
        UI.tint(icon, active ? p.sidebarTextActive : p.sidebarText);
        LinearLayout.LayoutParams iconLp = new LinearLayout.LayoutParams(UI.dp(18), UI.dp(18));
        iconLp.rightMargin = UI.dp(10);
        row.addView(icon, iconLp);

        row.addView(UI.text(this, item.label, 14,
                active ? p.sidebarTextActive : p.sidebarText,
                active ? Typeface.BOLD : Typeface.NORMAL));

        row.setBackground(UI.pressable(UI.roundedStroke(
                active ? p.sidebarActiveBg : p.sidebar,
                UI.RADIUS_MD,
                active ? p.sidebarDivider : p.sidebar,
                1), p.btnGhostHover));
        row.setClickable(true);
        row.setOnClickListener(v -> open(item.path));
        return row;
    }

    private LinearLayout buildBottomBar() {
        LinearLayout bar = UI.row(this);
        bar.setBackground(new GlassDrawable(UI.RADIUS_PILL, false));
        bar.setPadding(UI.dp(6), UI.dp(6), UI.dp(6), UI.dp(6));
        bar.setElevation(UI.dp(Theme.reducedEffects() ? 0 : 6));
        fillBottomBar(bar);
        return bar;
    }

    private void fillBottomBar(LinearLayout bar) {
        List<String> paths = new ArrayList<>();
        List<Nav.Item> visible = Nav.visible();
        for (String path : Nav.TAB_PATHS) {
            Nav.Item item = Nav.byPath(path);
            if ("/dns".equals(path) && canCreate()) paths.add("create");
            if (item != null && visible.contains(item)) paths.add(path);
        }
        paths.add("more");
        if (!paths.equals(bottomPaths)) {
            bottomPaths.clear();
            bottomPaths.addAll(paths);
            bar.removeAllViews();
            for (String path : paths) {
                if ("create".equals(path)) {
                    FrameLayout slot = new FrameLayout(this);
                    slot.addView(fabButton(), new FrameLayout.LayoutParams(UI.dp(44), UI.dp(44), Gravity.CENTER));
                    bar.addView(slot, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.MATCH_PARENT, 1f));
                } else if ("more".equals(path)) {
                    bar.addView(tabButton(R.drawable.ic_nav_more, "更多", false, this::showMoreMenu));
                } else {
                    Nav.Item item = Nav.byPath(path);
                    bar.addView(tabButton(item.icon, "/dns".equals(path) ? "DNS" : item.tabLabel,
                            false, view -> open(path)));
                }
            }
            bar.setWeightSum(paths.size());
        }
        for (int index = 0; index < paths.size(); index++) {
            String path = paths.get(index);
            if ("create".equals(path)) continue;
            boolean active = "more".equals(path) ? currentPath != null && !onTab(currentPath)
                    : Nav.isActive(currentPath, path);
            paintTab((LinearLayout) bar.getChildAt(index), active);
        }
    }

    private boolean canCreate() {
        return Session.hasPerm("tunnels") || Session.hasPerm("domain_bind");
    }

    private void paintTab(LinearLayout box, boolean active) {
        Palette palette = Theme.p();
        int tint = active ? palette.success : palette.mute;
        if (box.isSelected() != active) {
            box.setSelected(active);
            box.setBackground(active ? new GlassDrawable(UI.RADIUS_PILL, true) : null);
        }
        UI.tint((ImageView) box.getChildAt(0), tint);
        if (box.getChildCount() > 1) ((TextView) box.getChildAt(1)).setTextColor(tint);
    }

    private View tabButton(int icon, String label, boolean active, Consumer<View> action) {
        Palette p = Theme.p();
        int tint = active ? p.success : p.mute;
        LinearLayout box = UI.column(this);
        box.setGravity(Gravity.CENTER);
        box.setLayoutParams(new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.MATCH_PARENT, 1f));

        ImageView image = new ImageView(this);
        image.setImageResource(icon);
        UI.tint(image, tint);
        LinearLayout.LayoutParams iconLp = new LinearLayout.LayoutParams(UI.dp(22), UI.dp(22));
        box.addView(image, iconLp);

        if (label != null) {
            TextView caption = UI.text(this, label, 10, tint, active ? Typeface.BOLD : Typeface.NORMAL);
            UI.margin(caption, 0, 3, 0, 0);
            box.addView(caption);
        }
        box.setClickable(true);
        box.setFocusable(true);
        box.setContentDescription(label);
        UI.pressFeedback(box);
        box.setOnClickListener(action::accept);
        return box;
    }

    private View fabButton() {
        Palette p = Theme.p();
        ImageView fab = new ImageView(this);
        fab.setImageResource(R.drawable.ic_nav_plus);
        UI.tint(fab, p.btnPrimaryText);
        int pad = UI.dp(12);
        fab.setPadding(pad, pad, pad, pad);
        fab.setBackground(UI.pressable(UI.circle(p.btnPrimaryBg, 0, 0), p.btnGhostHover));
        fab.setClickable(true);
        fab.setFocusable(true);
        fab.setContentDescription("新建隧道或绑定域名");
        UI.pressFeedback(fab);
        fab.setOnClickListener(this::showCreateMenu);
        return fab;
    }

    private View iconButton(int icon, int tint, View.OnClickListener onClick) {
        Palette p = Theme.p();
        ImageView v = new ImageView(this);
        v.setImageResource(icon);
        UI.tint(v, tint);
        int pad = UI.dp(9);
        v.setPadding(pad, pad, pad, pad);
        v.setLayoutParams(new LinearLayout.LayoutParams(UI.dp(44), UI.dp(44)));
        v.setBackground(UI.pressable(UI.rounded(p.headerBg, UI.RADIUS_PILL), p.btnGhostHover));
        v.setClickable(true);
        v.setFocusable(true);
        UI.pressFeedback(v);
        v.setOnClickListener(onClick);
        return v;
    }

    private View avatarButton() {
        Palette p = Theme.p();
        avatarSlot = new FrameLayout(this);
        LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(UI.dp(32), UI.dp(32));
        lp.leftMargin = UI.dp(UI.XS);
        avatarSlot.setLayoutParams(lp);
        avatarSlot.setBackground(UI.pressable(
                UI.roundedStroke(p.btnPrimaryBg, UI.RADIUS_PILL, p.hairline, 1), p.btnGhostHover));
        clipToPill(avatarSlot);
        avatarSlot.setClickable(true);
        avatarSlot.setOnClickListener(this::showAccountMenu);
        paintAvatar(avatarSlot, 14, p.btnPrimaryText);
        return avatarSlot;
    }

    /**
     * Draws the account avatar: the uploaded image when there is one, the
     * initial otherwise. Both the bar button and the popover head run through
     * here so they can never disagree about which is showing.
     */
    private void paintAvatar(FrameLayout slot, float textSizeSp, int textColor) {
        if (slot == null) return;
        slot.removeAllViews();
        if (avatarBitmap != null) {
            ImageView image = new ImageView(this);
            image.setImageBitmap(avatarBitmap);
            image.setScaleType(ImageView.ScaleType.CENTER_CROP);
            slot.addView(image, new FrameLayout.LayoutParams(
                    ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT));
            return;
        }
        String name = Session.nickname();
        TextView initial = UI.text(this, name.isEmpty() ? "?" : name.substring(0, 1).toUpperCase(),
                textSizeSp, textColor, Typeface.BOLD);
        initial.setGravity(Gravity.CENTER);
        slot.addView(initial, new FrameLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT));
    }

    /** Clips a view to a pill so a square upload shows as a circle. */
    private static void clipToPill(final View view) {
        view.setOutlineProvider(new ViewOutlineProvider() {
            @Override
            public void getOutline(View v, Outline outline) {
                outline.setRoundRect(0, 0, v.getWidth(), v.getHeight(), v.getWidth() / 2f);
            }
        });
        view.setClipToOutline(true);
    }

    private void loadAvatar() {
        final String url = Session.avatar();
        if (url.isEmpty() || url.equals(avatarUrl)) return;
        Api.async(() -> Api.asset(url), bytes -> {
            if (!url.equals(Session.avatar())) return;
            Bitmap bitmap = BitmapFactory.decodeByteArray(bytes, 0, bytes.length);
            if (bitmap == null) return;
            avatarBitmap = bitmap;
            avatarUrl = url;
            paintAvatar(avatarSlot, 14, Theme.p().btnPrimaryText);
        }, this::ignoreFailure);
    }

    // ---------------------------------------------------------------- actions

    private void toggleDark() {
        closeAccountMenu();
        Palette previous = Theme.p();
        Theme.setDark(this, !Theme.isDark());
        UI.retheme(shell, previous);
        applySystemBars();
        themeButton.setImageResource(Theme.isDark() ? R.drawable.ic_nav_sun : R.drawable.ic_nav_moon);
        refreshChrome();
        Fragment page = getSupportFragmentManager().findFragmentById(R.id.tm_content);
        if (page instanceof WebPageFragment) ((WebPageFragment) page).refreshTheme();
    }

    private void closeAccountMenu() {
        if (accountMenu == null) return;
        accountMenu.dismiss();
        accountMenu = null;
    }

    /**
     * The identity popover: who is signed in, the two display switches, and the
     * way out. Anchored under the avatar the way the phone web chrome does it.
     */
    private void showAccountMenu(View anchor) {
        Palette p = Theme.p();
        final int WIDTH = 224;

        LinearLayout card = UI.column(this);
        card.setBackground(UI.roundedStroke(p.canvasRaised, 10, p.hairline, 1));
        card.setClipToOutline(true);

        card.addView(identityHead());
        card.addView(menuSeparator());
        card.addView(menuRow(R.drawable.ic_nav_account, "账户信息", false, () -> {
            closeAccountMenu();
            open("/account");
        }));
        card.addView(darkModeRow());
        card.addView(reducedEffectsRow());
        card.addView(menuSeparator());
        card.addView(menuRow(R.drawable.ic_nav_logout, "退出登录", true, () -> {
            Session.logout(this);
            backToLogin(null);
        }));

        PopupWindow popup = new PopupWindow(card, UI.dp(WIDTH), ViewGroup.LayoutParams.WRAP_CONTENT);
        accountMenu = popup;
        popup.setBackgroundDrawable(UI.roundedStroke(p.canvasRaised, 10, p.hairline, 1));
        popup.setOutsideTouchable(true);
        popup.setFocusable(true);
        popup.setElevation(UI.dp(12));
        popup.setOnDismissListener(() -> accountMenu = null);
        // Right edges line up 12dp in from the screen edge, the web's placement.
        popup.showAsDropDown(anchor, anchor.getWidth() - UI.dp(4) - UI.dp(WIDTH), UI.dp(2));
    }

    private void showCreateMenu(View anchor) {
        Sheet.Builder sheet = Sheet.of(this, "新建连接");
        if (Session.hasPerm("tunnels")) sheet.action(R.drawable.ic_nav_tunnels,
                "新建隧道", "创建 Cloudflare Tunnel 并获取连接命令", this::showCreateTunnel);
        if (Session.hasPerm("domain_bind")) sheet.action(R.drawable.ic_nav_domain,
                "绑定域名", "为当前服务配置访问地址", () -> open("/domain"));
        sheet.show();
    }

    void showCreateTunnel() {
        if (tunnelCreateSheet != null) tunnelCreateSheet.show();
    }

    void onTunnelCreated() {
        Fragment page = getSupportFragmentManager().findFragmentById(R.id.tm_content);
        if (page instanceof TunnelsFragment) ((TunnelsFragment) page).reload();
    }

    private void showMoreMenu(View anchor) {
        Sheet.Builder sheet = Sheet.of(this, "你的工作空间");
        sheet.content(identityHead());
        List<Nav.Item> visible = Nav.visible();
        List<Nav.Item> network = new ArrayList<>();
        String[] paths = {"/tunnels", "/domain", "/dns", "/lab/ip-selector", "/monitors"};
        for (String path : paths) {
            for (Nav.Item item : visible) {
                if (path.equals(item.path)) network.add(item);
            }
        }
        if (Session.hasPerm("domain_bind")) network.add(new Nav.Item("/domain/batch", "批量绑定", null,
                R.drawable.ic_nav_plus, Nav.Group.NETWORK, "domain_bind", false, false, true));
        if (!network.isEmpty()) sheet.label("网络与解析").content(moreGrid(sheet, network));
        for (Nav.Group group : new Nav.Group[]{Nav.Group.SYSTEM, Nav.Group.PERSONAL}) {
            boolean labelled = false;
            for (Nav.Item item : visible) {
                if (item.group != group) continue;
                if (!labelled) {
                    sheet.label(groupLabel(group));
                    labelled = true;
                }
                sheet.action(item.icon, item.label, routeDescription(item.path), () -> open(item.path));
            }
        }
        sheet.show();
    }

    private static boolean onTab(String path) {
        for (String candidate : Nav.TAB_PATHS) {
            if (Nav.isActive(path, candidate)) return true;
        }
        return false;
    }

    private View moreGrid(Sheet.Builder sheet, List<Nav.Item> items) {
        LinearLayout grid = UI.column(this);
        int width = Math.min(getResources().getConfiguration().screenWidthDp, 560);
        int columns = width >= 380 && getResources().getConfiguration().fontScale <= 1.15f ? 3 : 2;
        for (int offset = 0; offset < items.size(); offset += columns) {
            LinearLayout row = UI.row(this);
            for (int column = 0; column < columns; column++) {
                int index = offset + column;
                LinearLayout tile = UI.column(this);
                LinearLayout.LayoutParams cell = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
                if (column > 0) cell.leftMargin = UI.dp(UI.SM);
                row.addView(tile, cell);
                if (index >= items.size()) continue;
                Nav.Item item = items.get(index);
                tile.setGravity(Gravity.CENTER);
                tile.setPadding(UI.dp(UI.SM), UI.dp(UI.MD), UI.dp(UI.SM), UI.dp(UI.MD));
                tile.setMinimumHeight(UI.dp(104));
                tile.setBackground(UI.pressable(UI.rounded(Theme.p().canvasSoft2, UI.RADIUS_LG), Theme.p().btnGhostHover));
                ImageView icon = new ImageView(this);
                icon.setImageResource(item.icon);
                UI.tint(icon, Theme.p().success);
                tile.addView(icon, new LinearLayout.LayoutParams(UI.dp(24), UI.dp(24)));
                String label = "/lab/ip-selector".equals(item.path) ? "IP 优选" : item.label;
                TextView title = UI.text(this, label, 13, Theme.p().ink, Typeface.NORMAL);
                title.setGravity(Gravity.CENTER);
                UI.margin(title, 0, UI.SM, 0, 0);
                tile.addView(title);
                tile.setContentDescription(item.label);
                tile.setClickable(true);
                tile.setFocusable(true);
                UI.pressFeedback(tile);
                tile.setOnClickListener(view -> {
                    sheet.dismiss();
                    open(item.path);
                });
            }
            UI.addRow(grid, row, offset == 0 ? 0 : UI.SM);
        }
        return grid;
    }

    private static String routeDescription(String path) {
        switch (path) {
            case "/settings": return "站点品牌、网络与全局配置";
            case "/notifications": return "通知渠道、事件与设备告警";
            case "/account": return "个人资料、通行密钥与账户授权";
            case "/about": return "版本、更新与项目文档";
            default: return "打开工具";
        }
    }

    // ------------------------------------------------------------ popover bits

    private View identityHead() {
        Palette p = Theme.p();
        String name = Session.nickname();

        LinearLayout head = UI.row(this);
        head.setPadding(UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD));

        FrameLayout avatar = new FrameLayout(this);
        avatar.setBackground(UI.rounded(p.canvasSoft2, UI.RADIUS_PILL));
        clipToPill(avatar);
        paintAvatar(avatar, 15, p.ink);
        LinearLayout.LayoutParams avatarLp = new LinearLayout.LayoutParams(UI.dp(38), UI.dp(38));
        avatarLp.rightMargin = UI.dp(10);
        head.addView(avatar, avatarLp);

        LinearLayout ident = UI.column(this);
        ident.addView(UI.text(this, name, 14, p.ink, Typeface.BOLD));
        ident.addView(UI.text(this, "管理员", 12, p.mute, Typeface.NORMAL));
        head.addView(ident);
        return head;
    }

    private View menuSeparator() {
        View v = new View(this);
        v.setBackgroundColor(Theme.p().hairline);
        v.setLayoutParams(new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, Math.max(1, UI.dp(1))));
        return v;
    }

    /** A popover row: 17dp icon, label, and a chevron when it leads somewhere. */
    private View menuRow(int iconRes, String label, boolean danger, Runnable onClick) {
        Palette p = Theme.p();
        int tint = danger ? p.error : p.ink;

        LinearLayout row = UI.row(this);
        row.setPadding(UI.dp(UI.MD), UI.dp(9), UI.dp(UI.MD), UI.dp(9));
        row.setBackground(UI.pressable(UI.rounded(p.canvasRaised, UI.RADIUS_LG), p.btnGhostHover));

        ImageView icon = new ImageView(this);
        icon.setImageResource(iconRes);
        UI.tint(icon, danger ? p.error : p.body);
        LinearLayout.LayoutParams iconLp = new LinearLayout.LayoutParams(UI.dp(17), UI.dp(17));
        iconLp.rightMargin = UI.dp(10);
        row.addView(icon, iconLp);

        row.addView(UI.text(this, label, 14, tint, Typeface.NORMAL),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));

        if (!danger) {
            ImageView chevron = new ImageView(this);
            chevron.setImageResource(R.drawable.ic_nav_chevron);
            UI.tint(chevron, p.mute);
            row.addView(chevron, new LinearLayout.LayoutParams(UI.dp(15), UI.dp(15)));
        }
        row.setClickable(true);
        row.setOnClickListener(v -> onClick.run());
        return row;
    }

    private View darkModeRow() {
        Palette p = Theme.p();
        LinearLayout row = UI.row(this);
        row.setPadding(UI.dp(UI.MD), UI.dp(9), UI.dp(UI.MD), UI.dp(9));
        row.setBackground(UI.pressable(UI.rounded(p.canvasRaised, UI.RADIUS_LG), p.btnGhostHover));

        ImageView icon = new ImageView(this);
        icon.setImageResource(Theme.isDark() ? R.drawable.ic_nav_sun : R.drawable.ic_nav_moon);
        UI.tint(icon, p.body);
        LinearLayout.LayoutParams iconLp = new LinearLayout.LayoutParams(UI.dp(17), UI.dp(17));
        iconLp.rightMargin = UI.dp(10);
        row.addView(icon, iconLp);

        row.addView(UI.text(this, "暗色模式", 14, p.ink, Typeface.NORMAL),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));

        UI.Toggle toggle = new UI.Toggle(this, Theme.isDark());
        row.addView(toggle, new LinearLayout.LayoutParams(UI.dp(34), UI.dp(20)));

        row.setClickable(true);
        row.setOnClickListener(v -> {
            toggleDark();
        });
        return row;
    }

    // ------------------------------------------------------------- navigation

    void open(String path) {
        if (content == null) return;
        // MainActivity hands the console a route every time it is brought
        // forward, so the page already on screen arrives here a second time.
        // Re-opening it would stack a duplicate and leave the container empty
        // once the back stack unwound past it.
        if (path.equals(currentPath)) return;

        Fragment page = nativePage(path);
        if (page == null) {
            Nav.Item item = Nav.byPath(path);
            // A sidebar item that claims a native page but has none is a bug in
            // the switchboard, not a route the web console should render.
            if (item != null && item.nativePage) return;
            // Everything else — batch binding, public status, the routes with no
            // sidebar entry of their own — still has a real console page.
            page = WebPageFragment.forRoute(path);
        }
        currentPath = path;
        getSupportFragmentManager().beginTransaction()
                .setReorderingAllowed(true)
                .replace(R.id.tm_content, page)
                .addToBackStack(path)
                .commit();
        refreshChrome();
    }

    /**
     * Page routes the native shell can draw itself; null means "not yet", and
     * the caller falls back to the web console. Detail routes carry an id and
     * so have no {@link Nav.Item} of their own.
     */
    private Fragment nativePage(String path) {
        if ("/dashboard".equals(path)) return new DashboardFragment();
        if ("/tunnels".equals(path)) return new TunnelsFragment();
        if (path.startsWith("/tunnels/")) {
            String id = path.substring("/tunnels/".length());
            if (!id.isEmpty()) return TunnelDetailFragment.forId(id);
        }
        if ("/domain".equals(path)) return new DomainFragment();
        if ("/domain/batch".equals(path)) return new BatchBindFragment();
        if ("/dns".equals(path)) return new DNSFragment();
        if ("/notifications".equals(path)) return new NotificationsFragment();
        if ("/account".equals(path)) return new AccountFragment();
        if ("/about".equals(path)) return new AboutFragment();
        if ("/settings".equals(path)) return new SettingsFragment();
        if ("/lab/ip-selector".equals(path)) return new LabFragment();
        if ("/monitors".equals(path)) return new MonitorsFragment();
        if (path.startsWith("/monitors/")) {
            String id = path.substring("/monitors/".length());
            if (!id.isEmpty()) return MonitorDetailFragment.forId(id);
        }
        return null;
    }

    /** Rebuilds the navigation so the active row follows the current page. */
    private void refreshChrome() {
        if (navigationButton != null) {
            navigationButton.setImageResource(isDetailRoute() ? R.drawable.ic_nav_back : R.drawable.ic_nav_tunnels);
            navigationButton.setContentDescription(isDetailRoute() ? "返回上一页" : "控制面板");
        }
        // A dozen views at most: rebuilding beats tracking which row moved.
        if (sidebar != null) {
            LinearLayout column = (LinearLayout) sidebar.getChildAt(0);
            column.removeAllViews();
            fillSidebar(column);
        }
        if (bottomBar != null) fillBottomBar(bottomBar);
    }

    private void installBackHandler() {
        final FragmentManager manager = getSupportFragmentManager();
        manager.addOnBackStackChangedListener(() -> {
            Fragment current = manager.findFragmentById(R.id.tm_content);
            if (current instanceof PageFragment) currentPath = ((PageFragment) current).route();
            refreshChrome();
        });
        getOnBackPressedDispatcher().addCallback(this, new OnBackPressedCallback(true) {
            @Override
            public void handleOnBackPressed() {
                // A PopupWindow does not consume BACK the way a dialog does, so
                // an open identity menu would otherwise take the page with it.
                if (accountMenu != null && accountMenu.isShowing()) {
                    closeAccountMenu();
                    return;
                }
                if (Sheet.dismissVisible()) return;
                Fragment top = manager.findFragmentById(R.id.tm_content);
                if (top instanceof WebPageFragment && ((WebPageFragment) top).canGoBack()) {
                    ((WebPageFragment) top).goBack();
                    return;
                }
                if (manager.getBackStackEntryCount() > 1) {
                    manager.popBackStack();
                    return;
                }
                setEnabled(false);
                getOnBackPressedDispatcher().onBackPressed();
                setEnabled(true);
            }
        });
    }

    // ------------------------------------------------------------------ setup

    /** Branding and permissions; the nav cannot be filtered until both land. */
    private void loadIdentity() {
        Api.async(() -> Api.get("/api/auth/me"), me -> {
            Session.applyConfig(me);
            loadAvatar();
            rebuildNav();
        }, failure -> {
            if (failure.auth) onSessionLost();
        });
        Api.async(() -> Api.get("/api/config"), config -> {
            Session.applyConfig(config);
            rebuildNav();
        }, failure -> ignoreFailure(failure));
    }

    private void ignoreFailure(Api.Failure failure) {
        // Branding is cosmetic; the shell already has usable defaults.
    }

    private void rebuildNav() {
        if (isFinishing() || isDestroyed()) return;
        refreshChrome();
    }

    private static String groupLabel(Nav.Group group) {
        switch (group) {
            case SYSTEM:
                return "系统";
            case PERSONAL:
                return "个人";
            case NETWORK:
            default:
                return "网络与解析";
        }
    }

    @SuppressWarnings("deprecation")
    @SuppressLint("WrongConstant")
    private void applySystemBars() {
        Palette p = Theme.p();
        getWindow().setBackgroundDrawable(new android.graphics.drawable.ColorDrawable(p.canvas));
        getWindow().setStatusBarColor(p.headerBg);
        getWindow().setNavigationBarColor(p.canvas);
        View decor = getWindow().getDecorView();
        int flags = decor.getSystemUiVisibility();
        if (p.dark) {
            flags &= ~View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR;
            flags &= ~View.SYSTEM_UI_FLAG_LIGHT_NAVIGATION_BAR;
        } else {
            flags |= View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR;
            flags |= View.SYSTEM_UI_FLAG_LIGHT_NAVIGATION_BAR;
        }
        decor.setSystemUiVisibility(flags);
    }

    private boolean isDetailRoute() {
        return currentPath != null && (currentPath.startsWith("/tunnels/")
                || currentPath.startsWith("/monitors/") || "/domain/batch".equals(currentPath));
    }

    private View reducedEffectsRow() {
        LinearLayout row = UI.row(this);
        row.setPadding(UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD));
        TextView caption = UI.body(this, "轻量效果");
        row.addView(caption, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        UI.Toggle toggle = new UI.Toggle(this, Theme.reducedEffects());
        toggle.setImportantForAccessibility(View.IMPORTANT_FOR_ACCESSIBILITY_NO);
        row.addView(toggle);
        row.setContentDescription("轻量效果，关闭模糊与动效");
        row.setClickable(true);
        row.setFocusable(true);
        row.setOnClickListener(view -> {
            Theme.setReducedEffects(this, !Theme.reducedEffects());
            toggle.setOn(Theme.reducedEffects());
            UI.retheme(shell, Theme.p());
            if (bottomBar != null) bottomBar.setElevation(UI.dp(Theme.reducedEffects() ? 0 : 6));
        });
        return row;
    }

    private void backToLogin(String notice) {
        Intent intent = new Intent(this, MainActivity.class);
        intent.addFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP | Intent.FLAG_ACTIVITY_NEW_TASK);
        if (notice != null) intent.putExtra(MainActivity.EXTRA_NOTICE, notice);
        startActivity(intent);
        finish();
    }
}
