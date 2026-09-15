package com.tunnelmanager.app;

import android.content.Context;
import android.graphics.Canvas;
import android.graphics.Paint;
import android.graphics.Path;
import android.graphics.RectF;
import android.util.TypedValue;
import android.view.View;

import org.json.JSONArray;
import org.json.JSONObject;

import java.util.Calendar;

/**
 * The dashboard's seven-day latency chart.
 *
 * Same reading as the web chart: each day is a column with a wide, faint peak
 * bar behind and a narrow solid average bar in front, coloured by the worst
 * state that day saw, with dashed grid lines behind both. Drawn on a canvas
 * rather than assembled from views because seven two-bar columns as nested
 * layouts measure and allocate far more than they are worth.
 */
final class ChartView extends View {

    private static final int TRACK_DP = 150;
    private static final int TOP_DP = 18;
    private static final int SIDE_DP = 12;
    private static final int BOTTOM_DP = 12;
    private static final int LABEL_GAP_DP = 8;
    private static final int LABEL_DP = 16;
    private static final int COLUMN_GAP_DP = 8;
    private final Paint paint = new Paint(Paint.ANTI_ALIAS_FLAG);
    private final Paint labelPaint = new Paint(Paint.ANTI_ALIAS_FLAG);
    private final RectF rect = new RectF();
    private final Path path = new Path();
    private final Calendar calendar = Calendar.getInstance();

    private JSONArray buckets = new JSONArray();
    /** Width of one column in seconds. Hourly is what a server that predates
     *  {@code bucket_sec} sends, so it is the safe assumption. */
    private int bucketSec = 3600;
    private long ceiling = 200;

    ChartView(Context ctx) {
        super(ctx);
        Palette p = Theme.p();
        labelPaint.setColor(p.body);
        labelPaint.setTextSize(TypedValue.applyDimension(
                TypedValue.COMPLEX_UNIT_SP, 11, getResources().getDisplayMetrics()));
        labelPaint.setTextAlign(Paint.Align.CENTER);
    }

    /** The API's buckets; an empty list draws the grid and nothing else. */
    void setBuckets(JSONArray value, int bucketSec) {
        buckets = value == null ? new JSONArray() : value;
        this.bucketSec = bucketSec;
        long peak = 0;
        for (int i = 0; i < buckets.length(); i++) {
            peak = Math.max(peak, buckets.optJSONObject(i).optLong("peak_ms"));
        }
        // The web floors the scale at 200ms so a quiet day does not turn a few
        // milliseconds into a full-height bar.
        ceiling = Math.max(200, peak);
        invalidate();
    }

    @Override
    protected void onMeasure(int widthSpec, int heightSpec) {
        int height = UI.dp(TOP_DP + TRACK_DP + LABEL_GAP_DP + LABEL_DP + BOTTOM_DP);
        setMeasuredDimension(resolveSize(UI.dp(240), widthSpec), resolveSize(height, heightSpec));
    }

    @Override
    protected void onDraw(Canvas canvas) {
        super.onDraw(canvas);
        Palette p = Theme.p();
        float trackTop = UI.dp(TOP_DP);
        float trackBottom = trackTop + UI.dp(TRACK_DP);
        float left = UI.dp(SIDE_DP);
        float right = getWidth() - UI.dp(SIDE_DP);

        drawGrid(canvas, left, right, trackTop, trackBottom, p);

        int count = buckets.length();
        if (count == 0) return;
        float gap = UI.dp(COLUMN_GAP_DP);
        float column = Math.max(UI.dp(4), (right - left - gap * (count - 1)) / count);
        // Seven days fit a label each; a denser window has to skip some.
        int labelEvery = count > 12 ? 3 : 1;

        for (int i = 0; i < count; i++) {
            JSONObject bucket = buckets.optJSONObject(i);
            float x = left + i * (column + gap);
            drawColumn(canvas, bucket, x, column, trackTop, trackBottom, p);
            if (i % labelEvery == 0) {
                float baseline = trackBottom + UI.dp(LABEL_GAP_DP) + UI.dp(11);
                canvas.drawText(bucketLabel(bucket.optLong("hour")), x + column / 2f, baseline, labelPaint);
            }
        }
    }

    private void drawGrid(Canvas canvas, float left, float right, float top, float bottom, Palette p) {
        paint.setStyle(Paint.Style.STROKE);
        paint.setStrokeWidth(Math.max(1, UI.dp(1)));
        paint.setColor(alpha(p.hairline, 0.5f));
        Paint.Cap cap = paint.getStrokeCap();
        paint.setStrokeCap(Paint.Cap.BUTT);
        // Three dashed rules, evenly spread — the web draws the same three.
        float[] dashes = {UI.dp(3), UI.dp(3)};
        for (int i = 0; i < 3; i++) {
            float y = top + (bottom - top) * i / 2f;
            paint.setPathEffect(new android.graphics.DashPathEffect(dashes, 0));
            canvas.drawLine(left, y, right, y, paint);
        }
        paint.setPathEffect(null);
        paint.setStrokeCap(cap);
    }

    private void drawColumn(Canvas canvas, JSONObject bucket, float x, float column,
                            float trackTop, float trackBottom, Palette p) {
        boolean empty = bucket.optInt("total") == 0;
        boolean down = bucket.optInt("down") > 0;
        boolean warn = bucket.optInt("warn") > 0;
        // The web paints the average bar in the semantic colour, not the pill's
        // border tint — that is the whole point of the legend.
        int accent = down ? p.error : warn ? p.warning : p.ink;

        if (empty) {
            paint.setStyle(Paint.Style.FILL);
            paint.setColor(alpha(p.hairlineStrong, 0.55f));
            rect.set(x + column * 0.27f, trackBottom - UI.dp(3), x + column * 0.73f, trackBottom);
            canvas.drawRoundRect(rect, UI.dp(2), UI.dp(2), paint);
            return;
        }

        // Peak: a tall, faint shell that shows how far above the average the
        // hour reached. Border and fill share a hue at two alphas, and the
        // outline is left open at the bottom so no seam shows under the bar.
        long peakMs = bucket.optLong("peak_ms");
        float peakHeight = height(peakMs, trackTop, trackBottom);
        if (peakHeight > 0) {
            float pl = x + column * 0.10f;
            float pr = x + column * 0.90f;
            float top = trackBottom - peakHeight;
            float r = UI.dp(3);
            path.reset();
            path.moveTo(pl, trackBottom);
            path.lineTo(pl, top + r);
            path.quadTo(pl, top, pl + r, top);
            path.lineTo(pr - r, top);
            path.quadTo(pr, top, pr, top + r);
            path.lineTo(pr, trackBottom);
            paint.setStyle(Paint.Style.FILL);
            paint.setColor(alpha(accent, down ? 0.12f : 0.10f));
            canvas.drawPath(path, paint);
            paint.setStyle(Paint.Style.STROKE);
            paint.setStrokeWidth(Math.max(1, UI.dp(1)));
            paint.setColor(alpha(accent, down ? 0.25f : 0.14f));
            canvas.drawPath(path, paint);
        }

        // Average: the solid bar the eye actually reads.
        double avgMs = bucket.optDouble("avg_ms", 0);
        if (avgMs <= 0) avgMs = Math.max(peakMs * 0.35, 6);
        float avgHeight = height((long) avgMs, trackTop, trackBottom);
        if (avgHeight > 0) {
            paint.setStyle(Paint.Style.FILL);
            paint.setColor(accent);
            rect.set(x + column * 0.27f, trackBottom - avgHeight, x + column * 0.73f, trackBottom);
            canvas.drawRoundRect(rect, UI.dp(3), UI.dp(3), paint);
        }
    }

    /** Percentage of the track, floored at 2% so a tiny value still reads. */
    private float height(long ms, float trackTop, float trackBottom) {
        if (ms <= 0) return 0;
        double pct = Math.max(2, Math.round(ms / (double) ceiling * 100));
        return (float) (pct / 100.0 * (trackBottom - trackTop));
    }

    /** A date for day-wide buckets, a clock time for anything shorter. */
    private String bucketLabel(long seconds) {
        calendar.setTimeInMillis(seconds * 1000L);
        if (bucketSec >= 86400) {
            return (calendar.get(Calendar.MONTH) + 1) + "/" + calendar.get(Calendar.DAY_OF_MONTH);
        }
        return calendar.get(Calendar.HOUR_OF_DAY) + "时";
    }

    private static int alpha(int color, float value) {
        return (color & 0x00FFFFFF) | (Math.round(255 * value) << 24);
    }
}
