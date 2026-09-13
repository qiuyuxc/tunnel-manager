package com.tunnelmanager.app;

import android.app.Dialog;
import android.graphics.Typeface;
import android.view.Gravity;
import android.view.View;
import android.view.ViewGroup;
import android.view.Window;
import android.view.WindowManager;
import android.widget.LinearLayout;
import android.widget.TextView;

import android.content.Context;

/**
 * The console's centred dialog: a title, some content, and a row of actions.
 *
 * Used for the things a bottom sheet is wrong for — a one-line confirmation, a
 * single-field create prompt — where the sheet's drag handle and full-height
 * body would be more ceremony than the task deserves.
 */
final class Modal {

    private Modal() {
    }

    static Builder of(Context ctx, String title) {
        return new Builder(ctx, title);
    }

    static final class Builder {

        private final Context ctx;
        private final Dialog dialog;
        private final LinearLayout body;
        private final LinearLayout actions;

        private Builder(Context ctx, String title) {
            this.ctx = ctx;
            Palette p = Theme.p();

            LinearLayout card = UI.column(ctx);
            card.setBackground(UI.roundedStroke(p.canvasRaised, UI.RADIUS_LG + 2, p.hairline, 1));
            card.setPadding(UI.dp(UI.LG), UI.dp(UI.LG), UI.dp(UI.LG), UI.dp(UI.LG));

            card.addView(UI.text(ctx, title, 16, p.ink, Typeface.BOLD));

            body = UI.column(ctx);
            card.addView(body);

            actions = UI.row(ctx);
            actions.setGravity(Gravity.END);
            LinearLayout.LayoutParams actionsLp = new LinearLayout.LayoutParams(
                    ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT);
            actionsLp.topMargin = UI.dp(UI.MD);
            card.addView(actions, actionsLp);

            dialog = new Dialog(ctx, R.style.ModalDialog);
            dialog.setContentView(card);
            Window window = dialog.getWindow();
            if (window != null) {
                window.setGravity(Gravity.CENTER);
                window.addFlags(WindowManager.LayoutParams.FLAG_DIM_BEHIND);
                window.setDimAmount(0.5f);
                window.setLayout(UI.dp(320), ViewGroup.LayoutParams.WRAP_CONTENT);
            }
            dialog.setCanceledOnTouchOutside(true);
        }

        /** A line of explanatory copy under the title. */
        Builder message(String text) {
            TextView v = UI.muted(ctx, text);
            UI.margin(v, 0, UI.SM, 0, 0);
            body.addView(v);
            return this;
        }

        /** A control the dialog is asking about, e.g. a name field. */
        Builder content(View view) {
            UI.fill(view);
            UI.margin(view, 0, UI.SM, 0, 0);
            body.addView(view);
            return this;
        }

        Builder cancel(String label) {
            TextView button = UI.button(ctx, label, UI.BTN_SECONDARY);
            button.setOnClickListener(v -> dialog.dismiss());
            LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(
                    ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT);
            lp.leftMargin = UI.dp(UI.SM);
            actions.addView(button, lp);
            return this;
        }

        /** The affirmative action. Dismisses before running, so callers can navigate. */
        Builder confirm(String label, boolean danger, Runnable action) {
            TextView button = UI.button(ctx, label, danger ? UI.BTN_DANGER : UI.BTN_PRIMARY);
            button.setOnClickListener(v -> {
                dialog.dismiss();
                action.run();
            });
            actions.addView(button);
            return this;
        }

        void dismiss() {
            dialog.dismiss();
        }

        void show() {
            dialog.show();
        }
    }
}
