package com.tunnelmanager.app;

import android.content.Context;
import android.content.res.ColorStateList;
import android.os.SystemClock;
import android.view.View;
import android.view.ViewGroup;
import android.widget.LinearLayout;
import android.widget.ProgressBar;
import android.widget.TextView;

final class OperationProgress extends LinearLayout {
    private final long startedAt;
    private final TextView elapsed;
    private final TextView detail;
    private final Runnable tick = new Runnable() {
        @Override public void run() {
            long seconds = Math.max(0, (SystemClock.elapsedRealtime() - startedAt) / 1000);
            elapsed.setText("已等待 " + seconds + " 秒");
            detail.setText(seconds >= 15 ? "仍未收到服务端结果，请继续等待，不要重复提交。" : "请求已发出，正在等待服务端返回结果。");
            postDelayed(this, 1000);
        }
    };

    OperationProgress(Context context, String title, long startedAt) {
        super(context);
        this.startedAt = startedAt;
        setOrientation(VERTICAL);
        LinearLayout card = UI.card(context);
        LinearLayout row = UI.row(context);
        if (Theme.motionEnabled()) {
            ProgressBar spinner = new ProgressBar(context);
            spinner.setIndeterminate(true);
            spinner.setIndeterminateTintList(ColorStateList.valueOf(Theme.p().success));
            spinner.setImportantForAccessibility(View.IMPORTANT_FOR_ACCESSIBILITY_NO);
            LinearLayout.LayoutParams size = new LinearLayout.LayoutParams(UI.dp(20), UI.dp(20));
            size.rightMargin = UI.dp(UI.SM);
            row.addView(spinner, size);
        }
        TextView heading = UI.label(context, title);
        heading.setAccessibilityLiveRegion(View.ACCESSIBILITY_LIVE_REGION_POLITE);
        row.addView(heading, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
        card.addView(row);
        detail = UI.muted(context, "请求已发出，正在等待服务端返回结果。");
        UI.margin(detail, 0, UI.SM, 0, UI.XS);
        card.addView(detail);
        elapsed = UI.mono(context, "已等待 0 秒", Theme.p().mute);
        elapsed.setImportantForAccessibility(View.IMPORTANT_FOR_ACCESSIBILITY_NO);
        card.addView(elapsed);
        addView(card, new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT));
    }

    @Override protected void onAttachedToWindow() {
        super.onAttachedToWindow();
        removeCallbacks(tick);
        tick.run();
    }

    @Override protected void onDetachedFromWindow() {
        removeCallbacks(tick);
        super.onDetachedFromWindow();
    }

    static void setInputsEnabled(View view, boolean enabled) {
        if (view instanceof OperationProgress) return;
        view.setEnabled(enabled);
        if (view instanceof ViewGroup) {
            ViewGroup group = (ViewGroup) view;
            for (int index = 0; index < group.getChildCount(); index++) setInputsEnabled(group.getChildAt(index), enabled);
        }
    }
}
