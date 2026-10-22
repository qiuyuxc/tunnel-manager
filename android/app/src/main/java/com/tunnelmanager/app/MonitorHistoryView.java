package com.tunnelmanager.app;

import android.content.Context;
import android.graphics.Canvas;
import android.graphics.Paint;
import android.graphics.RectF;
import android.view.View;

import java.util.Locale;

final class MonitorHistoryView extends View {
    private final Paint paint = new Paint(Paint.ANTI_ALIAS_FLAG);
    private final RectF rectangle = new RectF();
    private final MonitorHistory history;
    private final boolean chart;
    private final long intervalMillis;

    MonitorHistoryView(Context context, MonitorHistory history, boolean chart, int intervalSeconds) {
        super(context);
        this.history = history;
        this.chart = chart;
        intervalMillis = (long) intervalSeconds * 1000;
        MonitorHistory.Counts counts = new MonitorHistory.Counts();
        for (MonitorHistory.Sample sample : history.samples) counts.add(sample.state);
        setContentDescription((chart ? I18n.text("m_9678bf23c0ca") : I18n.text("m_fdd7eab6188f")) + history.samples.size()
                + I18n.text("m_ce502035bacc") + MonitorViews.range(history) + "，" + MonitorViews.summary(counts)
                + (chart ? I18n.text("m_ce8e2991cf69") : I18n.text("m_16c76db0c6ee")));
        setImportantForAccessibility(IMPORTANT_FOR_ACCESSIBILITY_YES);
    }

    @Override protected void onMeasure(int widthSpec, int heightSpec) {
        int height = chart ? Math.round(UI.dp(164) * Math.max(1f, getResources().getConfiguration().fontScale)) : UI.dp(28);
        setMeasuredDimension(resolveSize(UI.dp(280), widthSpec), resolveSize(height, heightSpec));
    }

    @Override protected void onDraw(Canvas canvas) {
        super.onDraw(canvas);
        paint.setStyle(Paint.Style.FILL);
        if (chart) drawChart(canvas);
        else drawBars(canvas);
    }

    private void drawBars(Canvas canvas) {
        int slots = Math.max(24, history.samples.size());
        float gap = UI.dp(2);
        float width = Math.max(1, (getWidth() - (slots - 1) * gap) / slots);
        for (int index = 0; index < slots; index++) {
            paint.setColor(index < history.samples.size()
                    ? MonitorViews.color(history.samples.get(index).state) : Theme.p().hairline);
            float left = index * (width + gap);
            rectangle.set(left, UI.dp(3), left + width, getHeight() - UI.dp(3));
            canvas.drawRoundRect(rectangle, UI.dp(2), UI.dp(2), paint);
        }
    }

    private void drawChart(Canvas canvas) {
        float textSize = 10 * getResources().getDisplayMetrics().scaledDensity;
        paint.setTextSize(textSize);
        paint.setTypeface(android.graphics.Typeface.create("sans-serif", android.graphics.Typeface.NORMAL));
        double maximum = Math.max(10, Math.ceil(history.maxLatency() / 10) * 10);
        String topLabel = axisLabel(maximum);
        float left = Math.max(UI.dp(40), paint.measureText(topLabel) + UI.dp(8));
        float right = getWidth() - UI.dp(6);
        float top = textSize + UI.dp(2);
        float bottom = getHeight() - textSize - UI.dp(16);
        if (right <= left || bottom <= top) return;
        for (int tick = 0; tick <= 2; tick++) {
            float position = top + (bottom - top) * tick / 2;
            paint.setColor(Theme.p().hairline);
            paint.setStrokeWidth(UI.dp(1));
            canvas.drawLine(left, position, right, position, paint);
            paint.setColor(Theme.p().mute);
            paint.setTextAlign(Paint.Align.RIGHT);
            canvas.drawText(axisLabel(maximum * (2 - tick) / 2), left - UI.dp(8), position + textSize / 3, paint);
        }
        float previousX = 0;
        float previousY = 0;
        for (int index = 0; index < history.samples.size(); index++) {
            MonitorHistory.Sample sample = history.samples.get(index);
            float positionX = left + (right - left) * (float) history.fraction(index);
            float positionY = sample.hasLatency() ? bottom - (bottom - top) * (float) (sample.latency / maximum) : bottom + UI.dp(10);
            paint.setColor(MonitorViews.color(sample.state));
            paint.setStrokeWidth(UI.dp(2));
            if (history.connects(index, intervalMillis)) canvas.drawLine(previousX, previousY, positionX, positionY, paint);
            if (sample.hasLatency()) canvas.drawCircle(positionX, positionY, UI.dp(2.5f), paint);
            else {
                canvas.drawLine(positionX - UI.dp(2), positionY - UI.dp(2), positionX + UI.dp(2), positionY + UI.dp(2), paint);
                canvas.drawLine(positionX - UI.dp(2), positionY + UI.dp(2), positionX + UI.dp(2), positionY - UI.dp(2), paint);
            }
            previousX = positionX;
            previousY = positionY;
        }
        if (history.samples.isEmpty()) {
            paint.setColor(Theme.p().mute);
            paint.setTextAlign(Paint.Align.CENTER);
            canvas.drawText(I18n.text("m_e7380808d95e"), (left + right) / 2, (top + bottom) / 2, paint);
        }
    }

    private String axisLabel(double milliseconds) {
        if (milliseconds >= 1000000) return String.format(Locale.US, "%.0f ks", milliseconds / 1000000);
        if (milliseconds >= 1000) return String.format(Locale.US, "%.1f s", milliseconds / 1000);
        return String.format(Locale.US, "%.0f ms", milliseconds);
    }
}
