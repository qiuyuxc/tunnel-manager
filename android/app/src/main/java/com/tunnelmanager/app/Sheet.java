package com.tunnelmanager.app;

import android.animation.ValueAnimator;
import android.app.Dialog;
import android.content.Context;
import android.graphics.Color;
import android.graphics.Typeface;
import android.graphics.drawable.GradientDrawable;
import android.graphics.drawable.ColorDrawable;
import android.view.Gravity;
import android.view.View;
import android.view.ViewGroup;
import android.view.Window;
import android.view.WindowManager;
import android.widget.ImageView;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;

/**
 * The phone console's bottom sheet — the web {@code SheetPanel} in native form.
 *
 * Everything the tab bar cannot hold lives behind one of these: the "+" create
 * actions and the "更多" page list. It is a dialog rather than a view in the
 * activity because it has to dim the page, take the back button, and sit above
 * whatever the current fragment is drawing — including a WebView, which no
 * amount of z-ordering inside the fragment would cover.
 */
final class Sheet {

    /** Mirrors the web sheet's max-height: 88vh, less the handle and header. */
    private static final float MAX_HEIGHT_RATIO = 0.88f;

    /** The sheet on screen, if any — the shell's back handler needs to see it. */
    private static Dialog visible;

    private Sheet() {
    }

    /** Closes the open sheet. Returns false when there was nothing to close. */
    static boolean dismissVisible() {
        if (visible == null || !visible.isShowing()) return false;
        visible.dismiss();
        return true;
    }

    static Builder of(Context ctx, String title) {
        return new Builder(ctx, title);
    }

    static final class Builder {

        private final Context ctx;
        private final Dialog dialog;
        private final LinearLayout body;
        private final ScrollView scroll;
        /** Rows added since the last {@link #label}; drives the hairline seams. */
        private int rowsInGroup;

        private Builder(Context ctx, String title) {
            this.ctx = ctx;
            Palette p = Theme.p();

            LinearLayout root = UI.column(ctx);
            root.setBackground(topRounded(p.canvasRaised));
            // Without this the dim scrim's tap would fall through to the page.
            root.setClickable(true);

            root.addView(buildHandle());
            root.addView(buildHead(title));

            body = UI.column(ctx);
            int side = UI.dp(UI.MD);
            body.setPadding(side, 0, side, UI.dp(20));

            scroll = new Capped(ctx);
            scroll.setClipToPadding(false);
            scroll.setVerticalScrollBarEnabled(false);
            scroll.addView(body, new LinearLayout.LayoutParams(
                    ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));
            root.addView(scroll, new LinearLayout.LayoutParams(
                    ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));

            dialog = new Dialog(ctx, R.style.SheetDialog);
            dialog.setContentView(root);
            Window window = dialog.getWindow();
            if (window != null) {
                window.setGravity(Gravity.BOTTOM);
                window.setLayout(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT);
                // A floating dialog window still carries the platform's own
                // inset background; the sheet paints its own.
                window.setBackgroundDrawable(new ColorDrawable(Color.TRANSPARENT));
                window.addFlags(WindowManager.LayoutParams.FLAG_DIM_BEHIND);
                window.setDimAmount(0f);
                applyInsets(window, root);
            }
            dialog.setCanceledOnTouchOutside(true);
        }

        private View buildHandle() {
            Palette p = Theme.p();
            LinearLayout wrap = UI.row(ctx);
            wrap.setGravity(Gravity.CENTER);
            wrap.setPadding(0, UI.dp(9), 0, 0);
            View handle = new View(ctx);
            handle.setBackground(UI.rounded(p.hairlineStrong, UI.RADIUS_PILL));
            wrap.addView(handle, new LinearLayout.LayoutParams(UI.dp(36), UI.dp(4)));
            return wrap;
        }

        private View buildHead(String title) {
            Palette p = Theme.p();
            LinearLayout head = UI.row(ctx);
            head.setPadding(UI.dp(UI.LG), UI.dp(UI.MD), UI.dp(UI.LG), UI.dp(10));
            head.addView(UI.text(ctx, title, 15, p.ink, Typeface.BOLD));

            View spacer = new View(ctx);
            head.addView(spacer, new LinearLayout.LayoutParams(0, 1, 1f));

            ImageView close = new ImageView(ctx);
            close.setImageResource(R.drawable.ic_nav_close);
            close.setColorFilter(p.mute);
            int pad = UI.dp(7);
            close.setPadding(pad, pad, pad, pad);
            close.setBackground(UI.pressable(UI.rounded(p.canvasSoft2, UI.RADIUS_PILL), p.btnGhostHover));
            close.setClickable(true);
            close.setOnClickListener(v -> dismiss());
            head.addView(close, new LinearLayout.LayoutParams(UI.dp(28), UI.dp(28)));
            return head;
        }

        /** Keeps the last row clear of the gesture bar when the window is flush. */
        private void applyInsets(Window window, View root) {
            window.getDecorView().setOnApplyWindowInsetsListener((v, insets) -> {
                int bottom = insets.getSystemWindowInsetBottom();
                if (bottom > 0) {
                    root.setPadding(0, 0, 0, bottom);
                }
                return insets;
            });
        }

        // ------------------------------------------------------------- content

        /** Section heading, matching the web sheet's uppercase group label. */
        Builder label(String text) {
            LinearLayout wrap = UI.column(ctx);
            wrap.setPadding(UI.dp(10), UI.dp(14), UI.dp(10), UI.dp(5));
            wrap.addView(UI.label(ctx, text));
            body.addView(wrap);
            rowsInGroup = 0;
            return this;
        }

        /**
         * A create action: icon tile, title, description, chevron. The web
         * {@code .action-row}, used by the "+" sheet.
         */
        Builder action(int iconRes, String title, String desc, Runnable onClick) {
            Palette p = Theme.p();
            LinearLayout row = UI.row(ctx);
            row.setBackground(UI.pressable(
                    UI.roundedStroke(p.canvasRaised, UI.RADIUS_LG, p.hairline, 1), p.btnGhostHover));
            row.setPadding(UI.dp(10), UI.dp(UI.MD), UI.dp(10), UI.dp(UI.MD));
            if (rowsInGroup > 0) {
                LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(
                        ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT);
                lp.topMargin = UI.dp(UI.SM);
                row.setLayoutParams(lp);
            }

            LinearLayout tile = UI.column(ctx);
            tile.setGravity(Gravity.CENTER);
            tile.setBackground(UI.rounded(p.canvasSoft2, UI.RADIUS_LG));
            ImageView icon = new ImageView(ctx);
            icon.setImageResource(iconRes);
            icon.setColorFilter(p.ink);
            tile.addView(icon, new LinearLayout.LayoutParams(UI.dp(20), UI.dp(20)));
            LinearLayout.LayoutParams tileLp = new LinearLayout.LayoutParams(UI.dp(40), UI.dp(40));
            tileLp.rightMargin = UI.dp(UI.MD);
            row.addView(tile, tileLp);

            LinearLayout text = UI.column(ctx);
            text.addView(UI.strong(ctx, title));
            TextView sub = UI.muted(ctx, desc);
            UI.margin(sub, 0, UI.dp(1), 0, 0);
            text.addView(sub);
            row.addView(text, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));

            ImageView chevron = new ImageView(ctx);
            chevron.setImageResource(R.drawable.ic_nav_chevron);
            chevron.setColorFilter(p.mute);
            row.addView(chevron, new LinearLayout.LayoutParams(UI.dp(15), UI.dp(15)));

            row.setClickable(true);
            row.setOnClickListener(v -> {
                dismiss();
                onClick.run();
            });
            body.addView(row);
            rowsInGroup++;
            return this;
        }

        /** A page row inside a group, separated by hairlines. The web {@code .group-row}. */
        Builder item(int iconRes, String title, Runnable onClick) {
            Palette p = Theme.p();
            if (rowsInGroup > 0) {
                View seam = new View(ctx);
                seam.setBackgroundColor(p.hairline);
                body.addView(seam, new LinearLayout.LayoutParams(
                        ViewGroup.LayoutParams.MATCH_PARENT, Math.max(1, UI.dp(1))));
            }

            LinearLayout row = UI.row(ctx);
            row.setMinimumHeight(UI.dp(44));
            row.setPadding(UI.dp(10), UI.dp(UI.SM), UI.dp(10), UI.dp(UI.SM));
            row.setBackground(UI.pressable(UI.rounded(p.canvasRaised, UI.RADIUS_LG), p.btnGhostHover));

            ImageView icon = new ImageView(ctx);
            icon.setImageResource(iconRes);
            icon.setColorFilter(p.body);
            LinearLayout.LayoutParams iconLp = new LinearLayout.LayoutParams(UI.dp(19), UI.dp(19));
            iconLp.rightMargin = UI.dp(UI.MD);
            row.addView(icon, iconLp);

            row.addView(UI.text(ctx, title, 14, p.ink, Typeface.NORMAL),
                    new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));

            ImageView chevron = new ImageView(ctx);
            chevron.setImageResource(R.drawable.ic_nav_chevron);
            chevron.setColorFilter(p.mute);
            row.addView(chevron, new LinearLayout.LayoutParams(UI.dp(15), UI.dp(15)));

            row.setClickable(true);
            row.setOnClickListener(v -> {
                dismiss();
                onClick.run();
            });
            body.addView(row);
            rowsInGroup++;
            return this;
        }

        /** Arbitrary content, for sheets that are not a list of links. */
        Builder content(View view) {
            body.addView(view);
            rowsInGroup++;
            return this;
        }

        // ------------------------------------------------------------ lifecycle

        void dismiss() {
            dialog.dismiss();
        }

        void show() {
            visible = dialog;
            dialog.setOnDismissListener(d -> {
                if (visible == dialog) visible = null;
            });
            dialog.show();
            fadeDim(0f, 0.45f);
        }

        /**
         * The dialog's own window animation can only move the sheet; the scrim
         * has to be animated by hand or it snaps in ahead of the panel.
         */
        private void fadeDim(float from, float to) {
            Window window = dialog.getWindow();
            if (window == null) return;
            ValueAnimator anim = ValueAnimator.ofFloat(from, to);
            anim.setDuration(220);
            anim.addUpdateListener(a -> window.setDimAmount((float) a.getAnimatedValue()));
            anim.start();
        }
    }

    /** Tops the corners only; the bottom edge is the screen edge. */
    private static GradientDrawable topRounded(int fill) {
        GradientDrawable d = new GradientDrawable();
        d.setShape(GradientDrawable.RECTANGLE);
        float r = UI.dp(18);
        d.setCornerRadii(new float[]{r, r, r, r, 0, 0, 0, 0});
        d.setColor(fill);
        return d;
    }

    /**
     * A ScrollView that stops growing at {@link #cap}. Without it the sheet would
     * measure its content at full length and the header would be pushed off the
     * top of the screen instead of the body scrolling.
     */
    private static final class Capped extends ScrollView {

        private final int cap;

        Capped(Context ctx) {
            super(ctx);
            int screen = ctx.getResources().getDisplayMetrics().heightPixels;
            cap = (int) (screen * MAX_HEIGHT_RATIO) - UI.dp(63);
        }

        @Override
        protected void onMeasure(int widthSpec, int heightSpec) {
            super.onMeasure(widthSpec, MeasureSpec.makeMeasureSpec(cap, MeasureSpec.AT_MOST));
        }
    }
}
