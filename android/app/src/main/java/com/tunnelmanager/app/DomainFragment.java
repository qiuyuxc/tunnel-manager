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
    private boolean pending;
    private String pendingTitle = "";
    private long startedAt;
    private String draftService;

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

        body.addView(UI.muted(requireContext(), I18n.text("m_4927a53bcc88")));
        load();
        return refresh;
    }

    private void load() {
        if (pending) { refresh.setRefreshing(false); render(); return; }
        Api.async(() -> Api.get("/api/config"), loaded -> {
            refresh.setRefreshing(false);
            config = loaded;
            // A refresh is the operator asking the server again, so the drafts
            // go with it; the fields rebuild from the fetched config.
            draftCname = "";
            draftMain = "";
            draftAux = "";
            draftService = null;
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
        body.addView(UI.banner(requireContext(), message == null ? I18n.text("m_d1d044826a45") : message, true));
    }

    private void render() {
        if (body == null || !alive()) return;
        body.removeAllViews();

        body.addView(UI.pageTitle(requireContext(), I18n.text("m_4bf8b66d7390")));

        TextView subtitle = UI.muted(requireContext(), I18n.text("m_de782b9792c7"));
        UI.margin(subtitle, 0, UI.XS, 0, UI.MD);
        body.addView(subtitle);

        String tunnelId = config.optString("tunnel_id", "");
        String serviceUrlValue = draftService == null ? config.optString("service_url", "") : draftService;
        if (tunnelId.isEmpty() || serviceUrlValue.isEmpty()) {
            body.addView(UI.banner(requireContext(), I18n.text("m_87d788a3e1be"), true));
            body.addView(UI.spacer(requireContext(), UI.LG));
        }

        body.addView(summaryCard(tunnelId, serviceUrlValue));

        body.addView(UI.spacer(requireContext(), UI.XL));
        body.addView(UI.label(requireContext(), I18n.text("m_bce4481a7278")));
        body.addView(UI.spacer(requireContext(), UI.SM));
        body.addView(bindCard());
        refresh.setEnabled(!pending);
        if (pending) {
            OperationProgress.setInputsEnabled(body, false);
        }

        if (!resultMessage.isEmpty()) {
            TextView banner = UI.banner(requireContext(), resultMessage, !resultOk);
            UI.margin(banner, 0, UI.LG, 0, 0);
            body.addView(banner);
        }
    }

    private View summaryCard(String tunnelId, String serviceUrlValue) {
        Palette p = Theme.p();
        LinearLayout card = UI.card(requireContext());

        TextView tunnelLabel = UI.label(requireContext(), I18n.text("m_b903d203f70a"));
        card.addView(tunnelLabel);
        TextView tunnel = tunnelId.isEmpty()
                ? UI.muted(requireContext(), I18n.text("m_80a57e03f071"))
                : UI.text(requireContext(), config.optString("tunnel_name", I18n.text("m_8ce5320404d5")), 14, p.ink, Typeface.BOLD);
        UI.margin(tunnel, 0, UI.XS, 0, UI.MD);
        card.addView(tunnel);

        card.addView(UI.label(requireContext(), I18n.text("m_c762fbaaad6f")));
        serviceUrl = UI.input(requireContext(), "http://localhost:3000");
        serviceUrl.setInputType(InputType.TYPE_TEXT_VARIATION_URI);
        serviceUrl.setText(serviceUrlValue);
        UI.margin(serviceUrl, 0, UI.XS, 0, 0);
        card.addView(serviceUrl);

        if (pending && pendingTitle.equals(I18n.text("m_f0556b822864"))) card.addView(new OperationProgress(requireContext(), pendingTitle, startedAt));
        TextView save = UI.button(requireContext(), pending ? I18n.text("m_574ec7517de1") : I18n.text("m_ec7be3f5d41f"), UI.BTN_SECONDARY);
        UI.fill(save);
        UI.margin(save, 0, UI.SM, 0, UI.MD);
        save.setOnClickListener(v -> saveServiceUrl());
        card.addView(save);

        card.addView(UI.label(requireContext(), I18n.text("m_54125d36ad63")));
        TextView preferred = UI.mono(requireContext(), config.optString("preferred_cname", "—"), p.body);
        preferred.setSingleLine(true);
        preferred.setEllipsize(TextUtils.TruncateAt.MIDDLE);
        UI.margin(preferred, 0, UI.XS, 0, 0);
        card.addView(preferred);
        return card;
    }

    private View bindCard() {
        LinearLayout card = UI.card(requireContext());

        card.addView(UI.segmented(requireContext(), new String[]{I18n.text("m_fa521927b78a"), I18n.text("m_a7974623c82b")}, mode, selected -> {
            mode = selected;
            cnameField.setVisibility(selected == 1 ? View.VISIBLE : View.GONE);
            auxField.setVisibility(selected == 1 ? View.VISIBLE : View.GONE);
        }));

        cname = UI.input(requireContext(), I18n.text("m_ecd690c9dc84"));
        cname.setText(draftCname);
        TextView pick = UI.button(requireContext(), I18n.text("m_0bb90daa665f"), UI.BTN_SECONDARY);
        pick.setOnClickListener(v -> openCnamePicker());

        LinearLayout cnameRow = UI.row(requireContext());
        cnameRow.addView(cname, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        LinearLayout.LayoutParams pickLp = new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);
        pickLp.leftMargin = UI.dp(UI.SM);
        cnameRow.addView(pick, pickLp);

        cnameField = UI.field(requireContext(), I18n.text("m_309710e63b6d"), cnameRow, UI.MD);
        card.addView(cnameField);

        mainDomain = UI.input(requireContext(), I18n.text("m_6aede1cf2f10"));
        mainDomain.setInputType(InputType.TYPE_TEXT_VARIATION_URI);
        mainDomain.setText(draftMain);
        card.addView(UI.field(requireContext(), I18n.text("m_6aa6a4279118"), mainDomain, UI.MD));

        auxDomain = UI.input(requireContext(), I18n.text("m_c0c3f6dc7617"));
        auxDomain.setInputType(InputType.TYPE_TEXT_VARIATION_URI);
        auxDomain.setText(draftAux);
        auxField = UI.field(requireContext(), I18n.text("m_c1fb6c5a78c6"), auxDomain, UI.MD);
        card.addView(auxField);

        cnameField.setVisibility(mode == 1 ? View.VISIBLE : View.GONE);
        auxField.setVisibility(mode == 1 ? View.VISIBLE : View.GONE);

        if (pending && pendingTitle.equals(I18n.text("m_919b869a5aef"))) card.addView(new OperationProgress(requireContext(), pendingTitle, startedAt));
        TextView bind = UI.button(requireContext(), pending ? I18n.text("m_574ec7517de1") : I18n.text("m_dd799f925f94"), UI.BTN_PRIMARY);
        UI.fill(bind);
        UI.margin(bind, 0, UI.LG, 0, 0);
        bind.setOnClickListener(v -> bind());
        card.addView(bind);

        TextView batch = UI.button(requireContext(), I18n.text("m_911b0b0d2885"), UI.BTN_SECONDARY);
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
        Sheet.Builder sheet = Sheet.of(requireContext(), I18n.text("m_309710e63b6d"))
                .label(I18n.text("m_0bb90daa665f"));

        sheet.item(R.drawable.ic_nav_check,
                fallback.isEmpty() ? I18n.text("m_dba90d12bb8a") : I18n.text("m_15aadd942734") + fallback + "）",
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
        if (pending) return;
        String value = serviceUrl.getText().toString().trim();
        if (value.isEmpty()) {
            toast(I18n.text("m_d1c4ce4fe6f4"));
            return;
        }
        beginOperation(I18n.text("m_f0556b822864"));
        Api.async(() -> {
            JSONObject payload = new JSONObject();
            payload.put("value", value);
            return Api.post("/api/config/service", payload);
        }, ok -> {
            pending = false;
            config.remove("service_url");
            try {
                config.put("service_url", value);
            } catch (Exception ignored) {
            }
            toast(I18n.text("m_0aaac7dade14"));
            render();
        }, failure -> {
            pending = false;
            resultOk = false;
            resultMessage = failure.getMessage();
            render();
            toast(resultMessage);
        });
    }

    private void bind() {
        if (pending) return;
        final String service = serviceUrl.getText().toString().trim();
        final String main = mainDomain.getText().toString().trim();
        final String aux = auxDomain.getText().toString().trim();
        final String preferred = cname.getText().toString().trim();

        if (service.isEmpty() || main.isEmpty() || (mode == 1 && aux.isEmpty())) {
            setResult(false, mode == 1
                    ? I18n.text("m_72586c7b53ca")
                    : I18n.text("m_cc6561d5be5e"));
            return;
        }

        final String previous = config.optString("service_url", "").trim();
        final int requestedMode = mode;
        beginOperation(I18n.text("m_919b869a5aef"));
        Api.async(() -> {
            // The web console saves a changed forwarding address as part of the
            // same click, so the two never disagree about what was bound.
            if (!service.equals(previous)) {
                JSONObject servicePayload = new JSONObject();
                servicePayload.put("value", service);
                Api.post("/api/config/service", servicePayload);
            }
            JSONObject payload = new JSONObject();
            payload.put("mode", requestedMode == 1 ? "preferred" : "simple");
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
            setResult(true, response.optString("message", I18n.text("m_27347bf9fa01")));
        }, failure -> setResult(false, failure.getMessage()));
    }

    private void setResult(boolean ok, String message) {
        pending = false;
        resultOk = ok;
        resultMessage = message == null ? "" : message;
        captureDraft();
        render();
        toast(ok ? I18n.text("m_67076a58977b") : resultMessage);
    }

    private void captureDraft() {
        if (cname == null) return;
        draftService = serviceUrl.getText().toString();
        draftCname = cname.getText().toString();
        draftMain = mainDomain.getText().toString();
        draftAux = auxDomain.getText().toString();
    }

    private void beginOperation(String title) {
        captureDraft();
        pending = true;
        pendingTitle = title;
        startedAt = android.os.SystemClock.elapsedRealtime();
        resultMessage = "";
        render();
    }

    private void toast(String message) {
        if (message == null || message.isEmpty() || !alive()) return;
        android.widget.Toast.makeText(requireContext(), message, android.widget.Toast.LENGTH_LONG).show();
    }
}
