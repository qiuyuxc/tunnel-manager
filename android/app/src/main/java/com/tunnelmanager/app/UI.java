package com.tunnelmanager.app;

import android.content.Context;
import android.animation.AnimatorSet;
import android.animation.ObjectAnimator;
import android.animation.StateListAnimator;
import android.content.res.ColorStateList;
import android.content.res.Resources;
import android.graphics.Canvas;
import android.graphics.Paint;
import android.graphics.Typeface;
import android.graphics.drawable.ColorDrawable;
import android.graphics.drawable.LayerDrawable;
import android.graphics.drawable.Drawable;
import android.graphics.drawable.GradientDrawable;
import android.graphics.drawable.RippleDrawable;
import android.util.TypedValue;
import android.view.Gravity;
import android.view.View;
import android.view.ViewGroup;
import android.widget.EditText;
import android.widget.ImageView;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;
import android.widget.CompoundButton;
import android.view.accessibility.AccessibilityNodeInfo;
import android.view.animation.DecelerateInterpolator;
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout;
import android.view.ViewTreeObserver;

import java.util.function.IntConsumer;

final class UI {

    static final int XS = 4;
    static final int SM = 8;
    static final int MD = 12;
    static final int LG = 16;
    static final int XL = 24;
    static final int XXL = 32;
    static final int XXXL = 48;

    static final int RADIUS_SM = 10;
    static final int RADIUS_MD = 14;
    static final int RADIUS_LG = 22;
    static final int RADIUS_PILL = 999;

    static final int HEADER_H = 56;
    static final int TABBAR_H = 64;
    static final int SIDEBAR_W = 240;
    static final int CONTENT_MAX_W = 760;
    /** Below this width the shell uses the phone chrome (tabs, not a sidebar). */
    static final int WIDE_DP = 900;

    static final int BTN_PRIMARY = 0;
    static final int BTN_SECONDARY = 1;
    static final int BTN_GHOST = 2;
    static final int BTN_DANGER = 3;

    private static final float DENSITY = Resources.getSystem().getDisplayMetrics().density;

    private UI() {
    }

    static int dp(float value) {
        return Math.round(value * DENSITY);
    }

    // ------------------------------------------------------------- containers

    static LinearLayout column(Context ctx) {
        LinearLayout l = new LinearLayout(ctx);
        l.setOrientation(LinearLayout.VERTICAL);
        return l;
    }

    static LinearLayout row(Context ctx) {
        LinearLayout l = new LinearLayout(ctx);
        l.setOrientation(LinearLayout.HORIZONTAL);
        l.setGravity(Gravity.CENTER_VERTICAL);
        return l;
    }

    /** A settings card: raised surface, hairline border, generous padding. */
    static LinearLayout card(Context ctx) {
        Palette p = Theme.p();
        LinearLayout l = column(ctx);
        l.setBackground(roundedStroke(p.canvasRaised, RADIUS_LG, p.hairline, 1));
        l.setPadding(dp(20), dp(20), dp(20), dp(20));
        return l;
    }

    /**
     * The body inset every page shares.
     *
     * The top is deliberately tighter than the sides: the shell's header is a
     * 56dp band with its own hairline, and matching that with a 16dp gap on top
     * of it reads as a page that starts too far down.
     */
    static void pagePadding(LinearLayout body) {
        int side = dp(20);
        boolean wide = body.getResources().getConfiguration().screenWidthDp >= WIDE_DP;
        body.setPadding(side, dp(MD), side, dp(wide ? XXXL : TABBAR_H + 40));
    }

    static View spacer(Context ctx, int heightDp) {
        View v = new View(ctx);
        v.setLayoutParams(new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, dp(heightDp)));
        return v;
    }

    /**
     * Adds a full-width child to a vertical container.
     *
     * Used instead of add-then-{@code margin} so rows stretch across the card:
     * a view that is margined before it is attached has no LayoutParams yet, and
     * the fallback those helpers create is WRAP_CONTENT.
     */
    static void addRow(LinearLayout parent, View child, int topDp) {
        parent.addView(child, new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));
        if (topDp > 0) margin(child, 0, topDp, 0, 0);
    }

    static View divider(Context ctx) {
        Palette p = Theme.p();
        View v = new View(ctx);
        LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, Math.max(1, dp(1)));
        lp.topMargin = dp(MD);
        lp.bottomMargin = dp(MD);
        v.setLayoutParams(lp);
        v.setBackgroundColor(p.hairline);
        return v;
    }

    static void margin(View v, int left, int top, int right, int bottom) {
        ViewGroup.LayoutParams raw = v.getLayoutParams();
        ViewGroup.MarginLayoutParams lp = raw instanceof ViewGroup.MarginLayoutParams
                ? (ViewGroup.MarginLayoutParams) raw
                : new LinearLayout.LayoutParams(
                        raw != null ? raw.width : ViewGroup.LayoutParams.WRAP_CONTENT,
                        raw != null ? raw.height : ViewGroup.LayoutParams.WRAP_CONTENT);
        lp.setMargins(dp(left), dp(top), dp(right), dp(bottom));
        v.setLayoutParams(lp);
    }

    /** Gives a view a weight inside a horizontal LinearLayout. */
    static void weight(View v, float value) {
        ViewGroup.LayoutParams raw = v.getLayoutParams();
        LinearLayout.LayoutParams lp = raw instanceof LinearLayout.LayoutParams
                ? (LinearLayout.LayoutParams) raw
                : new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT);
        lp.width = 0;
        lp.weight = value;
        v.setLayoutParams(lp);
    }

    /** Stretches a view to its parent's width, keeping any margins it had. */
    static void fill(View v) {
        ViewGroup.LayoutParams raw = v.getLayoutParams();
        int height = raw == null ? ViewGroup.LayoutParams.WRAP_CONTENT : raw.height;
        ViewGroup.MarginLayoutParams lp = new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, height);
        if (raw instanceof ViewGroup.MarginLayoutParams) {
            ViewGroup.MarginLayoutParams margins = (ViewGroup.MarginLayoutParams) raw;
            lp.setMargins(margins.leftMargin, margins.topMargin, margins.rightMargin, margins.bottomMargin);
        }
        v.setLayoutParams(lp);
    }

    // ------------------------------------------------------------------ text

    static TextView text(Context ctx, String value, float sizeSp, int color, int style) {
        TextView v = new TextView(ctx);
        v.setText(value);
        v.setTextSize(TypedValue.COMPLEX_UNIT_SP, sizeSp);
        v.setTextColor(color);
        v.setTypeface(Typeface.DEFAULT, style);
        v.setFontFeatureSettings("tnum");
        return v;
    }

    /** Page heading, matches the console's h2. */
    static TextView pageTitle(Context ctx, String value) {
        TextView title = text(ctx, value, 26, Theme.p().ink, Typeface.BOLD);
        title.setLetterSpacing(-0.025f);
        return title;
    }

    /** Card heading. */
    static TextView cardTitle(Context ctx, String value) {
        return text(ctx, value, 15, Theme.p().ink, Typeface.BOLD);
    }

    static TextView strong(Context ctx, String value) {
        return text(ctx, value, 14, Theme.p().ink, Typeface.BOLD);
    }

    static TextView body(Context ctx, String value) {
        TextView v = text(ctx, value, 14, Theme.p().body, Typeface.NORMAL);
        v.setLineSpacing(dp(3), 1f);
        return v;
    }

    static TextView muted(Context ctx, String value) {
        TextView v = text(ctx, value, 13, Theme.p().mute, Typeface.NORMAL);
        v.setLineSpacing(dp(3), 1f);
        return v;
    }

    /** Uppercase mono label — the console uses these above form fields. */
    static TextView label(Context ctx, String value) {
        TextView v = text(ctx, value, 12, Theme.p().body, Typeface.NORMAL);
        return v;
    }

    static TextView mono(Context ctx, String value, int color) {
        TextView v = text(ctx, value, 13, color, Typeface.NORMAL);
        v.setTypeface(Typeface.MONOSPACE, Typeface.NORMAL);
        return v;
    }

    // ------------------------------------------------------------- decoration

    static GradientDrawable rounded(int fill, float radiusDp) {
        GradientDrawable d = new ThemedShape();
        d.setShape(GradientDrawable.RECTANGLE);
        d.setCornerRadius(radiusDp >= RADIUS_PILL ? dp(999) : dp(radiusDp));
        d.setColor(fill);
        return d;
    }

    static GradientDrawable roundedStroke(int fill, float radiusDp, int stroke, float strokeDp) {
        GradientDrawable d = rounded(fill, radiusDp);
        d.setStroke(Math.max(1, dp(strokeDp)), stroke);
        return d;
    }

    /**
     * Ripple over a shape. The mask copy matters: handing the same Drawable in
     * as both content and mask makes the ripple mutate the shape it is drawing
     * on, which shows up as corners losing their radius while pressed.
     */
    /** A filled or hollow circle — the phone's checkbox, used for row selection. */
    static GradientDrawable circle(int fill, int stroke, float strokeDp) {
        GradientDrawable d = new ThemedShape();
        d.setShape(GradientDrawable.OVAL);
        d.setColor(fill);
        if (stroke != 0) d.setStroke(Math.max(1, dp(strokeDp)), stroke);
        return d;
    }

    static Drawable pressable(Drawable content, int rippleColor) {
        Drawable mask = content.getConstantState() != null
                ? content.getConstantState().newDrawable().mutate()
                : null;
        return new RippleDrawable(ColorStateList.valueOf(rippleColor), content, mask);
    }

    // ---------------------------------------------------------------- buttons

    static TextView button(Context ctx, String label, int kind) {
        Palette p = Theme.p();
        TextView v = text(ctx, label, 14, p.btnPrimaryText, Typeface.BOLD);
        v.setGravity(Gravity.CENTER);
        v.setPadding(dp(16), dp(11), dp(16), dp(11));
        v.setClickable(true);
        v.setFocusable(true);
        v.setMinHeight(dp(48));
        pressFeedback(v);

        int fill;
        Drawable shape;
        switch (kind) {
            case BTN_PRIMARY:
                fill = p.btnPrimaryBg;
                v.setTextColor(p.btnPrimaryText);
                shape = rounded(fill, RADIUS_PILL);
                break;
            case BTN_DANGER:
                fill = p.resultErrorBg;
                v.setTextColor(p.error);
                shape = roundedStroke(fill, RADIUS_PILL, p.resultErrorBorder, 1);
                break;
            case BTN_GHOST:
                v.setTextColor(p.mute);
                v.setTypeface(Typeface.DEFAULT, Typeface.NORMAL);
                shape = rounded(p.canvas, RADIUS_PILL);
                break;
            case BTN_SECONDARY:
            default:
                fill = p.canvasSoft2;
                v.setTextColor(p.ink);
                shape = rounded(fill, RADIUS_PILL);
                break;
        }
        v.setBackground(pressable(shape, p.btnGhostHover));
        return v;
    }

    /**
     * A square icon button, the console's {@code .btn-icon}.
     *
     * Used for the actions that sit inside a card's header row, where a labelled
     * button would eat the width the title needs.
     */
    static ImageView iconButton(Context ctx, int iconRes, int tint, View.OnClickListener onClick) {
        Palette p = Theme.p();
        ImageView button = new ImageView(ctx);
        button.setImageResource(iconRes);
        tint(button, tint);
        button.setClickable(true);
        button.setFocusable(true);
        button.setBackground(pressable(rounded(p.canvasSoft2, RADIUS_PILL), p.btnGhostHover));
        pressFeedback(button);
        int inset = dp(SM);
        button.setPadding(inset, inset, inset, inset);
        LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(dp(44), dp(44));
        lp.leftMargin = dp(XS);
        button.setLayoutParams(lp);
        button.setOnClickListener(onClick);
        return button;
    }

    // ------------------------------------------------------------- form input

    static EditText input(Context ctx, String hint) {
        Palette p = Theme.p();
        EditText e = new EditText(ctx);
        e.setHint(hint);
        e.setTextSize(TypedValue.COMPLEX_UNIT_SP, 15);
        e.setTextColor(p.ink);
        e.setHintTextColor(p.mute);
        e.setBackground(roundedStroke(p.canvasSoft2, RADIUS_MD, p.hairline, 1));
        e.setMinHeight(dp(48));
        e.setPadding(dp(MD), dp(MD), dp(MD), dp(MD));
        e.setSingleLine(true);
        // Stated rather than left to the default: a WRAP_CONTENT EditText sizes
        // itself from its hint, so two fields in one card end up different widths.
        e.setLayoutParams(new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));
        return e;
    }

    static EditText textArea(Context ctx, String hint) {
        EditText e = input(ctx, hint);
        e.setSingleLine(false);
        e.setGravity(Gravity.TOP | Gravity.START);
        e.setMinLines(3);
        e.setTypeface(Typeface.MONOSPACE);
        e.setTextSize(TypedValue.COMPLEX_UNIT_SP, 13);
        return e;
    }

    /** Label stacked above its control, the console's `.field` pattern. */
    static LinearLayout field(Context ctx, String label, View control) {
        return field(ctx, label, control, 0);
    }

    /** As above, with a gap above the whole field. */
    static LinearLayout field(Context ctx, String label, View control, int topDp) {
        LinearLayout box = column(ctx);
        box.setLayoutParams(new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));
        TextView caption = UI.label(ctx, label);
        margin(caption, 0, 0, 0, 6);
        box.addView(caption);
        box.addView(control, new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));
        if (topDp > 0) margin(box, 0, topDp, 0, 0);
        return box;
    }

    /** A whole card used as a page's form container. */
    static LinearLayout form(Context ctx) {
        LinearLayout card = card(ctx);
        return card;
    }

    // ----------------------------------------------------------- status pills

    /** Renders a target state with the console's semantic colours. */
    static TextView statusPill(Context ctx, String state) {
        Palette p = Theme.p();
        int fill;
        int stroke;
        int ink;
        String text;
        if ("ok".equals(state)) {
            fill = p.statusHealthyBg;
            stroke = p.statusHealthyBorder;
            ink = p.statusHealthyText;
            text = "正常";
        } else if ("warn".equals(state)) {
            fill = p.statusDegradedBg;
            stroke = p.statusDegradedBorder;
            ink = p.statusDegradedText;
            text = "降级";
        } else if ("down".equals(state)) {
            fill = p.statusDownBg;
            stroke = p.statusDownBorder;
            ink = p.statusDownText;
            text = "异常";
        } else {
            fill = p.canvasSoft;
            stroke = p.hairlineStrong;
            ink = p.mute;
            text = "未知";
        }
        TextView v = text(ctx, text, 11, ink, Typeface.BOLD);
        v.setPadding(dp(9), dp(3), dp(9), dp(3));
        v.setBackground(roundedStroke(fill, RADIUS_PILL, stroke, 1));
        return v;
    }

    static TextView tunnelStatusPill(Context ctx, String status) {
        TunnelFilter.State state = TunnelFilter.state(status);
        TextView pill = statusPill(ctx, state.style);
        pill.setText(state.label);
        return pill;
    }

    static TextView tag(Context ctx, String label, boolean ok) {
        Palette p = Theme.p();
        TextView v = text(ctx, label, 11, ok ? p.statusHealthyText : p.mute, Typeface.BOLD);
        v.setPadding(dp(9), dp(3), dp(9), dp(3));
        v.setBackground(ok
                ? roundedStroke(p.statusHealthyBg, RADIUS_PILL, p.statusHealthyBorder, 1)
                : roundedStroke(p.canvasSoft, RADIUS_PILL, p.hairlineStrong, 1));
        return v;
    }

    // -------------------------------------------------------------- messages

    /** Inline banner, matching `.settings-card` result blocks on the web. */
    static TextView banner(Context ctx, String message, boolean error) {
        Palette p = Theme.p();
        TextView v = text(ctx, message, 13, error ? p.resultErrorText : p.resultSuccessText, Typeface.NORMAL);
        v.setPadding(dp(MD), dp(MD), dp(MD), dp(MD));
        // Banners span the card the way the web's result blocks do; a margin
        // applied by the caller then keeps this width instead of shrinking it.
        v.setLayoutParams(new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));
        v.setBackground(roundedStroke(
                error ? p.resultErrorBg : p.resultSuccessBg,
                RADIUS_MD,
                error ? p.resultErrorBorder : p.resultSuccessBorder,
                1));
        return v;
    }

    /**
     * Puts a rebuilt page back where it was scrolled to.
     *
     * A plain {@code post} is not enough: the scroll range is still zero until
     * the replacement children have been laid out, so the offset is applied on
     * the first pre-draw of the frame that follows instead.
     */
    static void restoreScroll(final ScrollView scroll, final int y) {
        if (scroll == null || y <= 0) return;
        scroll.post(() -> scroll.scrollTo(0, y));
        final ViewTreeObserver observer = scroll.getViewTreeObserver();
        observer.addOnPreDrawListener(new ViewTreeObserver.OnPreDrawListener() {
            @Override
            public boolean onPreDraw() {
                if (observer.isAlive()) observer.removeOnPreDrawListener(this);
                scroll.scrollTo(0, y);
                return true;
            }
        });
    }

    // -------------------------------------------------------------- controls

    /**
     * A row of mutually exclusive chips. The console's segmented control, used
     * wherever a form would otherwise need a Spinner.
     */
    static LinearLayout segmented(Context ctx, String[] options, int selected, IntConsumer onChange) {
        Palette p = Theme.p();
        LinearLayout row = row(ctx);
        row.setLayoutParams(new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));
        row.setBackground(rounded(p.canvasSoft2, RADIUS_PILL));
        row.setPadding(dp(4), dp(4), dp(4), dp(4));

        for (int i = 0; i < options.length; i++) {
            final int index = i;
            TextView chip = text(ctx, options[i], 13, p.mute, Typeface.NORMAL);
            chip.setGravity(Gravity.CENTER);
            chip.setSingleLine(true);
            chip.setPadding(dp(6), dp(7), dp(6), dp(7));
            chip.setLayoutParams(new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
            chip.setClickable(true);
            chip.setFocusable(true);
            chip.setMinHeight(dp(40));
            pressFeedback(chip);
            chip.setOnClickListener(v -> {
                paintSegmented(row, index);
                onChange.accept(index);
            });
            row.addView(chip);
        }
        paintSegmented(row, selected);
        return row;
    }

    private static void paintSegmented(LinearLayout row, int selected) {
        Palette p = Theme.p();
        for (int i = 0; i < row.getChildCount(); i++) {
            TextView chip = (TextView) row.getChildAt(i);
            boolean active = i == selected;
            chip.setTextColor(active ? p.ink : p.mute);
            chip.setTypeface(Typeface.DEFAULT, Typeface.NORMAL);
            chip.setSelected(active);
            chip.setBackground(active
                    ? rounded(p.canvasRaised, RADIUS_PILL)
                    : null);
        }
    }

    /**
     * The console's switch: a 34x20 track with a 16dp knob, the web
     * {@code .switch}. Drawn rather than borrowed from Material so both
     * palettes paint it from the same tokens as everything else.
     */
    static final class Toggle extends View {

        private final Paint paint = new Paint(Paint.ANTI_ALIAS_FLAG);
        private boolean on;

        Toggle(Context ctx, boolean on) {
            super(ctx);
            this.on = on;
        }

        boolean isOn() {
            return on;
        }

        void setOn(boolean value) {
            if (on == value) return;
            on = value;
            invalidate();
        }

        @Override
        public void onInitializeAccessibilityNodeInfo(AccessibilityNodeInfo info) {
            super.onInitializeAccessibilityNodeInfo(info);
            info.setClassName("android.widget.Switch");
            info.setCheckable(true);
            info.setChecked(on);
        }

        /**
         * A bare View offered {@code AT_MOST} takes the whole space, so a
         * switch added without explicit LayoutParams would eat the width a
         * weighted sibling needs — and LinearLayout then skips the pass that
         * would have given that sibling its share, collapsing it to zero.
         * The switch has an intrinsic size; state it instead of relying on the
         * caller to remember.
         */
        @Override
        protected void onMeasure(int widthMeasureSpec, int heightMeasureSpec) {
            setMeasuredDimension(
                    resolveSize(dp(38), widthMeasureSpec),
                    resolveSize(dp(24), heightMeasureSpec));
        }

        @Override
        protected void onDraw(Canvas canvas) {
            float w = getWidth();
            float h = getHeight();
            float radius = h / 2f;
            paint.setColor(on ? Theme.p().btnPrimaryBg : Theme.p().hairlineStrong);
            canvas.drawRoundRect(0, 0, w, h, radius, radius, paint);

            float knobRadius = radius - dp(2);
            float cx = on ? w - knobRadius - dp(2) : knobRadius + dp(2);
            paint.setColor(on ? Theme.p().btnPrimaryText : Theme.p().body);
            canvas.drawCircle(cx, h / 2f, knobRadius, paint);
        }
    }

    static void pressFeedback(View view) {
        if (!Theme.motionEnabled()) {
            view.setStateListAnimator(null);
            view.setScaleX(1f);
            view.setScaleY(1f);
            return;
        }
        StateListAnimator states = new StateListAnimator();
        AnimatorSet pressed = new AnimatorSet();
        pressed.playTogether(ObjectAnimator.ofFloat(view, View.SCALE_X, 0.97f),
                ObjectAnimator.ofFloat(view, View.SCALE_Y, 0.97f));
        pressed.setDuration(100);
        AnimatorSet released = new AnimatorSet();
        released.playTogether(ObjectAnimator.ofFloat(view, View.SCALE_X, 1f),
                ObjectAnimator.ofFloat(view, View.SCALE_Y, 1f));
        released.setDuration(180);
        released.setInterpolator(new DecelerateInterpolator());
        states.addState(new int[]{android.R.attr.state_pressed, android.R.attr.state_enabled}, pressed);
        states.addState(new int[]{}, released);
        view.setStateListAnimator(states);
    }

    static void retheme(View view, Palette previous) {
        Palette next = Theme.p();
        repaint(view.getBackground(), previous, next);
        if (view instanceof TextView) {
            TextView text = (TextView) view;
            text.setTextColor(previous.remap(text.getCurrentTextColor(), next));
            text.setHintTextColor(previous.remap(text.getCurrentHintTextColor(), next));
            for (Drawable drawable : text.getCompoundDrawablesRelative()) repaint(drawable, previous, next);
        }
        if (view instanceof ImageView) {
            ImageView image = (ImageView) view;
            if (image.getImageTintList() != null) {
                int color = image.getImageTintList().getDefaultColor();
                tint(image, previous.remap(color, next));
            }
        }
        if (view instanceof CompoundButton) {
            ((CompoundButton) view).setButtonTintList(new ColorStateList(
                    new int[][]{new int[]{android.R.attr.state_checked}, new int[]{}},
                    new int[]{next.btnPrimaryBg, next.hairlineStrong}));
        }
        if (view instanceof SwipeRefreshLayout) {
            ((SwipeRefreshLayout) view).setColorSchemeColors(next.success);
            ((SwipeRefreshLayout) view).setProgressBackgroundColorSchemeColor(next.canvasRaised);
        }
        if (view.isClickable()) pressFeedback(view);
        if (view instanceof ViewGroup) {
            ViewGroup group = (ViewGroup) view;
            for (int index = 0; index < group.getChildCount(); index++) retheme(group.getChildAt(index), previous);
        }
        view.invalidate();
    }

    static void tint(ImageView image, int color) {
        image.setImageTintList(ColorStateList.valueOf(color));
    }

    private static void repaint(Drawable drawable, Palette previous, Palette next) {
        if (drawable == null) return;
        if (drawable instanceof ThemedShape) {
            ((ThemedShape) drawable).repaint(previous, next);
        } else if (drawable instanceof ColorDrawable) {
            ColorDrawable color = (ColorDrawable) drawable;
            color.setColor(previous.remap(color.getColor(), next));
        } else if (drawable instanceof GradientDrawable) {
            GradientDrawable shape = (GradientDrawable) drawable;
            if (shape.getColor() != null) shape.setColor(previous.remap(shape.getColor().getDefaultColor(), next));
        }
        if (drawable instanceof LayerDrawable) {
            LayerDrawable layers = (LayerDrawable) drawable;
            for (int index = 0; index < layers.getNumberOfLayers(); index++) repaint(layers.getDrawable(index), previous, next);
        }
        if (drawable instanceof RippleDrawable) ((RippleDrawable) drawable).setColor(ColorStateList.valueOf(next.btnGhostHover));
        drawable.invalidateSelf();
    }

    private static final class ThemedShape extends GradientDrawable {
        private int fill;
        private int stroke;
        private int strokeWidth;

        @Override
        public void setColor(int color) {
            fill = color;
            super.setColor(color);
        }

        @Override
        public void setStroke(int width, int color) {
            strokeWidth = width;
            stroke = color;
            super.setStroke(width, color);
        }

        void repaint(Palette previous, Palette next) {
            setColor(previous.remap(fill, next));
            if (strokeWidth > 0) setStroke(strokeWidth, previous.remap(stroke, next));
        }
    }
}
