package com.tunnelmanager.app;

import android.content.ClipData;
import android.content.ClipboardManager;
import android.content.Context;
import android.os.Bundle;
import android.text.Editable;
import android.text.InputFilter;
import android.text.TextWatcher;
import android.view.View;
import android.view.inputmethod.EditorInfo;
import android.view.inputmethod.InputMethodManager;
import android.widget.EditText;
import android.widget.LinearLayout;
import android.widget.TextView;
import android.widget.Toast;

import androidx.lifecycle.MutableLiveData;
import androidx.lifecycle.ViewModel;
import androidx.lifecycle.ViewModelProvider;

import org.json.JSONObject;

public final class TunnelCreateSheet {
    private final ConsoleActivity activity;
    private final CreationState state;
    private Sheet.Builder sheet;
    private LinearLayout content;
    private EditText name;
    private TextView error;
    private TextView submit;
    private boolean showingResult;
    private boolean disposed;

    TunnelCreateSheet(ConsoleActivity activity, Bundle saved) {
        this.activity = activity;
        state = new ViewModelProvider(activity).get(CreationState.class);
        if (!state.open && saved != null && saved.getBoolean("tunnel_create_open")) {
            state.open = true;
            state.name = saved.getString("tunnel_create_name", "");
            if (saved.getBoolean("tunnel_create_pending")) {
                state.error = I18n.text("m_6f47a98020cc");
            }
        }
        state.revision.observe(activity, ignored -> render());
    }

    void show() {
        if (!Session.hasPerm("tunnels")) return;
        state.open = true;
        state.changed();
    }

    void save(Bundle saved) {
        if (!state.open || state.result != null) return;
        saved.putBoolean("tunnel_create_open", true);
        saved.putString("tunnel_create_name", state.name);
        saved.putBoolean("tunnel_create_pending", state.busy);
    }

    void dispose() {
        disposed = true;
        if (sheet != null) sheet.dismiss();
    }

    private void render() {
        if (disposed || activity.isFinishing() || activity.isDestroyed() || !state.open) return;
        if (!Session.hasPerm("tunnels")) {
            if (sheet != null) sheet.dismiss();
            state.open = false;
            return;
        }
        if (sheet == null || !sheet.isShowing()) build();
        sheet.setDismissible(!state.busy);
        submit.setEnabled(!state.busy);
        submit.setText(state.result != null ? I18n.text("m_c0b3fbff51cc") : state.busy ? I18n.text("m_56b63def0e29") : I18n.text("m_e33080d615cb"));
        if (state.result != null) {
            if (!showingResult) showResult(state.result);
            if (state.refreshPending) {
                state.refreshPending = false;
                activity.onTunnelCreated();
            }
        } else {
            name.setEnabled(!state.busy);
            error.setText(state.error);
            error.setVisibility(state.error.isEmpty() ? View.GONE : View.VISIBLE);
        }
    }

    private void build() {
        showingResult = false;
        content = UI.column(activity);
        content.addView(UI.muted(activity, I18n.text("m_b0e77cb061b2")));
        name = UI.input(activity, I18n.text("m_e06a8e35aa96"));
        name.setSingleLine(true);
        name.setSaveEnabled(false);
        name.setFilters(new InputFilter[]{new InputFilter.LengthFilter(60)});
        name.setText(state.name);
        name.setImeOptions(EditorInfo.IME_ACTION_DONE);
        name.addTextChangedListener(new TextWatcher() {
            @Override public void beforeTextChanged(CharSequence text, int start, int count, int after) {}
            @Override public void onTextChanged(CharSequence text, int start, int before, int count) {}
            @Override public void afterTextChanged(Editable text) { state.name = text.toString(); }
        });
        name.setOnEditorActionListener((view, action, event) -> {
            if (action != EditorInfo.IME_ACTION_DONE) return false;
            create();
            return true;
        });
        content.addView(UI.field(activity, I18n.text("m_b679939ed787"), name, UI.LG));
        error = UI.banner(activity, "", true);
        error.setAccessibilityLiveRegion(View.ACCESSIBILITY_LIVE_REGION_POLITE);
        error.setVisibility(View.GONE);
        UI.margin(error, 0, UI.MD, 0, 0);
        content.addView(error);
        submit = UI.button(activity, I18n.text("m_e33080d615cb"), UI.BTN_PRIMARY);
        UI.fill(submit);
        submit.setOnClickListener(view -> {
            if (state.result != null) sheet.dismiss();
            else create();
        });
        sheet = Sheet.of(activity, I18n.text("m_389b92b75e48")).content(content).footer(submit).onDismiss(() -> {
            if (disposed) return;
            state.open = false;
            state.name = "";
            state.error = "";
            state.result = null;
        });
        sheet.show();
    }

    private void create() {
        if (state.busy || state.result != null) return;
        if (!state.name.trim().isEmpty()) {
            InputMethodManager keyboard = (InputMethodManager) activity.getSystemService(Context.INPUT_METHOD_SERVICE);
            if (keyboard != null) keyboard.hideSoftInputFromWindow(name.getWindowToken(), 0);
        }
        state.submit();
    }

    private void showResult(JSONObject result) {
        showingResult = true;
        content.removeAllViews();
        content.addView(UI.strong(activity, I18n.text("m_2c410ba6c66f")));
        TextView hint = UI.muted(activity, I18n.text("m_b83305d65d40"));
        UI.margin(hint, 0, UI.SM, 0, UI.MD);
        content.addView(hint);
        String command = result.optString("run_command", "");
        boolean hasCommand = !command.isEmpty();
        if (!hasCommand) command = result.optString("token", "");
        if (!command.isEmpty()) {
            final String value = command;
            TextView code = UI.mono(activity, command, Theme.p().ink);
            code.setTextIsSelectable(true);
            code.setSaveEnabled(false);
            code.setPadding(UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD), UI.dp(UI.MD));
            code.setBackground(UI.rounded(Theme.p().canvasSoft2, UI.RADIUS_MD));
            content.addView(code);
            TextView copy = UI.button(activity, hasCommand ? I18n.text("m_209c6b3fd0a8") : I18n.text("m_6bef8cdd90c0"), UI.BTN_SECONDARY);
            UI.fill(copy);
            UI.margin(copy, 0, UI.SM, 0, 0);
            copy.setOnClickListener(view -> {
                ClipboardManager clipboard = (ClipboardManager) activity.getSystemService(Context.CLIPBOARD_SERVICE);
                if (clipboard == null) return;
                clipboard.setPrimaryClip(ClipData.newPlainText(I18n.text("m_a8933dc6d806"), value));
                if (android.os.Build.VERSION.SDK_INT < 33) Toast.makeText(activity, I18n.text("m_8f6f8d979c98"), Toast.LENGTH_SHORT).show();
            });
            content.addView(copy);
        }
        String warning = result.optString("warning", "");
        if (command.isEmpty() && warning.isEmpty()) warning = I18n.text("m_d0b9e70b0e09");
        if (!warning.isEmpty()) {
            TextView message = UI.banner(activity, warning, true);
            UI.margin(message, 0, UI.MD, 0, 0);
            content.addView(message);
        }
        String id = result.optString("id", "");
        if (!id.isEmpty()) {
            TextView detail = UI.button(activity, I18n.text("m_f2ce3be31088"), UI.BTN_GHOST);
            UI.margin(detail, 0, UI.SM, 0, 0);
            detail.setOnClickListener(view -> {
                sheet.dismiss();
                activity.open("/tunnels/" + id);
            });
            content.addView(detail);
        }
    }

    public static final class CreationState extends ViewModel {
        final MutableLiveData<Integer> revision = new MutableLiveData<>(0);
        String name = "";
        String error = "";
        boolean open;
        boolean busy;
        boolean refreshPending;
        boolean cleared;
        JSONObject result;

        public CreationState() {}

        void changed() {
            Integer value = revision.getValue();
            revision.setValue(value == null ? 1 : value + 1);
        }

        void submit() {
            if (busy || result != null || !Session.hasPerm("tunnels")) return;
            String submittedName = name.trim();
            if (submittedName.isEmpty()) {
                error = I18n.text("m_d34e09f1e95f");
                changed();
                return;
            }
            busy = true;
            error = "";
            changed();
            Api.async(() -> Api.post("/api/tunnels", new JSONObject().put("name", submittedName)), created -> {
                if (cleared) return;
                busy = false;
                result = created;
                refreshPending = true;
                changed();
            }, failure -> {
                if (cleared) return;
                busy = false;
                error = failure.getMessage();
                if (failure.code == 0) error += I18n.text("m_f24ff46c10ce");
                changed();
            });
        }

        @Override protected void onCleared() {
            cleared = true;
            result = null;
        }
    }
}
