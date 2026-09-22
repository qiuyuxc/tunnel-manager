package com.tunnelmanager.app;

import android.content.Intent;
import android.content.pm.PackageInfo;
import android.content.pm.PackageManager;
import android.graphics.Typeface;
import android.net.Uri;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.ImageView;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.appcompat.content.res.AppCompatResources;

import org.json.JSONObject;

import java.util.regex.Matcher;
import java.util.regex.Pattern;

public class AboutFragment extends PageFragment {

    private static final String REPO_URL = "https://github.com/qiuyuxc/tunnel-manager";
    private static final String DOCS_URL = "https://docs.kukie.cn";
    private static final String RELEASE_API =
            "https://api.github.com/repos/qiuyuxc/tunnel-manager/releases/latest";
    // The single-user edition tags releases vX.Y.Z-slim (GitHub pre-releases);
    // its update check scans the list and keeps only slim tags so a full-edition
    // release never looks like an update.
    private static final String RELEASES_API =
            "https://api.github.com/repos/qiuyuxc/tunnel-manager/releases?per_page=100";
    private static final String SLIM_SUFFIX = "-slim";

    /** Highlights of the current release, as the web page lists them. */
    private static final String[] HIGHLIGHTS = {
            "原生 App 换用石墨与薄荷绿配色，统一登录、面板、按钮和浮动导航，支持深浅主题与轻量效果",
            "更多菜单增加账户摘要和工具网格，常用入口一目了然",
            "隧道支持名称或 ID 搜索与状态筛选，当前选择、应用路由和危险操作分区更清楚",
            "全局新建、隧道列表和空状态共用创建弹层，提交区固定在底部，保留运行命令和令牌反馈",
            "监控列表增加真实七天可用率和服务历史条，未知、降级、异常与空数据分别展示",
            "监控详情增加可切换服务的响应趋势，添加、编辑、检测间隔和公开链接改为独立弹层",
            "关于页分别显示已安装 App 与服务端版本，避免旧服务端版本被误认为 App 版本",
    };

    private static final String[][] FEATURES = {
            {"隧道管理", "新建、删除、列出与选择 Cloudflare Tunnel，增删改应用路由（Ingress）并可连带清理 DNS"},
            {"域名绑定", "简化直连 / 优选模式，支持批量绑定，自动配置 Tunnel 路由与 DNS"},
            {"DNS 管理", "按 Zone 增删改查 A / AAAA / CNAME / TXT / MX 记录，支持多选批量修改与批量删除"},
            {"IP 优选实验室", "实验性直连探测 IP 段，按 Host / SNI 与状态码筛选，展示实时进度与分段命中，可一键剔除未命中段"},
            {"Cloudflare OAuth", "授权连接 Cloudflare 账户，支持多账户授权与随时切换，免去手动复制 API Token"},
            {"状态告警", "服务状态变化时自动推送告警（仅状态变化触发），支持邮件与 Telegram 通知，可视化配置并支持测试发送"},
            {"安全认证", "单管理员自托管，Argon2id 密码哈希、TOTP 双重验证、通行密钥与恢复码，密钥与令牌加密存储"},
    };

    private static final String[][] STACK = {
            {"Android", "Java · Android Views · AndroidX"},
            {"前端", "Vue 3 · TypeScript · Naive UI · Vite · Pinia"},
            {"后端", "Go · chi · SQLite"},
            {"集成", "Cloudflare API · Huawei Cloud DNS API · Telegram Bot API · GitHub Actions"},
    };

    private ScrollView scroll;
    private LinearLayout body;

    private String appVersion = "未知";
    private String serverVersion = "读取中…";
    private long appVersionCode;
    private String latestTag = "";
    private String latestBody = "";
    private String checkError = "";
    private boolean checking = false;
    private int generation;

    @Override
    String route() {
        return "/about";
    }

    @Override
    protected View build(@NonNull LayoutInflater inflater, @Nullable ViewGroup container) {
        readAppVersion();
        scroll = new ScrollView(requireContext());
        body = UI.column(requireContext());
        UI.pagePadding(body);
        scroll.addView(body);
        render();
        loadVersion();
        loadRelease();
        return scroll;
    }

    private void readAppVersion() {
        try {
            PackageInfo info = requireContext().getPackageManager().getPackageInfo(requireContext().getPackageName(), 0);
            appVersion = info.versionName == null ? "未知" : "v" + info.versionName;
            appVersionCode = android.os.Build.VERSION.SDK_INT >= 28 ? info.getLongVersionCode() : info.versionCode;
        } catch (PackageManager.NameNotFoundException failure) {
            appVersion = "未知";
            appVersionCode = 0;
        }
    }

    @Override public void onDestroyView() {
        generation++;
        checking = false;
        scroll = null;
        body = null;
        super.onDestroyView();
    }

    private void loadVersion() {
        int request = generation;
        Api.async(() -> Api.external(Session.server() + "/api/health", "application/json"), raw -> {
            if (request != generation || body == null) return;
            try {
                serverVersion = new JSONObject(raw).optString("version", "未知");
            } catch (Exception ignored) {
                serverVersion = "无法读取";
            }
            render();
        }, failure -> {
            if (request != generation || body == null) return;
            serverVersion = "无法连接";
            render();
        });
    }

    private void loadRelease() {
        if (checking) return;
        int request = generation;
        checking = true;
        checkError = "";
        latestTag = "";
        latestBody = "";
        render();
        boolean slim = appVersion != null && appVersion.toLowerCase().contains(SLIM_SUFFIX);
        Api.async(() -> {
            JSONObject out = new JSONObject();
            if (slim) {
                // Scan the release list, keep only slim tags, pick the highest.
                String raw = Api.external(RELEASES_API, "application/vnd.github+json");
                org.json.JSONArray list = new org.json.JSONArray(raw);
                String bestTag = "";
                String bestBody = "";
                for (int i = 0; i < list.length(); i++) {
                    JSONObject rel = list.optJSONObject(i);
                    if (rel == null) continue;
                    String tag = rel.optString("tag_name", "");
                    if (!tag.toLowerCase().contains(SLIM_SUFFIX)) continue;
                    if (bestTag.isEmpty() || compare(tag, bestTag) > 0) {
                        bestTag = tag;
                        bestBody = rel.optString("body", "");
                    }
                }
                out.put("tag", bestTag);
                out.put("body", bestBody);
            } else {
                String raw = Api.external(RELEASE_API, "application/vnd.github+json");
                JSONObject release = new JSONObject(raw);
                out.put("tag", release.optString("tag_name", ""));
                out.put("body", release.optString("body", ""));
            }
            return out;
        }, payload -> {
            if (request != generation || body == null) return;
            checking = false;
            latestTag = payload.optString("tag", "");
            latestBody = payload.optString("body", "");
            if (slim && latestTag.isEmpty()) checkError = "尚未发布精简版（slim）版本。";
            render();
        }, failure -> {
            if (request != generation || body == null) return;
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
        ImageView mark = new ImageView(requireContext());
        // logo_mark fills use ?attr theme colours; a plain ImageView loading it
        // via setImageResource does not resolve those attributes, so the paths
        // came out transparent and the previous SRC_IN tint then erased the mark
        // entirely (it rendered blank). AppCompatResources inflates the vector
        // against the context theme so the two-tone brand mark shows, matching
        // how the login screen renders the same drawable.
        mark.setImageDrawable(AppCompatResources.getDrawable(requireContext(), R.drawable.logo_mark));
        mark.setBackground(UI.rounded(Theme.p().canvasSoft2, UI.RADIUS_LG));
        int pad = UI.dp(UI.SM);
        mark.setPadding(pad, pad, pad, pad);
        card.addView(mark, new LinearLayout.LayoutParams(UI.dp(64), UI.dp(64)));
        UI.addRow(card, UI.text(requireContext(), "Tunnel Manager", 24, Theme.p().ink, Typeface.BOLD), UI.MD);
        UI.addRow(card, UI.muted(requireContext(), "连接本地，让服务自由抵达。"), UI.XS);
        UI.addRow(card, UI.muted(requireContext(), "Android 原生 · " + appVersion), UI.SM);
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
        card.addView(UI.cardTitle(requireContext(), "版本与更新"));
        UI.addRow(card, UI.muted(requireContext(), "App 版本来自当前安装包，服务端版本来自已连接的面板，两者分别更新。"), UI.XS);
        UI.addRow(card, versionRow("App 版本", appVersion, Theme.p().ink), UI.MD);
        card.addView(versionRow("构建号", appVersionCode == 0 ? "未知" : String.valueOf(appVersionCode), Theme.p().mute));
        card.addView(versionRow("服务端版本", serverVersion, Theme.p().ink));
        card.addView(versionRow("项目最新发布", latestTag.isEmpty() ? "尚未取得" : latestTag, Theme.p().ink));
        card.addView(versionRow("App 更新状态", versionStatus(appVersion), versionStatusColor(appVersion)));
        card.addView(versionRow("服务端更新状态", versionStatus(serverVersion), versionStatusColor(serverVersion)));
        TextView check = UI.button(requireContext(), checking ? "检查中…" : "检查更新", UI.BTN_SECONDARY);
        check.setEnabled(!checking);
        check.setOnClickListener(v -> loadRelease());
        UI.addRow(card, check, UI.MD);
        UI.addRow(card, UI.muted(requireContext(), "更新比较以项目发布标签为准，App 与服务端需分别更新：Android 安装包见发布页附件，服务端替换二进制并重启。"), UI.SM);
        return card;
    }

    private View versionRow(String label, String value, int color) {
        LinearLayout row = UI.row(requireContext());
        row.setPadding(0, UI.dp(UI.SM), 0, UI.dp(UI.SM));
        row.addView(UI.muted(requireContext(), label), new LinearLayout.LayoutParams(
                0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        TextView text = UI.text(requireContext(), value, 14, color, Typeface.BOLD);
        text.setGravity(android.view.Gravity.END);
        LinearLayout.LayoutParams valueLayout = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1.5f);
        valueLayout.leftMargin = UI.dp(UI.SM);
        row.addView(text, valueLayout);
        return row;
    }

    /** Summarises one installed version against the latest published release tag. */
    private String versionStatus(String installed) {
        if (checking) return "检查中…";
        if (latestTag.isEmpty()) return checkError.isEmpty() ? "尚未检查" : "检查失败";
        if (parse(latestTag) == null || parse(installed) == null) return "无法比较版本";
        int diff = compare(latestTag, installed);
        if (diff > 0) return "可更新到 " + latestTag;
        if (diff == 0) return "已是最新版本";
        return "领先线上发布";
    }

    private int versionStatusColor(String installed) {
        if (checking || !checkError.isEmpty() || parse(latestTag) == null || parse(installed) == null) return Theme.p().mute;
        int diff = compare(latestTag, installed);
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
        // find() not matches(): tolerate a suffix such as the -slim edition tag.
        if (!matcher.find()) return null;
        try {
            return new int[]{Integer.parseInt(matcher.group(1)), Integer.parseInt(matcher.group(2)), Integer.parseInt(matcher.group(3))};
        } catch (NumberFormatException failure) {
            return null;
        }
    }

    private View highlightsCard() {
        LinearLayout card = card("本版本亮点", "Android App " + appVersion + " 的重点变化。");
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
