package com.tunnelmanager.app;

import android.graphics.Canvas;
import android.graphics.Color;
import android.graphics.ColorFilter;
import android.graphics.LinearGradient;
import android.graphics.Outline;
import android.graphics.Paint;
import android.graphics.PixelFormat;
import android.graphics.Rect;
import android.graphics.RectF;
import android.graphics.Shader;
import android.graphics.drawable.Drawable;

final class GlassDrawable extends Drawable {
    private final Paint fill = new Paint(Paint.ANTI_ALIAS_FLAG);
    private final Paint edge = new Paint(Paint.ANTI_ALIAS_FLAG);
    private final RectF bounds = new RectF();
    private final float radius;
    private final boolean selected;
    private Palette paintedPalette;
    private int opacity = 255;

    GlassDrawable(float radiusDp, boolean selected) {
        radius = UI.dp(radiusDp);
        this.selected = selected;
        edge.setStyle(Paint.Style.STROKE);
        edge.setStrokeWidth(Math.max(1f, UI.dp(0.7f)));
    }

    @Override
    protected void onBoundsChange(Rect next) {
        bounds.set(next.left + 1, next.top + 1, next.right - 1, next.bottom - 1);
        paintedPalette = null;
    }

    @Override
    public void draw(Canvas canvas) {
        Palette palette = Theme.p();
        if (paintedPalette != palette) {
            edge.setShader(new LinearGradient(0, bounds.top, 0, bounds.bottom,
                    palette.glassHighlight, Color.TRANSPARENT, Shader.TileMode.CLAMP));
            paintedPalette = palette;
        }
        int color = selected ? palette.sidebarActiveBg
                : Theme.reducedEffects() ? palette.canvasSoft2 : palette.glass;
        fill.setColor(color);
        fill.setAlpha(Color.alpha(color) * opacity / 255);
        edge.setAlpha(opacity);
        canvas.drawRoundRect(bounds, radius, radius, fill);
        if (!Theme.reducedEffects()) canvas.drawRoundRect(bounds, radius, radius, edge);
    }

    @Override
    public void getOutline(Outline outline) {
        outline.setRoundRect(getBounds(), radius);
    }

    @Override
    public void setAlpha(int alpha) {
        opacity = alpha;
        invalidateSelf();
    }

    @Override
    public void setColorFilter(ColorFilter filter) {
        fill.setColorFilter(filter);
        edge.setColorFilter(filter);
        invalidateSelf();
    }

    @Override
    public int getOpacity() {
        return PixelFormat.TRANSLUCENT;
    }
}
