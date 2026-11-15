package com.tunnelmanager.app;

import android.annotation.SuppressLint;
import android.app.Dialog;
import android.content.Context;
import android.view.Window;
import android.view.WindowManager;
import android.webkit.WebResourceRequest;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;

/** External authorization shares WebView cookies, but receives no panel token
 * or native bridge. Only a verified panel origin returns to the console view. */
final class WebAuthorization {
    @SuppressLint("SetJavaScriptEnabled")
    static void open(Context context, WebView panel, String server, String url) {
        if (WebOrigin.origin(url).isEmpty()) return;
        Dialog dialog = new Dialog(context);
        dialog.requestWindowFeature(Window.FEATURE_NO_TITLE);
        WebView authorization = new WebView(context);
        WebSettings settings = authorization.getSettings();
        settings.setJavaScriptEnabled(true);
        settings.setDomStorageEnabled(true);
        settings.setMixedContentMode(WebSettings.MIXED_CONTENT_NEVER_ALLOW);
        settings.setAllowFileAccess(false);
        settings.setAllowContentAccess(false);
        authorization.setWebViewClient(new WebViewClient() {
            private boolean navigate(String destination) {
                if (WebOrigin.trusted(server, destination)) {
                    dialog.dismiss();
                    panel.loadUrl(destination);
                    return true;
                }
                return WebOrigin.origin(destination).isEmpty();
            }

            @Override public boolean shouldOverrideUrlLoading(WebView view, WebResourceRequest request) {
                if (!request.isForMainFrame()) return WebOrigin.origin(request.getUrl().toString()).isEmpty();
                return navigate(request.getUrl().toString());
            }

            @Override public void onPageStarted(WebView view, String destination, android.graphics.Bitmap icon) {
                if (navigate(destination)) view.stopLoading();
            }
        });
        dialog.setContentView(authorization);
        dialog.setOnDismissListener(ignored -> { authorization.stopLoading(); authorization.destroy(); });
        dialog.show();
        Window window = dialog.getWindow();
        if (window != null) window.setLayout(WindowManager.LayoutParams.MATCH_PARENT, WindowManager.LayoutParams.MATCH_PARENT);
        authorization.loadUrl(url);
    }
}
