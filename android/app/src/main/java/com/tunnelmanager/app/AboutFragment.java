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

    /** Highlights of the current release, as the web page lists them. */
    private final String[] HIGHLIGHTS = {
            I18n.text("m_1ae15c6bd1be"),
            I18n.text("m_251a77247045"),
            I18n.text("m_4c2f23822e3d"),
            I18n.text("m_19dff4cfa946"),
            I18n.text("m_3c816b7f18b2"),
            I18n.text("m_c046ce1f1e92"),
    };

    private final String[][] FEATURES = {
            {I18n.text("m_e920f47d0c05"), I18n.text("m_ff534389da66")},
            {I18n.text("m_4bf8b66d7390"), I18n.text("m_b3c3831a9225")},
            {I18n.text("m_474fb6002594"), I18n.text("m_9fbdbc4caeba")},
            {I18n.text("m_1b0049fcfd41"), I18n.text("m_23efa07a6b81")},
            {I18n.text("m_3db29474a184"), I18n.text("m_165170c6d7f2")},
            {"Telegram Bot", I18n.text("m_d2acf85ae520")},
            {"Cloudflare OAuth", I18n.text("m_20c9a5fcb58a")},
            {I18n.text("m_f6a93d54ba44"), I18n.text("m_4d4267727699")},
            {I18n.text("m_384a5d60f314"), I18n.text("m_d33996c19397")},
            {I18n.text("m_53b57f1e9171"), I18n.text("m_8716134bee6b")},
    };

    private final String[][] STACK = {
            {"Android", "Java · Android Views · AndroidX"},
            {I18n.text("m_866a4fa82c25"), "Vue 3 · TypeScript · Naive UI · Vite · Pinia"},
            {I18n.text("m_9b826346239e"), "Go · chi · SQLite"},
            {I18n.text("m_4115d63bd549"), "Cloudflare API · Huawei Cloud DNS API · Telegram Bot API · GitHub Actions"},
    };

    private ScrollView scroll;
    private LinearLayout body;

    private String appVersion = I18n.text("m_4d8c1c5b4283");
    private String serverVersion = I18n.text("m_346ff60e6c7c");
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
            appVersion = info.versionName == null ? I18n.text("m_4d8c1c5b4283") : "v" + info.versionName;
            appVersionCode = android.os.Build.VERSION.SDK_INT >= 28 ? info.getLongVersionCode() : info.versionCode;
        } catch (PackageManager.NameNotFoundException failure) {
            appVersion = I18n.text("m_4d8c1c5b4283");
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
                serverVersion = new JSONObject(raw).optString("version", I18n.text("m_4d8c1c5b4283"));
            } catch (Exception ignored) {
                serverVersion = I18n.text("m_c27041585556");
            }
            render();
        }, failure -> {
            if (request != generation || body == null) return;
            serverVersion = I18n.text("m_6340b951ecd3");
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
        Api.async(() -> {
            String raw = Api.external(RELEASE_API, "application/vnd.github+json");
            JSONObject release = new JSONObject(raw);
            JSONObject out = new JSONObject();
            out.put("tag", release.optString("tag_name", ""));
            out.put("body", release.optString("body", ""));
            return out;
        }, payload -> {
            if (request != generation || body == null) return;
            checking = false;
            latestTag = payload.optString("tag", "");
            latestBody = payload.optString("body", "");
            render();
        }, failure -> {
            if (request != generation || body == null) return;
            checking = false;
            checkError = I18n.text("m_ec5a2ccad49c");
            render();
        });
    }

    // -------------------------------------------------------------- rendering

    private void render() {
        if (body == null || !alive()) return;
        int keepScroll = scroll.getScrollY();
        body.removeAllViews();

        body.addView(UI.pageTitle(requireContext(), I18n.text("m_52d25a9e30ba")));
        TextView subtitle = UI.muted(requireContext(), I18n.text("m_5f96499d539c"));
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
        // logo_mark is a two-tone vector whose fills use ?attr theme colours.
        // AppCompatResources inflates it against the context theme so the paths
        // resolve; a plain setImageResource + SRC_IN tint erased the mark
        // (rendering blank), so no tint is applied here.
        mark.setImageDrawable(AppCompatResources.getDrawable(requireContext(), R.drawable.logo_mark));
        mark.setBackground(UI.rounded(Theme.p().canvasSoft2, UI.RADIUS_LG));
        card.addView(mark, new LinearLayout.LayoutParams(UI.dp(64), UI.dp(64)));
        UI.addRow(card, UI.text(requireContext(), "Tunnel Manager", 24, Theme.p().ink, Typeface.BOLD), UI.MD);
        UI.addRow(card, UI.muted(requireContext(), I18n.text("m_4ba7340afe15")), UI.XS);
        UI.addRow(card, UI.muted(requireContext(), I18n.text("m_84f41bb438c5") + appVersion), UI.SM);
        card.addView(linkRow(I18n.text("m_e6161ef37a62"), REPO_URL));
        card.addView(linkRow(I18n.text("m_1c4b87b88528"), DOCS_URL));
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
        card.addView(UI.cardTitle(requireContext(), I18n.text("m_cebbc1f61ca0")));
        UI.addRow(card, UI.muted(requireContext(), I18n.text("m_988b7bf951b5")), UI.XS);
        UI.addRow(card, versionRow(I18n.text("m_b40c58015bd3"), appVersion, Theme.p().ink), UI.MD);
        card.addView(versionRow(I18n.text("m_74589c24cada"), appVersionCode == 0 ? I18n.text("m_4d8c1c5b4283") : String.valueOf(appVersionCode), Theme.p().mute));
        card.addView(versionRow(I18n.text("m_4e87ee56bf2b"), serverVersion, Theme.p().ink));
        card.addView(versionRow(I18n.text("m_ca0bbfb59e95"), latestTag.isEmpty() ? I18n.text("m_43240f4f30b8") : latestTag, Theme.p().ink));
        card.addView(versionRow(I18n.text("m_5d487e656329"), versionStatus(appVersion), versionStatusColor(appVersion)));
        card.addView(versionRow(I18n.text("m_c7b8f4d7fd6d"), versionStatus(serverVersion), versionStatusColor(serverVersion)));
        TextView check = UI.button(requireContext(), checking ? I18n.text("m_fb11aa6f2982") : I18n.text("m_7f68ebad19ba"), UI.BTN_SECONDARY);
        check.setEnabled(!checking);
        check.setOnClickListener(v -> loadRelease());
        UI.addRow(card, check, UI.MD);
        UI.addRow(card, UI.muted(requireContext(), I18n.text("m_de8a4204c612")), UI.SM);
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
        if (checking) return I18n.text("m_fb11aa6f2982");
        if (latestTag.isEmpty()) return checkError.isEmpty() ? I18n.text("m_cdbc0434f568") : I18n.text("m_4f6f06254d16");
        if (parse(latestTag) == null || parse(installed) == null) return I18n.text("m_73b8bd534007");
        int diff = compare(latestTag, installed);
        if (diff > 0) return I18n.text("m_3effbd8332c9") + latestTag;
        if (diff == 0) return I18n.text("m_bc310480b4ce");
        return I18n.text("m_3886917af351");
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
        if (!matcher.matches()) return null;
        try {
            return new int[]{Integer.parseInt(matcher.group(1)), Integer.parseInt(matcher.group(2)), Integer.parseInt(matcher.group(3))};
        } catch (NumberFormatException failure) {
            return null;
        }
    }

    private View highlightsCard() {
        LinearLayout card = card(I18n.text("m_42c169ac53c2"), "Android App " + appVersion + I18n.text("m_a8ec2cc5f780"));
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
        card.addView(UI.cardTitle(requireContext(), I18n.text("m_25b119e0eede")));
        TextView desc = UI.muted(requireContext(),
                latestTag.isEmpty() ? I18n.text("m_3c25c74af188") : I18n.text("m_fbd5949788bc") + latestTag);
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
        LinearLayout card = card(I18n.text("m_9351fb06ea54"), I18n.text("m_0e22df176d79"));
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
        LinearLayout card = card(I18n.text("m_265b45019e65"), I18n.text("m_3d9b89277530"));
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
            android.widget.Toast.makeText(requireContext(), I18n.text("m_7f46936a74b7"), android.widget.Toast.LENGTH_SHORT).show();
        }
    }
}
