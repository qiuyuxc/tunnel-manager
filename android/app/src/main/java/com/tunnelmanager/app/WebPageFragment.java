package com.tunnelmanager.app;

import android.annotation.SuppressLint;
import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;

import org.json.JSONObject;

/**
 * Fallback for console pages the native shell cannot draw yet.
 *
 * Porting eighteen pages in one go would leave the app unusable for most of the
 * work, so unported routes keep opening the real console here and the port can
 * proceed one page at a time. The web chrome is hidden on purpose: the shell
 * already draws navigation, and two navigation bars stacked is worse than none.
 */
public class WebPageFragment extends PageFragment {

    private static final String ROUTE = "route";

    static WebPageFragment forRoute(String path) {
        WebPageFragment fragment = new WebPageFragment();
        Bundle args = new Bundle();
        args.putString(ROUTE, path);
        fragment.setArguments(args);
        return fragment;
    }

    /**
     * The console's palette keys, written once so the embedded pages stop
     * rendering the warm theme while the native shell draws Vercel. Static
     * because the WebView's localStorage is per origin, not per fragment.
     */
    private static boolean paletteSynced = false;

    /** Called when the shell flips palette; the next web page re-seeds itself. */
    static void resetPalette() {
        paletteSynced = false;
    }

    private String route = "/dashboard";
    private WebView web;

    void refreshTheme() {
        if (web == null) return;
        web.setBackgroundColor(Theme.p().canvas);
        injectSession(web);
    }

    @Override
    public void onCreate(@Nullable Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        if (getArguments() != null) route = getArguments().getString(ROUTE, route);
    }

    @Override
    String route() {
        return route;
    }

    @SuppressLint("SetJavaScriptEnabled")
    @Override
    protected View build(@NonNull LayoutInflater inflater, @Nullable ViewGroup container) {
        web = new WebView(requireContext());
        WebSettings settings = web.getSettings();
        settings.setJavaScriptEnabled(true);
        settings.setDomStorageEnabled(true);
        settings.setUseWideViewPort(true);
        settings.setLoadWithOverviewMode(true);
        settings.setMixedContentMode(WebSettings.MIXED_CONTENT_ALWAYS_ALLOW);
        web.setBackgroundColor(Theme.p().canvas);
        web.setWebViewClient(new WebViewClient() {
            @Override
            public void onPageFinished(WebView view, String url) {
                injectSession(view);
            }
        });
        web.loadUrl(Session.server() + route);
        return web;
    }

    /** Seeds the console's session, then takes its own navigation out of view. */
    private void injectSession(WebView view) {
        String script = "(function(){try{"
                + "var wantTheme='enterprise';"
                + "var wantDark=" + (Theme.isDark() ? "true" : "false") + ";"
                + "var changed=localStorage.getItem('visual_theme')!==wantTheme"
                + "||localStorage.getItem('dark_mode')!==String(wantDark);"
                + "localStorage.setItem('visual_theme',wantTheme);"
                + "localStorage.setItem('dark_mode',String(wantDark));"
                + "localStorage.setItem('auth_token'," + JSONObject.quote(Session.token()) + ");"
                + "localStorage.setItem('auth_username'," + JSONObject.quote(Session.username()) + ");"
                + "localStorage.setItem('auth_role'," + JSONObject.quote(Session.role()) + ");"
                + "return changed?'1':'0';"
                + "}catch(e){return '0'}})()";
        view.evaluateJavascript(script, value -> {
            // The Vue store reads these keys while it boots, so a page that was
            // already painted with the wrong palette has to be re-run once.
            if (!paletteSynced && value != null && value.contains("1")) {
                paletteSynced = true;
                view.reload();
                return;
            }
            paletteSynced = true;
            stripWebChrome(view);
        });
    }

    private void stripWebChrome(WebView view) {
        String script = "(function(){"
                + "if(!document.getElementById('tm-embed')){"
                + "var s=document.createElement('style');s.id='tm-embed';"
                + "s.textContent='.sidebar,.mobile-header,.tabbar{display:none!important}"
                + ".app-shell{padding-left:0!important;padding-bottom:0!important}"
                + ".main-content{padding-top:0!important}"
                + ".app-main{margin-left:0!important;padding-top:0!important;padding-bottom:104px!important}';"
                + "(document.head||document.documentElement).appendChild(s);"
                + "}})()";
        view.evaluateJavascript(script, null);
    }

    @Override
    public void onDestroyView() {
        if (web != null) {
            web.destroy();
            web = null;
        }
        super.onDestroyView();
    }

    /** Keeps the back button walking the page's own history first. */
    boolean canGoBack() {
        return web != null && web.canGoBack();
    }

    void goBack() {
        if (web != null) web.goBack();
    }
}
