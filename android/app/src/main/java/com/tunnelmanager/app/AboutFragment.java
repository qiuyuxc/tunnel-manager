package com.tunnelmanager.app;

import android.content.Intent;
import android.graphics.Typeface;
import android.net.Uri;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import org.json.JSONObject;

import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * 关于 — what this is, which version runs, and what changed.
 *
 * The version check hits GitHub directly rather than through the console,
 * because the question it answers ("is there something newer than my server?")
 * only makes sense against the upstream release feed.
 */
public class AboutFragment extends PageFragment {

    private static final String REPO_URL = "https://github.com/qiuyuxc/tunnel-manager";
    private static final String DOCS_URL = "https://docs.kukie.cn";
    private static final String RELEASE_API =
            "https://api.github.com/repos/qiuyuxc/tunnel-manager/releases/latest";

    /** Highlights of the current release, as the web page lists them. */
    private static final String[] HIGHLIGHTS = {
            "新增 Android 原生 App：概览、监控、隧道绑定、DNS、IP 优选实验室与通知全部原生实现",
            "App 通知接入系统通知渠道，不打开网页也能在手机上收到监控状态变化",
            "App 登录页内嵌 Cloudflare Turnstile，人机验证全程在原生页面完成，不再跳转网页",
            "登录页与控制台跟随主题偏好切换亮色 / 暗色，与网页端共用一套配色",
            "移动端改为底部标签导航与半屏弹窗，窄屏下监控、DNS 等页面不再横向挤压",
            "落地页双主题：Vercel 风格（企业蓝）与 Claude 风格（暖色），各含亮色 / 暗色",
            "错误信息统一脱敏，Telegram Bot Token 不再出现在日志、状态卡与截图中",
            "新增 /api/alerts 告警游标接口，客户端可按 since 增量拉取监控状态变化",
    };

    private static final String[][] FEATURES = {
            {"隧道管理", "新建、删除、列出与选择 Cloudflare Tunnel，增删改应用路由（Ingress）并可连带清理 DNS"},
            {"域名绑定", "简化直连 / 优选模式，支持批量绑定，自动配置 Tunnel 路由与 DNS"},
            {"DNS 管理", "按 Zone 增删改查 A / AAAA / CNAME / TXT / MX 记录，支持多选批量修改与批量删除"},
            {"IP 优选实验室", "实验性直连探测 IP 段，按 Host / SNI 与状态码筛选，展示实时进度与分段命中，可一键剔除未命中段"},
            {"Telegram Bot", "在手机上远程管理隧道、域名与 DNS，支持长轮询与 Webhook"},
            {"Cloudflare OAuth", "授权连接 Cloudflare 账户，支持多账户授权与随时切换，免去手动复制 API Token"},
            {"多用户与管理后台", "邮箱注册与用户组权限隔离，管理员统一管理用户、邀请码与注册策略"},
            {"邮件告警", "服务状态变化时自动发送告警邮件（仅状态变化触发），SMTP 可视化配置并支持测试发送"},
            {"安全认证", "Argon2id 密码哈希、TOTP 双重验证与恢复码，密钥与令牌加密存储"},
    };

    private static final String[][] STACK = {
            {"前端", "Vue 3 · TypeScript · Naive UI · Vite · Pinia"},
            {"后端", "Go · chi · SQLite"},
            {"集成", "Cloudflare API · Huawei Cloud DNS API · Telegram Bot API · GitHub Actions"},
    };

    private ScrollView scroll;
    private LinearLayout body;

    private String currentVersion = "—";
    private String latestTag = "";
    private String latestBody = "";
    private String checkError = "";
    private boolean checking = false;

    @Override
    String route() {
        return "/about";
    }

    @Override
    protected View build(@NonNull LayoutInflater inflater, @Nullable ViewGroup container) {
        scroll = new ScrollView(requireContext());
        body = UI.column(requireContext());
        UI.pagePadding(body);
        scroll.addView(body);
        render();
        loadVersion();
        loadRelease();
        return scroll;
    }

    private void loadVersion() {
        Api.async(() -> Api.external(Session.server() + "/api/health", "application/json"), raw -> {
            try {
                currentVersion = new JSONObject(raw).optString("version", "—");
            } catch (Exception ignored) {
                currentVersion = "—";
            }
            render();
        }, failure -> ignore());
    }

    private void ignore() {
        // The version row already shows a dash; a health failure is not news.
    }

    private void loadRelease() {
        checking = true;
        checkError = "";
        render();
        Api.async(() -> {
            String raw = Api.external(RELEASE_API, "application/vnd.github+json");
            JSONObject release = new JSONObject(raw);
            JSONObject out = new JSONObject();
            out.put("tag", release.optString("tag_name", ""));
            out.put("body", release.optString("body", ""));
            return out;
        }, payload -> {
            checking = false;
            latestTag = payload.optString("tag", "");
            latestBody = payload.optString("body", "");
            render();
        }, failure -> {
            checking = false;
            checkError = "无法连接 GitHub，请检查网络后重试。";
            render();
        });
    }

    // -------------------------------------------------------------- rendering

    private void render() {
        if (body == null || !alive()) return;
        int keepScroll = scroll.getScrollY();
        body.removeAllViews();

        body.addView(UI.pageTitle(requireContext(), "关于"));
        TextView subtitle = UI.muted(requireContext(), "版本信息、项目仓库与更新动态");
        UI.margin(subtitle, 0, UI.XS, 0, UI.MD);
        body.addView(subtitle);

        body.addView(appCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(versionCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(highlightsCard());
        if (!latestBody.isEmpty() || !checkError.isEmpty()) {
            body.addView(UI.spacer(requireContext(), UI.MD));
            body.addView(releaseCard());
        }
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(featuresCard());
        body.addView(UI.spacer(requireContext(), UI.MD));
        body.addView(stackCard());
        UI.restoreScroll(scroll, keepScroll);
    }

    private LinearLayout card(String title, String desc) {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), title));
        if (desc != null && !desc.isEmpty()) {
            TextView hint = UI.muted(requireContext(), desc);
            UI.margin(hint, 0, UI.XS, 0, UI.MD);
            card.addView(hint);
        }
        return card;
    }

    private View appCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), "Tunnel Manager"));
        TextView desc = UI.muted(requireContext(), "Cloudflare Tunnel 可视化管理面板");
        UI.margin(desc, 0, UI.XS, 0, UI.MD);
        card.addView(desc);

        TextView about = UI.body(requireContext(),
                "通过 Web UI 管理隧道、绑定域名、配置 DNS 优选与回退源，支持多用户注册与管理后台、"
                        + "服务状态变化邮件告警、Telegram Bot 远程管理和双重身份验证；管理员还可开启实验性 IP 优选实验室，"
                        + "按输入段探测、筛选并剔除无命中地址。");
        card.addView(about);

        card.addView(linkRow("仓库地址", REPO_URL));
        card.addView(linkRow("在线文档", DOCS_URL));
        return card;
    }

    private View linkRow(String label, String url) {
        Palette p = Theme.p();
        LinearLayout row = UI.row(requireContext());
        row.setLayoutParams(new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));
        row.setBackground(UI.pressable(UI.rounded(p.canvasSoft2, UI.RADIUS_MD), p.btnGhostHover));
        row.setPadding(UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD));
        row.addView(UI.strong(requireContext(), label));
        TextView value = UI.text(requireContext(), url, 13, p.link, Typeface.NORMAL);
        LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
        lp.leftMargin = UI.dp(UI.SM);
        value.setEllipsize(android.text.TextUtils.TruncateAt.MIDDLE);
        value.setSingleLine(true);
        row.addView(value, lp);
        row.setClickable(true);
        row.setOnClickListener(v -> open(url));
        UI.margin(row, 0, UI.MD, 0, 0);
        return row;
    }

    private View versionCard() {
        LinearLayout card = UI.card(requireContext());
        LinearLayout head = UI.row(requireContext());
        LinearLayout text = UI.column(requireContext());
        text.addView(UI.cardTitle(requireContext(), "版本与更新"));
        TextView desc = UI.muted(requireContext(), "对比 GitHub 上发布的最新版本，查看是否有新内容。");
        UI.margin(desc, 0, UI.XS, 0, 0);
        text.addView(desc);
        head.addView(text, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        TextView check = UI.button(requireContext(), checking ? "检查中…" : "检查更新", UI.BTN_PRIMARY);
        check.setEnabled(!checking);
        check.setOnClickListener(v -> loadRelease());
        head.addView(check);
        card.addView(head);

        card.addView(UI.spacer(requireContext(), UI.MD));
        card.addView(versionRow("当前版本", currentVersion, Theme.p().ink));
        String latest = latestTag.isEmpty() ? "—" : latestTag;
        card.addView(versionRow("线上最新", latest, Theme.p().ink));
        card.addView(versionRow("更新状态", statusText(), statusColor()));
        return card;
    }

    private View versionRow(String label, String value, int color) {
        LinearLayout row = UI.row(requireContext());
        row.setPadding(0, UI.dp(UI.SM), 0, UI.dp(UI.SM));
        row.addView(UI.muted(requireContext(), label),
                new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        TextView text = UI.text(requireContext(), value, 14, color, Typeface.BOLD);
        text.setGravity(android.view.Gravity.END);
        row.addView(text);
        return row;
    }

    private String statusText() {
        if (checking) return "检查中…";
        if (latestTag.isEmpty()) return checkError.isEmpty() ? "尚未检查" : "检查失败";
        int diff = compare(latestTag, currentVersion);
        if (diff > 0) return "发现新版本 " + latestTag;
        if (diff == 0) return "已是最新版本";
        return "当前版本领先线上发布";
    }

    private int statusColor() {
        if (latestTag.isEmpty()) return Theme.p().mute;
        int diff = compare(latestTag, currentVersion);
        if (diff > 0) return Theme.p().statusDegradedText;
        if (diff == 0) return Theme.p().statusHealthyText;
        return Theme.p().link;
    }

    /** Compares two vX.Y.Z tags; missing or malformed tags compare as 0. */
    private static int compare(String a, String b) {
        int[] left = parse(a);
        int[] right = parse(b);
        if (left == null || right == null) return 0;
        for (int i = 0; i < 3; i++) {
            if (left[i] != right[i]) return left[i] < right[i] ? -1 : 1;
        }
        return 0;
    }

    private static int[] parse(String tag) {
        Matcher matcher = Pattern.compile("v?(\\d+)\\.(\\d+)\\.(\\d+)").matcher(tag == null ? "" : tag);
        if (!matcher.find()) return null;
        return new int[]{
                Integer.parseInt(matcher.group(1)),
                Integer.parseInt(matcher.group(2)),
                Integer.parseInt(matcher.group(3)),
        };
    }

    private View highlightsCard() {
        LinearLayout card = card("本版本亮点", "当前版本（" + currentVersion + "）的重点变化。");
        for (String item : HIGHLIGHTS) {
            card.addView(bullet(item, 0));
        }
        return card;
    }

    private View bullet(String item, int topDp) {
        LinearLayout row = UI.row(requireContext());
        row.setGravity(android.view.Gravity.TOP);
        View dot = new View(requireContext());
        dot.setBackground(UI.circle(Theme.p().ink, 0, 0));
        LinearLayout.LayoutParams dotLp = new LinearLayout.LayoutParams(UI.dp(5), UI.dp(5));
        dotLp.topMargin = UI.dp(7);
        dotLp.rightMargin = UI.dp(UI.MD);
        row.addView(dot, dotLp);
        TextView text = UI.body(requireContext(), item);
        row.addView(text, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        UI.margin(row, 0, topDp, 0, 0);
        return row;
    }

    private View releaseCard() {
        LinearLayout card = UI.card(requireContext());
        card.addView(UI.cardTitle(requireContext(), "最新发布更新内容"));
        TextView desc = UI.muted(requireContext(),
                latestTag.isEmpty() ? "来自 GitHub Release。" : "来自 GitHub Release · " + latestTag);
        UI.margin(desc, 0, UI.XS, 0, UI.MD);
        card.addView(desc);

        if (latestBody.isEmpty()) {
            card.addView(UI.banner(requireContext(), checkError, true));
            return card;
        }
        TextView text = UI.text(requireContext(), plainMarkdown(latestBody), 13, Theme.p().body, Typeface.NORMAL);
        text.setLineSpacing(UI.dp(4), 1f);
        text.setTextIsSelectable(true);
        card.addView(text);
        return card;
    }

    /**
     * The release body is Markdown. Rendering it properly would mean shipping a
     * parser for one screen, so the syntax that carries meaning in this context
     * (headings, bullets, emphasis, links) is flattened to plain text instead.
     */
    private static String plainMarkdown(String body) {
        String text = body.length() > 20000 ? body.substring(0, 20000) : body;
        text = text.replaceAll("(?m)^#{1,6}\\s*", "");
        text = text.replaceAll("(?m)^\\s*[-*+]\\s+", "· ");
        text = text.replaceAll("\\*\\*(.+?)\\*\\*", "$1");
        text = text.replaceAll("`([^`]*)`", "$1");
        text = text.replaceAll("!\\[[^\\]]*\\]\\([^)]*\\)", "");
        text = text.replaceAll("\\[([^\\]]*)\\]\\(([^)]*)\\)", "$1（$2）");
        return text.trim();
    }

    private View featuresCard() {
        LinearLayout card = card("功能特性", "主要能力一览。");
        for (String[] feature : FEATURES) {
            LinearLayout box = UI.column(requireContext());
            box.setBackground(UI.roundedStroke(Theme.p().canvasSoft2, UI.RADIUS_MD, Theme.p().hairline, 1));
            box.setPadding(UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD));
            box.addView(UI.strong(requireContext(), feature[0]));
            TextView desc = UI.muted(requireContext(), feature[1]);
            UI.margin(desc, 0, UI.XS, 0, 0);
            box.addView(desc);
            UI.margin(box, 0, UI.SM, 0, 0);
            card.addView(box);
        }
        return card;
    }

    private View stackCard() {
        LinearLayout card = card("技术栈", "构建本项目使用的技术。");
        for (String[] row : STACK) {
            LinearLayout line = UI.row(requireContext());
            line.addView(UI.muted(requireContext(), row[0]),
                    new LinearLayout.LayoutParams(UI.dp(64), ViewGroup.LayoutParams.WRAP_CONTENT));
            TextView value = UI.mono(requireContext(), row[1], Theme.p().ink);
            value.setSingleLine(false);
            line.addView(value, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
            UI.margin(line, 0, UI.SM, 0, 0);
            card.addView(line);
        }
        return card;
    }

    private void open(String url) {
        try {
            startActivity(new Intent(Intent.ACTION_VIEW, Uri.parse(url)));
        } catch (Exception e) {
            android.widget.Toast.makeText(requireContext(), "没有可用的浏览器", android.widget.Toast.LENGTH_SHORT).show();
        }
    }
}
