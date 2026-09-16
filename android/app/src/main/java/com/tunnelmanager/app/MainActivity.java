package com.tunnelmanager.app;

import android.annotation.SuppressLint;
import android.app.Activity;
import android.content.Intent;
import android.content.SharedPreferences;
import android.content.pm.PackageManager;
import android.graphics.drawable.ColorDrawable;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.os.PowerManager;
import android.provider.Settings;
import android.view.KeyEvent;
import android.view.View;
import android.view.animation.DecelerateInterpolator;
import android.view.inputmethod.EditorInfo;
import android.webkit.JavascriptInterface;
import android.webkit.CookieManager;
import android.webkit.WebSettings;
import android.webkit.WebStorage;
import android.webkit.WebView;
import android.webkit.WebViewClient;
import android.widget.Button;
import android.widget.EditText;
import android.widget.FrameLayout;
import android.widget.ImageView;
import android.widget.ProgressBar;
import android.widget.ScrollView;
import android.widget.TextView;

import org.json.JSONObject;
import org.json.JSONTokener;

import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;

/**
 * Native shell around the console.
 *
 * The console itself is the same Vue build the browser gets; this activity only
 * owns the part a web page cannot do properly: remembering which server to talk
 * to, and signing in against /api/admin/login so the WebView starts out with a
 * session instead of bouncing through the web login form.
 */
public class MainActivity extends Activity {

    static final String PREFS = "tunnel_manager_app";
    static final String KEY_SERVER = "server";
    static final String KEY_TOKEN = "token";
    /** Extra the console sets when it had to drop a session, and its value. */
    static final String EXTRA_NOTICE = "notice";
    static final String NOTICE_EXPIRED = "expired";
    private static final String KEY_USERNAME = "username";
    private static final String KEY_ROLE = "role";

    /** Console routes that mean "nobody is signed in any more". */
    private static final String[] SIGNED_OUT_PATHS = {"/login", "/"};

    /** Notification channels owned by {@link AlertService}. */
    static final String CH_SERVICE = "tm_service";
    static final String CH_DOWN = "tm_alert_down";
    static final String CH_UP = "tm_alert_up";

    static final String ACTION_START_ALERTS = "com.tunnelmanager.app.START_ALERTS";
    static final String ACTION_STOP_ALERTS = "com.tunnelmanager.app.STOP_ALERTS";
    /** Extra on the launch intent: console route a notification wants opened. */
    static final String EXTRA_PATH = "path";

    private static final int REQ_NOTIFY = 41;

    private final Handler ui = new Handler(Looper.getMainLooper());

    private SharedPreferences prefs;
    private ScrollView authScroll;
    private View stepLogin;
    private View stepTwoFactor;
    private EditText inputServer;
    private EditText inputAccount;
    private EditText inputPassword;
    private EditText inputCode;
    private TextView textError;
    private TextView textError2fa;
    private Button btnLogin;
    private Button btnVerify;
    private Button btnWebLogin;
    private Button btnBackToLogin;
    private Button btnPasskeyLogin;
    private Button btnPasskeyVerify;
    private static final String PASSKEY_LOGIN_LABEL = "使用通行密钥登录";
    private static final String PASSKEY_VERIFY_LABEL = "使用通行密钥验证";
    /** True when the challenged account has a passkey, so the 2FA step can offer it. */
    private boolean passkeysAvailable;
    private ProgressBar progress;
    private WebView web;
    private View turnstileBox;
    private WebView turnstileView;
    private TextView turnstileHint;

    /** Human-verification widget state for the native sign-in form. */
    private String turnstileServer = "";
    private String turnstileSiteKey = "";
    private String turnstileToken = "";
    private boolean turnstileNeeded = false;
    /** Credentials held while the widget finishes solving. */
    private String pendingPassword = null;

    private String challengeToken = "";
    private boolean injecting = false;
    /** Guards the one-way hand-off to the console after a web sign-in. */
    private boolean launching = false;
    private boolean bounced = false;
    /** True while the console's own /login page owns the sign-in flow. */
    private boolean webLoginMode = false;
    /** Console route requested by a tapped notification, consumed once. */
    private String pendingPath = "";
    /** True once /login has been dropped from the WebView back/forward list. */
    private boolean historyArmed = false;

    // ---------------------------------------------------------------- lifecycle

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        // The form is inflated from XML, so its palette has to arrive through
        // the theme. Pick it from the app's own preference: following the system
        // night mode would disagree with the console the operator just left.
        Theme.init(this);
        setTheme(Theme.isDark() ? R.style.AppTheme_Dark : R.style.AppTheme);
        Palette palette = Theme.p();
        getWindow().setBackgroundDrawable(new ColorDrawable(palette.canvas));
        getWindow().setStatusBarColor(palette.canvas);
        getWindow().setNavigationBarColor(palette.canvas);
        setContentView(R.layout.activity_main);

        prefs = getSharedPreferences(PREFS, MODE_PRIVATE);

        authScroll = findViewById(R.id.auth_scroll);
        stepLogin = findViewById(R.id.step_login);
        stepTwoFactor = findViewById(R.id.step_two_factor);
        inputServer = findViewById(R.id.input_server);
        inputAccount = findViewById(R.id.input_account);
        inputPassword = findViewById(R.id.input_password);
        inputCode = findViewById(R.id.input_code);
        textError = findViewById(R.id.text_error);
        textError2fa = findViewById(R.id.text_error_2fa);
        btnLogin = findViewById(R.id.btn_login);
        btnVerify = findViewById(R.id.btn_verify);
        btnWebLogin = findViewById(R.id.btn_use_web_login);
        btnBackToLogin = findViewById(R.id.btn_back_to_login);
        btnPasskeyLogin = findViewById(R.id.btn_passkey_login);
        btnPasskeyVerify = findViewById(R.id.btn_passkey_verify);
        progress = findViewById(R.id.progress);
        web = findViewById(R.id.webview);
        turnstileBox = findViewById(R.id.turnstile_box);
        turnstileView = findViewById(R.id.turnstile_view);
        turnstileHint = findViewById(R.id.turnstile_hint);

        inputServer.setText(prefs.getString(KEY_SERVER, ""));
        inputAccount.setText(prefs.getString(KEY_USERNAME, ""));

        String deepLink = getIntent().getStringExtra(EXTRA_PATH);
        if (deepLink != null && !deepLink.isEmpty()) {
            pendingPath = deepLink;
        }

        btnLogin.setOnClickListener(v -> doLogin());
        btnVerify.setOnClickListener(v -> doVerify());
        // Passkeys need the platform credential APIs; older devices keep the
        // password form alone.
        btnPasskeyLogin.setVisibility(Passkey.supported() ? View.VISIBLE : View.GONE);
        btnPasskeyLogin.setOnClickListener(v -> doPasskeyLogin());
        btnPasskeyVerify.setOnClickListener(v -> doPasskeyVerify());
        btnWebLogin.setOnClickListener(v -> {
            unloadTurnstile();
            openConsole(true);
        });
        btnBackToLogin.setOnClickListener(v -> backToLogin());

        inputPassword.setOnEditorActionListener((v, actionId, event) -> {
            if (actionId == EditorInfo.IME_ACTION_DONE
                    || (event != null && event.getKeyCode() == KeyEvent.KEYCODE_ENTER)) {
                doLogin();
                return true;
            }
            return false;
        });
        inputServer.setOnFocusChangeListener((v, hasFocus) -> {
            if (!hasFocus) prepareTurnstile();
        });
        inputCode.setOnEditorActionListener((v, actionId, event) -> {
            if (actionId == EditorInfo.IME_ACTION_DONE
                    || (event != null && event.getKeyCode() == KeyEvent.KEYCODE_ENTER)) {
                doVerify();
                return true;
            }
            return false;
        });

        setupWebView();
        setupTurnstileView();

        String server = prefs.getString(KEY_SERVER, "");
        boolean signedIn = !server.isEmpty() && !prefs.getString(KEY_TOKEN, "").isEmpty();
        if (!signedIn) {
            // No app session means any web session left in the WebView is stale.
            // It is what makes /login bounce an authenticated visitor to the
            // landing page, where the app has nothing to sign in with and no way
            // forward.
            clearWebSession();
            prepareTurnstile();
        }
        // The console hands back an expired session here; say so once, on the
        // form the operator is about to use.
        if (getIntent() != null && NOTICE_EXPIRED.equals(getIntent().getStringExtra(EXTRA_NOTICE))) {
            getIntent().removeExtra(EXTRA_NOTICE);
            showError(textError, "登录状态已失效，请重新登录");
        }
        playSplash(signedIn ? this::launchConsole : null);
    }

    // ------------------------------------------------------------------- splash

    /** Once per process: a returning visitor should not sit through it again. */
    private static boolean splashPlayed = false;
    /** How long the artwork stays up before it starts handing over. */
    private static final long SPLASH_HOLD = 1300;
    /** The artwork settles from a slightly tighter crop rather than fading in:
     *  the window behind it is already its own top colour, so any fade would
     *  show the sign-in form first. */
    private static final long SPLASH_SETTLE = 700;
    private static final long SPLASH_FADE = 300;

    /**
     * Plays the launch artwork above whatever the activity has already built.
     *
     * The illustration's first frame is an even colour field that matches the
     * window the system painted before this activity existed, so it can fade in
     * without a seam — and the sign-in form (or the console hand-off) is already
     * working behind it while it plays.
     */
    private void playSplash(Runnable after) {
        if (splashPlayed) {
            if (after != null) after.run();
            return;
        }
        splashPlayed = true;

        final int splashBg = getResources().getColor(
        Theme.isDark() ? R.color.splash_night_bg : R.color.splash_day_bg, getTheme());
        getWindow().setStatusBarColor(splashBg);

        FrameLayout root = findViewById(android.R.id.content);
        final ImageView art = new ImageView(this);
        art.setImageResource(Theme.isDark() ? R.drawable.splash_night : R.drawable.splash_day);
        art.setScaleType(ImageView.ScaleType.CENTER_CROP);
        art.setScaleX(1.05f);
        art.setScaleY(1.05f);
        // Swallow taps while it is up: nothing behind it should react.
        art.setClickable(true);
        root.addView(art, new FrameLayout.LayoutParams(
                FrameLayout.LayoutParams.MATCH_PARENT, FrameLayout.LayoutParams.MATCH_PARENT));

        art.animate().scaleX(1f).scaleY(1f)
                .setDuration(SPLASH_SETTLE).setInterpolator(new DecelerateInterpolator()).start();
        art.postDelayed(() -> art.animate().alpha(0f).setDuration(SPLASH_FADE)
                .withEndAction(() -> {
                    root.removeView(art);
                    // Put the window back the way the rest of the app expects it.
                    Palette p = Theme.p();
                    getWindow().setStatusBarColor(p.canvas);
                    if (after != null) after.run();
                }).start(), SPLASH_HOLD);
    }

    /**
     * Wipes the WebView's cookies and storage.
     *
     * The console's session lives in a cookie plus localStorage, and neither is
     * cleared by dropping the app's own token. Signing out here without this
     * leaves the web page convinced it is still signed in.
     */
    private void clearWebSession() {
        try {
            CookieManager.getInstance().removeAllCookies(null);
            CookieManager.getInstance().flush();
        } catch (Exception ignored) {
        }
        try {
            WebStorage.getInstance().deleteAllData();
        } catch (Exception ignored) {
        }
        try {
            web.stopLoading();
            web.clearCache(true);
            web.clearHistory();
            web.clearFormData();
        } catch (Exception ignored) {
        }
    }

    /** Hands off to the native shell once a session exists. */
    private void launchConsole() {
        if (isFinishing()) return;
        Intent intent = new Intent(this, ConsoleActivity.class);
        // A notification tap lands here first; hand its route on to the shell.
        String route = getIntent().getStringExtra(EXTRA_PATH);
        if (route != null && !route.isEmpty()) {
            intent.putExtra(ConsoleActivity.EXTRA_PATH, route);
        }
        startActivity(intent);
        finish();
    }

    @SuppressLint("SetJavaScriptEnabled")
    private void setupWebView() {
        WebSettings s = web.getSettings();
        s.setJavaScriptEnabled(true);
        s.setDomStorageEnabled(true);
        s.setUseWideViewPort(true);
        s.setLoadWithOverviewMode(true);
        // Self-hosted consoles are routinely reached over plain http on a LAN.
        s.setMixedContentMode(WebSettings.MIXED_CONTENT_ALWAYS_ALLOW);
        s.setUserAgentString(s.getUserAgentString() + " TunnelManagerApp/1.0");

        web.addJavascriptInterface(new PathBridge(), "TMHost");
        web.setWebViewClient(new WebViewClient() {
            @Override
            public void onPageFinished(WebView view, String url) {
                if (injecting) return;
                injecting = true;
                view.evaluateJavascript(sessionScript(), value -> {
                    injecting = false;
                    onSessionInjected(url);
                });
            }
        });
    }

    // --------------------------------------------------------------- turnstile
    //
    // Cloudflare Turnstile only issues a token to pages on a hostname the widget
    // was configured for, so the widget is rendered inside a small WebView whose
    // base URL is the console origin. Everything else about signing in stays
    // native: the token comes back over a JS bridge and goes straight into the
    // credential POST below.

    @SuppressLint("SetJavaScriptEnabled")
    private void setupTurnstileView() {
        WebSettings s = turnstileView.getSettings();
        s.setJavaScriptEnabled(true);
        s.setDomStorageEnabled(true);
        s.setMixedContentMode(WebSettings.MIXED_CONTENT_ALWAYS_ALLOW);
        s.setUserAgentString(s.getUserAgentString() + " TunnelManagerApp/1.0");
        turnstileView.setBackgroundColor(0);
        turnstileView.setVerticalScrollBarEnabled(false);
        turnstileView.setHorizontalScrollBarEnabled(false);
        turnstileView.addJavascriptInterface(new TurnstileBridge(), "TMTurnstile");
    }

    /** Receives the widget's lifecycle from the embedded page. */
    private class TurnstileBridge {
        @JavascriptInterface
        public void ready() {
            ui.post(() -> setTurnstileHint(""));
        }

        @JavascriptInterface
        public void token(String value) {
            ui.post(() -> onTurnstileToken(value));
        }

        @JavascriptInterface
        public void failed(String reason) {
            ui.post(() -> onTurnstileFailed(reason));
        }
    }

    private void setTurnstileHint(String text) {
        if (text == null || text.isEmpty()) {
            turnstileHint.setVisibility(View.GONE);
            return;
        }
        turnstileHint.setText(text);
        turnstileHint.setVisibility(View.VISIBLE);
    }

    /** Renders a fresh widget; any previously issued token is single-use. */
    private void loadTurnstile(String server, String siteKey) {
        turnstileServer = server;
        turnstileSiteKey = siteKey;
        turnstileToken = "";
        turnstileBox.setVisibility(View.VISIBLE);
        setTurnstileHint("正在加载人机验证…");
        turnstileView.loadDataWithBaseURL(server + "/", turnstileHtml(siteKey, Theme.isDark()),
                "text/html", "utf-8", null);
    }

    private void unloadTurnstile() {
        turnstileNeeded = false;
        turnstileToken = "";
        turnstileBox.setVisibility(View.GONE);
        try {
            turnstileView.loadUrl("about:blank");
        } catch (Exception ignored) {
        }
    }

    private void onTurnstileToken(String value) {
        if (value == null || value.isEmpty()) return;
        turnstileToken = value;
        setTurnstileHint("");
        // The operator already pressed 登录, so finish what they started.
        if (pendingPassword != null && !pendingPassword.isEmpty()) {
            String server = turnstileServer;
            String account = inputAccount.getText().toString().trim();
            String password = pendingPassword;
            pendingPassword = null;
            postLogin(server, account, password, value);
        }
    }

    private void onTurnstileFailed(String reason) {
        turnstileToken = "";
        pendingPassword = null;
        setBusy(btnLogin, false, "登录");
        // A solved widget goes stale after five minutes; that is routine, not an
        // error, so quietly ask for a new one.
        if ("expired".equals(reason) && stepLogin.getVisibility() == View.VISIBLE
                && !turnstileServer.isEmpty() && !turnstileSiteKey.isEmpty()) {
            loadTurnstile(turnstileServer, turnstileSiteKey);
            return;
        }
        setTurnstileHint("人机验证加载失败，请改用网页版登录");
    }

    /**
     * Renders the widget as soon as the sign-in form is on screen, so it is
     * already solved by the time 登录 is pressed. Waiting for the first press
     * means the operator watches a cold WebView warm up, which reads as a
     * failure.
     */
    private void prepareTurnstile() {
        final String server = normalizeServer(inputServer.getText().toString());
        if (server.isEmpty()) return;
        new Thread(() -> {
            JSONObject config;
            try {
                config = fetchAuthConfig(server);
            } catch (Exception e) {
                return;
            }
            if (config == null) return;
            final boolean enabled = config.optBoolean("turnstile_enabled", false);
            final String siteKey = config.optString("turnstile_site_key", "").trim();
            ui.post(() -> {
                if (isFinishing() || stepLogin.getVisibility() != View.VISIBLE) return;
                if (!enabled || siteKey.isEmpty()) {
                    unloadTurnstile();
                    return;
                }
                turnstileNeeded = true;
                // Leave a widget that is already up (and possibly solved) alone.
                if (siteKey.equals(turnstileSiteKey) && server.equals(turnstileServer)
                        && turnstileBox.getVisibility() == View.VISIBLE) {
                    return;
                }
                if (pendingPassword != null) return;
                loadTurnstile(server, siteKey);
            });
        }).start();
    }

    private static String turnstileHtml(String siteKey, boolean dark) {
        return "<!doctype html><html><head><meta charset=\"utf-8\">"
                + "<meta name=\"viewport\" content=\"width=device-width,initial-scale=1\">"
                + "<style>html,body{margin:0;padding:0;background:transparent;overflow:hidden}"
                + "body{display:flex;justify-content:center}</style></head><body>"
                + "<div id=\"w\"></div><script>"
                + "var el=document.getElementById('w'),tries=0,shown=false;"
                + "function fail(m){try{TMTurnstile.failed(String(m))}catch(e){}}"
                + "function boot(){if(!window.turnstile){"
                + "if(++tries<150){return setTimeout(boot,100)}return fail('script')}"
                + "try{turnstile.render(el,{sitekey:'" + siteKey + "',action:'login',"
                + "theme:'" + (dark ? "dark" : "light") + "',"
                + "callback:function(t){try{TMTurnstile.token(t)}catch(e){}},"
                + "\"expired-callback\":function(){fail('expired')},"
                + "\"error-callback\":function(c){fail(c||'error')},"
                + "\"timeout-callback\":function(){fail('timeout')}});"
                + "shown=true;TMTurnstile.ready()}catch(e){fail(e&&e.message||e)}}"
                + "var s=document.createElement('script');"
                + "s.src='https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit';"
                + "s.onload=boot;s.onerror=function(){fail('script')};"
                + "document.head.appendChild(s);"
                // A CDN that never answers would otherwise leave the form stuck
                // on "正在加载人机验证…" forever.
                + "setTimeout(function(){if(!shown)fail('timeout')},12000);"
                + "</script></body></html>";
    }

    @Override
    protected void onDestroy() {
        if (web != null) {
            web.removeJavascriptInterface("TMHost");
            web.destroy();
        }
        if (turnstileView != null) {
            turnstileView.removeJavascriptInterface("TMTurnstile");
            turnstileView.destroy();
        }
        super.onDestroy();
    }

    @Override
    public boolean onKeyDown(int keyCode, KeyEvent event) {
        if (keyCode == KeyEvent.KEYCODE_BACK) {
            if (stepTwoFactor.getVisibility() == View.VISIBLE) {
                backToLogin();
                return true;
            }
            if (web.getVisibility() == View.VISIBLE) {
                if (web.canGoBack()) {
                    web.goBack();
                } else {
                    moveTaskToBack(true);
                }
                return true;
            }
        }
        return super.onKeyDown(keyCode, event);
    }

    // ------------------------------------------------------------------ sign in

    private void doLogin() {
        final String server = normalizeServer(inputServer.getText().toString());
        final String account = inputAccount.getText().toString().trim();
        final String password = inputPassword.getText().toString();

        if (server.isEmpty()) {
            showError(textError, "请填写服务器地址");
            inputServer.requestFocus();
            return;
        }
        if (account.isEmpty() || password.isEmpty()) {
            showError(textError, "请填写用户名和密码");
            return;
        }
        hide(textError);
        prefs.edit().putString(KEY_SERVER, server).apply();

        setBusy(btnLogin, true, "连接中…");
        new Thread(() -> {
            try {
                // Ask the server first: this both proves the address is right and
                // says whether a plain credential POST can work at all.
                final JSONObject config = fetchAuthConfig(server);
                ui.post(() -> {
                    boolean turnstile = config != null && config.optBoolean("turnstile_enabled", false);
                    String siteKey = config == null ? "" : config.optString("turnstile_site_key", "").trim();
                    if (!turnstile || siteKey.isEmpty()) {
                        turnstileNeeded = false;
                        postLogin(server, account, password, "");
                        return;
                    }
                    turnstileNeeded = true;
                    if (!turnstileToken.isEmpty() && siteKey.equals(turnstileSiteKey)
                            && server.equals(turnstileServer)) {
                        postLogin(server, account, password, turnstileToken);
                        return;
                    }
                    // Hold the credentials and let the widget finish; the token
                    // callback resumes the sign-in.
                    pendingPassword = password;
                    setBusy(btnLogin, false, "登录");
                    hide(textError);
                    setTurnstileHint("请先完成人机验证");
                    // Restarting a widget that is already mid-solve would throw
                    // away the work it has done so far.
                    boolean up = siteKey.equals(turnstileSiteKey) && server.equals(turnstileServer)
                            && turnstileBox.getVisibility() == View.VISIBLE;
                    if (!up) loadTurnstile(server, siteKey);
                });
            } catch (Exception e) {
                ui.post(() -> {
                    setBusy(btnLogin, false, "登录");
                    showError(textError, networkError(e));
                });
            }
        }).start();
    }

    /** POSTs the credentials, attaching a Turnstile token when the server wants one. */
    private void postLogin(String server, String account, String password, String turnstile) {
        if (isFinishing()) return;
        setBusy(btnLogin, true, "登录中…");
        new Thread(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("account", account);
                body.put("password", password);
                if (turnstile != null && !turnstile.isEmpty()) {
                    body.put("cf_turnstile_response", turnstile);
                }
                final Resp r = postJson(server + "/api/admin/login", body.toString(), null);
                ui.post(() -> {
                    setBusy(btnLogin, false, "登录");
                    onLoginResponse(r, server, account);
                });
            } catch (Exception e) {
                ui.post(() -> {
                    setBusy(btnLogin, false, "登录");
                    showError(textError, networkError(e));
                });
            }
        }).start();
    }

    private void onLoginResponse(Resp r, String server, String account) {
        if (r.code == 200) {
            try {
                JSONObject o = new JSONObject(r.body);
                prefs.edit()
                        .putString(KEY_SERVER, server)
                        .putString(KEY_TOKEN, o.optString("token", ""))
                        .putString(KEY_USERNAME, o.optString("username", account))
                        .putString(KEY_ROLE, o.optString("role", ""))
                        .apply();
                inputPassword.setText("");
                launchConsole();
                return;
            } catch (Exception e) {
                showError(textError, "响应解析失败");
                return;
            }
        }
        if (r.code == 202) {
            try {
                JSONObject o = new JSONObject(r.body);
                if (o.optBoolean("two_factor_required", false)) {
                    challengeToken = o.optString("challenge_token", "");
                    passkeysAvailable = o.optBoolean("passkeys_available", false);
                    btnPasskeyVerify.setVisibility(Passkey.supported() && passkeysAvailable
                            ? View.VISIBLE : View.GONE);
                    stepLogin.setVisibility(View.GONE);
                    stepTwoFactor.setVisibility(View.VISIBLE);
                    hide(textError2fa);
                    inputCode.setText("");
                    inputCode.requestFocus();
                    return;
                }
            } catch (Exception ignored) {
                // fall through to the generic error below
            }
        }
        // The submitted token is single-use, so ask for a fresh one before the
        // operator can meaningfully retry.
        refreshTurnstileAfterAttempt();
        showError(textError, apiError(r));
    }

    private void refreshTurnstileAfterAttempt() {
        if (!turnstileNeeded || turnstileSiteKey.isEmpty() || turnstileServer.isEmpty()) return;
        loadTurnstile(turnstileServer, turnstileSiteKey);
    }

    private void doVerify() {
        final String code = inputCode.getText().toString().trim();
        final String server = prefs.getString(KEY_SERVER, "");
        if (code.isEmpty()) {
            showError(textError2fa, "请输入验证码");
            return;
        }
        if (challengeToken.isEmpty() || server.isEmpty()) {
            showError(textError2fa, "验证会话已失效，请重新登录");
            return;
        }
        hide(textError2fa);
        setBusy(btnVerify, true, "验证中…");

        new Thread(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("challenge_token", challengeToken);
                body.put("code", code);
                final Resp r = postJson(server + "/api/admin/login/2fa", body.toString(), null);
                ui.post(() -> {
                    setBusy(btnVerify, false, "验证");
                    if (r.code == 200) {
                        try {
                            JSONObject o = new JSONObject(r.body);
                            prefs.edit()
                                    .putString(KEY_TOKEN, o.optString("token", ""))
                                    .putString(KEY_USERNAME, o.optString("username", ""))
                                    .putString(KEY_ROLE, o.optString("role", ""))
                                    .apply();
                        } catch (Exception ignored) {
                        }
                        challengeToken = "";
                        inputCode.setText("");
                        launchConsole();
                    } else {
                        showError(textError2fa, apiError(r));
                    }
                });
            } catch (Exception e) {
                ui.post(() -> {
                    setBusy(btnVerify, false, "验证");
                    showError(textError2fa, networkError(e));
                });
            }
        }).start();
    }

    // -------------------------------------------------------------- passkeys

    /**
     * Signs in with a passkey instead of a password.
     *
     * The panel's options are handed to Credential Manager untouched, and the
     * response it produces goes straight back to the finish endpoint, so the
     * challenge never passes through the app.
     */
    private void doPasskeyLogin() {
        final String server = normalizeServer(inputServer.getText().toString());
        final String account = inputAccount.getText().toString().trim();
        if (server.isEmpty()) {
            showError(textError, "请填写服务器地址");
            inputServer.requestFocus();
            return;
        }
        hide(textError);
        prefs.edit().putString(KEY_SERVER, server).apply();
        setBusy(btnPasskeyLogin, true, "等待验证…");

        new Thread(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("account", account);
                final Resp begin = postJson(server + "/api/auth/passkey/login/begin", body.toString(), null);
                if (begin.code != 200) {
                    ui.post(() -> {
                        setBusy(btnPasskeyLogin, false, PASSKEY_LOGIN_LABEL);
                        showError(textError, apiError(begin));
                    });
                    return;
                }
                final JSONObject options = new JSONObject(begin.body);
                ui.post(() -> Passkey.authenticate(this, options.optJSONObject("public_key"),
                        new Passkey.Callback() {
                            @Override
                            public void onResult(JSONObject credential) {
                                finishPasskeyLogin(server, options.optString("ceremony_token", ""), credential);
                            }

                            @Override
                            public void onError(Exception error) {
                                setBusy(btnPasskeyLogin, false, PASSKEY_LOGIN_LABEL);
                                showError(textError, error.getMessage());
                            }
                        }));
            } catch (Exception e) {
                ui.post(() -> {
                    setBusy(btnPasskeyLogin, false, PASSKEY_LOGIN_LABEL);
                    showError(textError, networkError(e));
                });
            }
        }).start();
    }

    private void finishPasskeyLogin(final String server, final String ceremonyToken, JSONObject credential) {
        new Thread(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("ceremony_token", ceremonyToken);
                body.put("credential", credential);
                final Resp r = postJson(server + "/api/auth/passkey/login/finish", body.toString(), null);
                ui.post(() -> {
                    setBusy(btnPasskeyLogin, false, PASSKEY_LOGIN_LABEL);
                    onPasskeyResponse(r, server, textError);
                });
            } catch (Exception e) {
                ui.post(() -> {
                    setBusy(btnPasskeyLogin, false, PASSKEY_LOGIN_LABEL);
                    showError(textError, networkError(e));
                });
            }
        }).start();
    }

    /** Second factor by passkey: the password step already produced the challenge. */
    private void doPasskeyVerify() {
        final String server = prefs.getString(KEY_SERVER, "");
        if (challengeToken.isEmpty() || server.isEmpty()) {
            showError(textError2fa, "验证会话已失效，请重新登录");
            return;
        }
        hide(textError2fa);
        setBusy(btnPasskeyVerify, true, "等待验证…");

        new Thread(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("challenge_token", challengeToken);
                final Resp begin = postJson(server + "/api/admin/login/2fa/passkey/begin", body.toString(), null);
                if (begin.code != 200) {
                    ui.post(() -> {
                        setBusy(btnPasskeyVerify, false, PASSKEY_VERIFY_LABEL);
                        showError(textError2fa, apiError(begin));
                    });
                    return;
                }
                final JSONObject options = new JSONObject(begin.body);
                ui.post(() -> Passkey.authenticate(this, options.optJSONObject("public_key"),
                        new Passkey.Callback() {
                            @Override
                            public void onResult(JSONObject credential) {
                                finishPasskeyVerify(server, options.optString("ceremony_token", ""), credential);
                            }

                            @Override
                            public void onError(Exception error) {
                                setBusy(btnPasskeyVerify, false, PASSKEY_VERIFY_LABEL);
                                showError(textError2fa, error.getMessage());
                            }
                        }));
            } catch (Exception e) {
                ui.post(() -> {
                    setBusy(btnPasskeyVerify, false, PASSKEY_VERIFY_LABEL);
                    showError(textError2fa, networkError(e));
                });
            }
        }).start();
    }

    private void finishPasskeyVerify(final String server, final String ceremonyToken, JSONObject credential) {
        new Thread(() -> {
            try {
                JSONObject body = new JSONObject();
                body.put("challenge_token", challengeToken);
                body.put("ceremony_token", ceremonyToken);
                body.put("credential", credential);
                final Resp r = postJson(server + "/api/admin/login/2fa/passkey/finish", body.toString(), null);
                ui.post(() -> {
                    setBusy(btnPasskeyVerify, false, PASSKEY_VERIFY_LABEL);
                    onPasskeyResponse(r, server, textError2fa);
                });
            } catch (Exception e) {
                ui.post(() -> {
                    setBusy(btnPasskeyVerify, false, PASSKEY_VERIFY_LABEL);
                    showError(textError2fa, networkError(e));
                });
            }
        }).start();
    }

    /** Both passkey paths land here: the finish call returns the session. */
    private void onPasskeyResponse(Resp r, String server, TextView errorView) {
        if (r.code != 200) {
            showError(errorView, apiError(r));
            return;
        }
        try {
            JSONObject o = new JSONObject(r.body);
            prefs.edit()
                    .putString(KEY_SERVER, server)
                    .putString(KEY_TOKEN, o.optString("token", ""))
                    .putString(KEY_USERNAME, o.optString("username", ""))
                    .putString(KEY_ROLE, o.optString("role", ""))
                    .apply();
        } catch (Exception e) {
            showError(errorView, "响应解析失败");
            return;
        }
        challengeToken = "";
        inputPassword.setText("");
        inputCode.setText("");
        launchConsole();
    }

    private void backToLogin() {
        challengeToken = "";
        passkeysAvailable = false;
        btnPasskeyVerify.setVisibility(View.GONE);
        stepTwoFactor.setVisibility(View.GONE);
        stepLogin.setVisibility(View.VISIBLE);
        hide(textError);
    }

    // ------------------------------------------------------------------ console

    /**
     * @param webLogin true to let the console's own /login page handle the sign-in
     *                 (the only way in when the server has Turnstile enabled).
     */
    private void openConsole(boolean webLogin) {
        final String server = prefs.getString(KEY_SERVER, "");
        if (server.isEmpty()) {
            backToLogin();
            showError(textError, "请先填写服务器地址");
            return;
        }
        // In web-login mode there is no token handshake to perform, so the first
        // signed-out path we see later really is a logout.
        bounced = webLogin;
        webLoginMode = webLogin;
        historyArmed = false;
        injecting = false;
        authScroll.setVisibility(View.GONE);
        progress.setVisibility(View.VISIBLE);
        web.setVisibility(View.INVISIBLE);
        if (webLogin) {
            // Hand the whole sign-in over to the console; nothing here should
            // second-guess where it navigates.
            clearWebSession();
            web.setVisibility(View.VISIBLE);
            progress.setVisibility(View.GONE);
        }
        web.loadUrl(server + "/login");
    }

    private void onSessionInjected(String url) {
        String path = pathOf(url);
        if (webLoginMode) {
            progress.setVisibility(View.GONE);
            web.setVisibility(View.VISIBLE);
            if (!isSignedOutPath(path)) dropSignInHistory();
            adoptWebSession();
            return;
        }
        if (isSignedOutPath(path)) {
            if (!bounced) {
                // First pass: the console had no session yet, so nudge it to the
                // dashboard now that the token is in localStorage.
                bounced = true;
                web.loadUrl(prefs.getString(KEY_SERVER, "") + takePendingPath());
                return;
            }
            prefs.edit().remove(KEY_TOKEN).apply();
            showLogin("登录状态已失效，请重新登录");
            return;
        }
        progress.setVisibility(View.GONE);
        web.setVisibility(View.VISIBLE);
        dropSignInHistory();
    }

    /**
     * Signing in is a two-hop dance — /login to seed the token, then the real
     * route — and both hops land in the WebView's back/forward list. A back
     * swipe would therefore walk into /login, which this activity reads as a
     * logout and would clear a perfectly good session. Dropping the list once
     * the console is actually up makes back behave like a native app: it walks
     * in-console history, and backs out of the activity at the root.
     */
    private void dropSignInHistory() {
        if (historyArmed) return;
        historyArmed = true;
        web.clearHistory();
    }

    /** Reads the route a tapped notification asked for, exactly once. */
    private String takePendingPath() {
        String path = pendingPath;
        pendingPath = "";
        return path == null || path.isEmpty() ? "/dashboard" : path;
    }

    @Override
    protected void onNewIntent(Intent intent) {
        super.onNewIntent(intent);
        setIntent(intent);
        String path = intent.getStringExtra(EXTRA_PATH);
        if (path == null || path.isEmpty()) return;
        // A session means the console owns the route: hand it over and let the
        // shell open it, whether or not this activity is still drawing the
        // web-login page.
        if (!prefs.getString(KEY_TOKEN, "").isEmpty()) {
            launchConsole();
            return;
        }
        // Still on the sign-in screen; replay it once the console is up.
        pendingPath = path;
    }

    private void showLogin(String message) {
        web.setVisibility(View.INVISIBLE);
        authScroll.setVisibility(View.VISIBLE);
        progress.setVisibility(View.GONE);
        stepTwoFactor.setVisibility(View.GONE);
        stepLogin.setVisibility(View.VISIBLE);
        inputServer.setText(prefs.getString(KEY_SERVER, ""));
        inputAccount.setText(prefs.getString(KEY_USERNAME, ""));
        inputPassword.setText("");
        if (message != null) {
            showError(textError, message);
        } else {
            hide(textError);
        }
        prepareTurnstile();
    }

    /** Console-side navigation is client routed, so the page never reloads; this
     *  bridge is what lets the native layer notice a logout or a session expiry. */
    private class PathBridge {
        @JavascriptInterface
        public void onPath(final String path) {
            ui.post(() -> {
                if (web.getVisibility() != View.VISIBLE) return;
                boolean signedOut = isSignedOutPath(path);
                if (webLoginMode) {
                    if (!signedOut) {
                        // The console's own form signed in; adopt that session so
                        // the next launch skips the web login entirely.
                        webLoginMode = false;
                        dropSignInHistory();
                        captureTokenIfMissing(MainActivity.this::launchConsole);
                        return;
                    }
                    // With the landing page enabled the console sends a freshly
                    // signed-in visitor to "/", which this activity also reads as
                    // signed out. Adopt the session when there is one; otherwise
                    // stay on the form so the operator can actually sign in.
                    adoptWebSession();
                    return;
                }
                if (signedOut) {
                    // A reload finishes the handshake; during boot the injected
                    // token makes this path transient.
                    if (bounced) {
                        prefs.edit().remove(KEY_TOKEN).apply();
                        showLogin("已退出登录");
                    }
                    return;
                }
                captureTokenIfMissing();
            });
        }

        // -------------------------------------------------- native notifications
        //
        // The console page drives these; the app deliberately never enables
        // alerting on its own, so the operator always knows why a background
        // service is running.

        @JavascriptInterface
        public String notifyState() {
            try {
                JSONObject o = new JSONObject();
                o.put("enabled", prefs.getBoolean(AlertService.KEY_ENABLED, false));
                o.put("permission", notificationsAllowed());
                o.put("battery", ignoringBatteryOptimizations());
                return o.toString();
            } catch (Exception e) {
                return "{}";
            }
        }

        @JavascriptInterface
        public void notifyEnable() {
            ui.post(() -> {
                if (Build.VERSION.SDK_INT < 33 || notificationsAllowed()) {
                    startAlerts();
                    return;
                }
                requestPermissions(new String[]{"android.permission.POST_NOTIFICATIONS"}, REQ_NOTIFY);
            });
        }

        @JavascriptInterface
        public void notifyDisable() {
            ui.post(MainActivity.this::stopAlerts);
        }

        @JavascriptInterface
        public void notifyTest() {
            ui.post(() -> AlertService.postTest(MainActivity.this));
        }

        @JavascriptInterface
        public void notifyBattery() {
            ui.post(() -> {
                try {
                    Intent i = new Intent(Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS);
                    i.setData(Uri.parse("package:" + getPackageName()));
                    startActivity(i);
                } catch (Exception e) {
                    // Some OEM builds hide that screen; the app's own settings
                    // page is the closest thing that always exists.
                    try {
                        startActivity(new Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS,
                                Uri.parse("package:" + getPackageName())));
                    } catch (Exception ignored) {
                    }
                }
            });
        }
    }

    private boolean notificationsAllowed() {
        return checkSelfPermission(android.Manifest.permission.POST_NOTIFICATIONS)
                == PackageManager.PERMISSION_GRANTED;
    }

    private boolean ignoringBatteryOptimizations() {
        PowerManager pm = getSystemService(PowerManager.class);
        return pm != null && pm.isIgnoringBatteryOptimizations(getPackageName());
    }

    private void startAlerts() {
        try {
            Intent i = new Intent(this, AlertService.class);
            i.setAction(ACTION_START_ALERTS);
            startForegroundService(i);
            prefs.edit().putBoolean(AlertService.KEY_ENABLED, true).apply();
        } catch (Exception e) {
            showError(textError, "无法启动后台告警：" + e.getMessage());
        }
        pushHostState();
    }

    private void stopAlerts() {
        stopService(new Intent(this, AlertService.class));
        prefs.edit().putBoolean(AlertService.KEY_ENABLED, false).apply();
        pushHostState();
    }

    @Override
    public void onRequestPermissionsResult(int requestCode, String[] permissions, int[] grantResults) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults);
        if (requestCode != REQ_NOTIFY) return;
        if (grantResults.length > 0 && grantResults[0] == PackageManager.PERMISSION_GRANTED) {
            startAlerts();
        } else {
            pushHostState();
        }
    }

    /** Lets the console page re-read state after something the page did not do. */
    private void pushHostState() {
        if (web == null) return;
        web.evaluateJavascript("window.dispatchEvent(new Event('tmhostchange'))", null);
    }

    /** Covers the web-login path: the console stored a token we never saw. */
    private void captureTokenIfMissing() {
        captureTokenIfMissing(null);
    }

    /**
     * Pulls the token the console's own login form just wrote, if any.
     *
     * Used on the paths this activity reads as "signed out" while a web session
     * exists: without it the WebView sits on the landing page, showing 进入控制台
     * and no login form, while the app has no token of its own to move on with.
     */
    private void adoptWebSession() {
        if (launching) return;
        captureTokenIfMissing(() -> {
            webLoginMode = false;
            dropSignInHistory();
            launching = true;
            launchConsole();
        });
    }

    private void captureTokenIfMissing(Runnable after) {
        if (!prefs.getString(KEY_TOKEN, "").isEmpty()) {
            if (after != null) after.run();
            return;
        }
        web.evaluateJavascript(
                "(function(){try{return JSON.stringify({"
                        + "t:localStorage.getItem('auth_token')||'',"
                        + "u:localStorage.getItem('auth_username')||'',"
                        + "r:localStorage.getItem('auth_role')||''})}catch(e){return ''}})()",
                value -> {
                    if (value == null) return;
                    try {
                        // evaluateJavascript hands back a JSON-encoded string, so
                        // unwrap one layer before reading the payload.
                        Object inner = new JSONTokener(value).nextValue();
                        if (!(inner instanceof String)) return;
                        JSONObject o = new JSONObject((String) inner);
                        String token = o.optString("t", "");
                        if (token.isEmpty()) return;
                        prefs.edit()
                                .putString(KEY_TOKEN, token)
                                .putString(KEY_USERNAME, o.optString("u", ""))
                                .putString(KEY_ROLE, o.optString("r", ""))
                                .apply();
                        if (after != null) after.run();
                    } catch (Exception ignored) {
                    }
                });
    }

    // ------------------------------------------------------------------ helpers

    private static boolean isSignedOutPath(String path) {
        for (String candidate : SIGNED_OUT_PATHS) {
            if (candidate.equals(path)) return true;
        }
        return false;
    }

    private static String pathOf(String url) {
        if (url == null) return "";
        try {
            String path = new URL(url).getPath();
            if (path.isEmpty()) return "/";
            if (path.length() > 1 && path.endsWith("/")) path = path.substring(0, path.length() - 1);
            return path;
        } catch (Exception e) {
            return "";
        }
    }

    static String normalizeServer(String raw) {
        String s = raw == null ? "" : raw.trim();
        if (s.isEmpty()) return "";
        if (!s.startsWith("http://") && !s.startsWith("https://")) {
            s = "http://" + s;
        }
        while (s.endsWith("/")) {
            s = s.substring(0, s.length() - 1);
        }
        return s;
    }

    private String sessionScript() {
        String token = prefs.getString(KEY_TOKEN, "");
        String username = prefs.getString(KEY_USERNAME, "");
        String role = prefs.getString(KEY_ROLE, "");
        return "(function(){"
                + "try{"
                + "if(" + quote(token) + "){"
                + "localStorage.setItem('auth_token'," + quote(token) + ");"
                + "localStorage.setItem('auth_username'," + quote(username) + ");"
                + "localStorage.setItem('auth_role'," + quote(role) + ");"
                + "}"
                + "}catch(e){}"
                + "if(!window.__tmHost){"
                + "window.__tmHost=true;"
                + "var last='';"
                + "var report=function(){var p=location.pathname;if(p!==last){last=p;try{TMHost.onPath(p)}catch(e){}}};"
                + "var ps=history.pushState,rs=history.replaceState;"
                + "history.pushState=function(){ps.apply(this,arguments);report()};"
                + "history.replaceState=function(){rs.apply(this,arguments);report()};"
                + "window.addEventListener('popstate',report);"
                + "setInterval(report,800);"
                + "report();"
                + "}"
                + "})()";
    }

    private static String quote(String value) {
        return JSONObject.quote(value == null ? "" : value);
    }

    private static final class Resp {
        final int code;
        final String body;

        Resp(int code, String body) {
            this.code = code;
            this.body = body;
        }
    }

    private static Resp postJson(String url, String body, String token) throws Exception {
        HttpURLConnection conn = (HttpURLConnection) new URL(url).openConnection();
        try {
            conn.setRequestMethod("POST");
            conn.setConnectTimeout(10000);
            conn.setReadTimeout(20000);
            conn.setDoOutput(true);
            conn.setRequestProperty("Content-Type", "application/json; charset=utf-8");
            conn.setRequestProperty("Accept", "application/json");
            if (token != null && !token.isEmpty()) {
                conn.setRequestProperty("X-Auth-Token", token);
            }
            byte[] payload = body.getBytes(StandardCharsets.UTF_8);
            conn.setFixedLengthStreamingMode(payload.length);
            OutputStream out = conn.getOutputStream();
            try {
                out.write(payload);
            } finally {
                out.close();
            }
            int code = conn.getResponseCode();
            InputStream in = code >= 400 ? conn.getErrorStream() : conn.getInputStream();
            return new Resp(code, readAll(in));
        } finally {
            conn.disconnect();
        }
    }

    /**
     * Reads the public auth config. When Turnstile is on, the login form renders
     * the widget itself (see {@link #loadTurnstile}) instead of handing the whole
     * sign-in to the console's web page.
     */
    private static JSONObject fetchAuthConfig(String server) throws Exception {
        HttpURLConnection conn = (HttpURLConnection) new URL(server + "/api/auth/config").openConnection();
        try {
            conn.setRequestMethod("GET");
            conn.setConnectTimeout(10000);
            conn.setReadTimeout(15000);
            conn.setRequestProperty("Accept", "application/json");
            int code = conn.getResponseCode();
            InputStream in = code >= 400 ? conn.getErrorStream() : conn.getInputStream();
            String body = readAll(in);
            if (code >= 400) return null;
            return new JSONObject(body);
        } finally {
            conn.disconnect();
        }
    }

    private static String readAll(InputStream in) throws Exception {
        if (in == null) return "";
        try {
            ByteArrayOutputStream buffer = new ByteArrayOutputStream();
            byte[] chunk = new byte[4096];
            int read;
            while ((read = in.read(chunk)) != -1) {
                buffer.write(chunk, 0, read);
            }
            return new String(buffer.toByteArray(), StandardCharsets.UTF_8);
        } finally {
            in.close();
        }
    }

    /** The API answers in English; show the operator something readable. */
    private static String apiError(Resp r) {
        String raw = "";
        try {
            raw = new JSONObject(r.body).optString("error", "");
        } catch (Exception ignored) {
        }
        if (raw.contains("invalid credentials")) return "用户名或密码错误";
        if (raw.contains("account is disabled")) return "账号已被禁用";
        if (raw.contains("required")) return "请填写用户名和密码";
        if (raw.contains("turnstile") || raw.contains("verification")) {
            return "人机验证未通过，请重新验证后再试";
        }
        if (raw.contains("temporarily unavailable")) return "服务暂时不可用，请稍后再试";
        if (raw.contains("challenge") || raw.contains("expired")) return "验证码已失效，请重新登录";
        if (!raw.isEmpty()) return raw;
        return "请求失败（HTTP " + r.code + "）";
    }

    private static String networkError(Exception e) {
        String message = e.getMessage() == null ? "" : e.getMessage();
        if (message.contains("ECONNREFUSED") || message.contains("Failed to connect")) {
            return "连不上服务器，请检查地址和端口";
        }
        if (message.contains("timed out") || message.contains("timeout")) {
            return "连接超时，请检查网络";
        }
        if (message.contains("Unable to resolve host")) {
            return "域名解析失败，请检查地址";
        }
        if (message.contains("CLEARTEXT")) {
            return "该地址不允许明文 HTTP 访问";
        }
        return "网络错误：" + (message.isEmpty() ? e.getClass().getSimpleName() : message);
    }

    private void setBusy(Button button, boolean busy, String label) {
        button.setEnabled(!busy);
        button.setText(label);
    }

    private void showError(TextView target, String message) {
        target.setText(message);
        target.setVisibility(View.VISIBLE);
    }

    private static void hide(View view) {
        view.setVisibility(View.GONE);
    }
}
