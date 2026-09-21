package com.tunnelmanager.app;

import android.content.Context;
import android.graphics.Canvas;
import android.graphics.Paint;
import android.graphics.RectF;
import android.view.View;

final class HealthRingView extends View {
    private final Paint paint = new Paint(Paint.ANTI_ALIAS_FLAG);
    private final RectF ring = new RectF();
    private final int healthy;
    private final int total;

    HealthRingView(Context context, int healthy, int total) {
        super(context);
        this.healthy = healthy;
        this.total = total;
        setContentDescription(total == 0 ? "暂无监控目标" : total + " 个目标中 " + healthy + " 个正常");
        setImportantForAccessibility(IMPORTANT_FOR_ACCESSIBILITY_YES);
    }

    @Override
    protected void onDraw(Canvas canvas) {
        super.onDraw(canvas);
        Palette palette = Theme.p();
        float inset = UI.dp(4);
        ring.set(inset, inset, getWidth() - inset, getHeight() - inset);
        paint.setStyle(Paint.Style.STROKE);
        paint.setStrokeWidth(UI.dp(3));
        paint.setStrokeCap(Paint.Cap.ROUND);
        paint.setColor(palette.hairline);
        canvas.drawOval(ring, paint);
        paint.setColor(palette.success);
        float sweep = total > 0 ? Math.min(1f, Math.max(0f, (float) healthy / total)) * 360f : 0f;
        canvas.drawArc(ring, -90, sweep, false, paint);
        paint.setStyle(Paint.Style.FILL);
        paint.setTextAlign(Paint.Align.CENTER);
        paint.setTextSize(UI.dp(22));
        String value = total == 0 ? "待检测" : String.valueOf(healthy);
        if (total == 0) paint.setTextSize(UI.dp(12));
        canvas.drawText(value, getWidth() / 2f, getHeight() / 2f - (paint.ascent() + paint.descent()) / 2f, paint);
    }
}
