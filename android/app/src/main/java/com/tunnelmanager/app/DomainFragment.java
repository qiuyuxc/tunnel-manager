package com.tunnelmanager.app;

import android.graphics.Typeface;
import android.text.InputType;
import android.text.TextUtils;
import android.view.Gravity;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.EditText;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout;

import org.json.JSONArray;
import org.json.JSONObject;

/**
 * 域名绑定 — point a domain at the locked tunnel.
 *
 * The web console puts the prerequisites, the summary and the form in three
 * stacked cards with a two column mode selector. On a phone the selector stays
 * two up (the labels are short), but the mode's extra fields are hidden rather
 * than greyed, so the common 简单模式 case fits a single screen.
 */
public class DomainFragment extends PageFragment {

    private SwipeRefreshLayout refresh;
    private LinearLayout body;

    private JSONObject config = new JSONObject();
    /** Index 0 = 简单模式, 1 = 优选模式; matches the web's `mode` values. */
    private int mode = 0;

    private EditText serviceUrl;
    private EditText cname;
    private EditText mainDomain;
    private EditText auxDomain;
    private LinearLayout cnameField;
    private LinearLayout auxField;

    /** Last bind outcome, kept across re-renders so it does not blink away. */
    private String resultMessage = "";
    private boolean resultOk = false;

    /**
     * The three domain fields are rebuilt whenever the result banner changes,
     * so what is typed is carried over instead of being wiped by a redraw.
     */
    private String draftCname = "";
    private String draftMain = "";
    private String draftAux = "";

    @Override
    String route() {
        return "/domain";
    }

    @Override
    protected View build(@NonNull LayoutInflater inflater, @Nullable ViewGroup container) {
        refresh = new SwipeRefreshLayout(requireContext());
        refresh.setColorSchemeColors(Theme.p().ink);
        refresh.setProgressBackgroundColorSchemeColor(Theme.p().canvasRaised);
        refresh.setOnRefreshListener(this::load);

        ScrollView scroll = new ScrollView(requireContext());
        body = UI.column(requireContext());
        UI.pagePadding(body);
        scroll.addView(body);
        refresh.addView(scroll);

        body.addView(UI.muted(requireContext(), "加载中…"));
        load();
        return refresh;
    }

    private void load() {
        Api.async(() -> Api.get("/api/config"), loaded -> {
            refresh.setRefreshing(false);
            config = loaded;
            // A refresh is the operator asking the server again, so the drafts
            // go with it; the fields rebuild from the fetched config.
            draftCname = "";
            draftMain = "";
            draftAux = "";
            render();
        }, failure -> {
            refresh.setRefreshing(false);
            renderFailure(failure.getMessage());
        });
    }

    // --------------------------------------------------------------- rendering

    private void renderFailure(String message) {
        if (body == null || !alive()) return;
        body.removeAllViews();
        body.addView(UI.banner(requireContext(), message == null ? "加载失败" : message, true));
    }

    private void render() {
        if (body == null || !alive()) return;
        body.removeAllViews();

        body.addView(UI.pageTitle(requireContext(), "域名绑定"));

        TextView subtitle = UI.muted(requireContext(), "将域名绑定到已配置的隧道，自动配置 DNS 和 SaaS 回源");
        UI.margin(subtitle, 0, UI.XS, 0, UI.MD);
        body.addView(subtitle);

        String tunnelId = config.optString("tunnel_id", "");
        String serviceUrlValue = config.optString("service_url", "");
        if (tunnelId.isEmpty() || serviceUrlValue.isEmpty()) {
            body.addView(UI.banner(requireContext(), "前置条件未满足：请先在「隧道管理」锁定隧道，并配置转发地址。", true));
            body.addView(UI.spacer(requireContext(), UI.LG));
        }

        body.addView(summaryCard(tunnelId, serviceUrlValue));

        body.addView(UI.spacer(requireContext(), UI.XL));
        body.addView(UI.label(requireContext(), "绑定新域名"));
        body.addView(UI.spacer(requireContext(), UI.SM));
        body.addView(bindCard());

        if (!resultMessage.isEmpty()) {
            TextView banner = UI.banner(requireContext(), resultMessage, !resultOk);
            UI.margin(banner, 0, UI.LG, 0, 0);
            body.addView(banner);
        }
    }

    private View summaryCard(String tunnelId, String serviceUrlValue) {
        Palette p = Theme.p();
        LinearLayout card = UI.card(requireContext());

        TextView tunnelLabel = UI.label(requireContext(), "当前隧道");
        card.addView(tunnelLabel);
        TextView tunnel = tunnelId.isEmpty()
                ? UI.muted(requireContext(), "未配置")
                : UI.text(requireContext(), config.optString("tunnel_name", "已选隧道"), 14, p.ink, Typeface.BOLD);
        UI.margin(tunnel, 0, UI.XS, 0, UI.MD);
        card.addView(tunnel);

        card.addView(UI.label(requireContext(), "转发地址"));
        serviceUrl = UI.input(requireContext(), "http://localhost:3000");
        serviceUrl.setInputType(InputType.TYPE_TEXT_VARIATION_URI);
        serviceUrl.setText(serviceUrlValue);
        UI.margin(serviceUrl, 0, UI.XS, 0, 0);
        card.addView(serviceUrl);

        TextView save = UI.button(requireContext(), "保存转发地址", UI.BTN_SECONDARY);
        UI.fill(save);
        UI.margin(save, 0, UI.SM, 0, UI.MD);
        save.setOnClickListener(v -> saveServiceUrl());
        card.addView(save);

        card.addView(UI.label(requireContext(), "默认 CNAME"));
        TextView preferred = UI.mono(requireContext(), config.optString("preferred_cname", "—"), p.body);
        preferred.setSingleLine(true);
        preferred.setEllipsize(TextUtils.TruncateAt.MIDDLE);
        UI.margin(preferred, 0, UI.XS, 0, 0);
        card.addView(preferred);
        return card;
    }

    private View bindCard() {
        LinearLayout card = UI.card(requireContext());

        card.addView(UI.segmented(requireContext(), new String[]{"简单模式", "优选模式"}, mode, selected -> {
            mode = selected;
            cnameField.setVisibility(selected == 1 ? View.VISIBLE : View.GONE);
            auxField.setVisibility(selected == 1 ? View.VISIBLE : View.GONE);
        }));

        cname = UI.input(requireContext(), "留空使用默认线路");
        cname.setText(draftCname);
        TextView pick = UI.button(requireContext(), "常用线路", UI.BTN_SECONDARY);
        pick.setOnClickListener(v -> openCnamePicker());

        LinearLayout cnameRow = UI.row(requireContext());
        cnameRow.addView(cname, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        LinearLayout.LayoutParams pickLp = new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);
        pickLp.leftMargin = UI.dp(UI.SM);
        cnameRow.addView(pick, pickLp);

        cnameField = UI.field(requireContext(), "本次优选 CNAME", cnameRow, UI.MD);
        card.addView(cnameField);

        mainDomain = UI.input(requireContext(), "例如：kukie.cn");
        mainDomain.setInputType(InputType.TYPE_TEXT_VARIATION_URI);
        mainDomain.setText(draftMain);
        card.addView(UI.field(requireContext(), "主域名 · 对外访问域名", mainDomain, UI.MD));

        auxDomain = UI.input(requireContext(), "例如：fallback.169977.xyz");
        auxDomain.setInputType(InputType.TYPE_TEXT_VARIATION_URI);
        auxDomain.setText(draftAux);
        auxField = UI.field(requireContext(), "辅助域名 · 用作回源", auxDomain, UI.MD);
        card.addView(auxField);

        cnameField.setVisibility(mode == 1 ? View.VISIBLE : View.GONE);
        auxField.setVisibility(mode == 1 ? View.VISIBLE : View.GONE);

        TextView bind = UI.button(requireContext(), "绑定域名", UI.BTN_PRIMARY);
        UI.fill(bind);
        UI.margin(bind, 0, UI.LG, 0, 0);
        bind.setOnClickListener(v -> bind());
        card.addView(bind);

        TextView batch = UI.button(requireContext(), "批量绑定多组域名", UI.BTN_SECONDARY);
        UI.fill(batch);
        UI.margin(batch, 0, UI.SM, 0, 0);
        batch.setOnClickListener(v -> openRoute("/domain/batch"));
        card.addView(batch);
        return card;
    }

    // ------------------------------------------------------------------ actions

    /** The console's CNAMEPicker, flattened into a sheet. */
    private void openCnamePicker() {
        String fallback = config.optString("preferred_cname", "");
        Sheet.Builder sheet = Sheet.of(requireContext(), "本次优选 CNAME")
                .label("常用线路");

        sheet.item(R.drawable.ic_nav_check,
                fallback.isEmpty() ? "使用默认配置" : "使用默认配置（" + fallback + "）",
                () -> cname.setText(""));

        JSONArray presets = config.optJSONArray("cname_presets");
        if (presets != null) {
            for (int i = 0; i < presets.length(); i++) {
                JSONObject preset = presets.optJSONObject(i);
                if (preset == null) continue;
                String value = preset.optString("value", "");
                if (value.isEmpty()) continue;
                String name = preset.optString("name", value);
                sheet.item(R.drawable.ic_nav_domain, name + " · " + value, () -> cname.setText(value));
            }
        }
        sheet.show();
    }

    private void saveServiceUrl() {
        String value = serviceUrl.getText().toString().trim();
        if (value.isEmpty()) {
            toast("转发地址不能为空");
            return;
        }
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("value", value);
            return Api.post("/api/config/service", payload);
        }, ok -> {
            config.remove("service_url");
            try {
                config.put("service_url", value);
            } catch (Exception ignored) {
            }
            toast("转发地址已更新");
        }, failure -> toast(failure.getMessage()));
    }

    private void bind() {
        final String service = serviceUrl.getText().toString().trim();
        final String main = mainDomain.getText().toString().trim();
        final String aux = auxDomain.getText().toString().trim();
        final String preferred = cname.getText().toString().trim();

        if (service.isEmpty() || main.isEmpty() || (mode == 1 && aux.isEmpty())) {
            setResult(false, mode == 1
                    ? "转发地址、主域名、辅助域名都要填"
                    : "转发地址和主域名都要填");
            return;
        }

        final String previous = config.optString("service_url", "").trim();
        Api.async(() -> {
            // The web console saves a changed forwarding address as part of the
            // same click, so the two never disagree about what was bound.
            if (!service.equals(previous)) {
                JSONObject servicePayload = new JSONObject();
                servicePayload.put("value", service);
                Api.post("/api/config/service", servicePayload);
            }
            JSONObject payload = new JSONObject();
            payload.put("mode", mode == 1 ? "preferred" : "simple");
            payload.put("preferred_cname", preferred);
            payload.put("main_domain", main);
            payload.put("aux_domain", aux);
            return Api.post("/api/domain/bind", payload);
        }, response -> {
            config.remove("service_url");
            try {
                config.put("service_url", service);
            } catch (Exception ignored) {
            }
            setResult(true, response.optString("message", "域名绑定成功"));
        }, failure -> setResult(false, failure.getMessage()));
    }

    private void setResult(boolean ok, String message) {
        resultOk = ok;
        resultMessage = message == null ? "" : message;
        captureDraft();
        render();
        toast(ok ? "绑定成功" : resultMessage);
    }

    private void captureDraft() {
        if (cname == null) return;
        draftCname = cname.getText().toString();
        draftMain = mainDomain.getText().toString();
        draftAux = auxDomain.getText().toString();
    }

    private void toast(String message) {
        if (message == null || message.isEmpty() || !alive()) return;
        android.widget.Toast.makeText(requireContext(), message, android.widget.Toast.LENGTH_LONG).show();
    }
}
