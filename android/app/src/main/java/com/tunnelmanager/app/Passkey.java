package com.tunnelmanager.app;

import android.content.Context;
import android.os.Build;

import androidx.credentials.Credential;
import androidx.credentials.CredentialManager;
import androidx.credentials.CredentialManagerCallback;
import androidx.credentials.CreateCredentialResponse;
import androidx.credentials.CreatePublicKeyCredentialRequest;
import androidx.credentials.CreatePublicKeyCredentialResponse;
import androidx.credentials.GetCredentialRequest;
import androidx.credentials.GetCredentialResponse;
import androidx.credentials.GetPublicKeyCredentialOption;
import androidx.credentials.PublicKeyCredential;
import androidx.credentials.exceptions.CreateCredentialCancellationException;
import androidx.credentials.exceptions.CreateCredentialException;
import androidx.credentials.exceptions.CreateCredentialInterruptedException;
import androidx.credentials.exceptions.CreateCredentialProviderConfigurationException;
import androidx.credentials.exceptions.CreateCredentialUnsupportedException;
import androidx.credentials.exceptions.GetCredentialCancellationException;
import androidx.credentials.exceptions.GetCredentialException;
import androidx.credentials.exceptions.GetCredentialInterruptedException;
import androidx.credentials.exceptions.GetCredentialProviderConfigurationException;
import androidx.credentials.exceptions.GetCredentialUnsupportedException;
import androidx.credentials.exceptions.NoCredentialException;

import org.json.JSONObject;

/**
 * Passkeys through Android's Credential Manager.
 *
 * The panel hands out the bare WebAuthn dictionaries, which is exactly the shape
 * Credential Manager consumes, so the options and the responses travel as opaque
 * JSON in both directions and the app never re-encodes a challenge.
 *
 * The library only exposes Kotlin suspend functions, so Java callers use the
 * async variants; results and failures come back on the main executor.
 */
final class Passkey {

    private Passkey() {
    }

    /**
     * Passkeys need the platform credential APIs. Android 9 is the floor for
     * {@code CredentialManager}; older devices keep the password form.
     */
    static boolean supported() {
        return Build.VERSION.SDK_INT >= Build.VERSION_CODES.P;
    }

    /** Delivers the browser-style response JSON, or a readable failure. */
    interface Callback {
        void onResult(JSONObject credential);

        void onError(Exception error);
    }

    /** Raised when the user dismisses the system sheet. */
    static final class Cancelled extends Exception {
        Cancelled(String message) {
            super(message);
        }
    }

    /** Creates a new credential; {@code options} is the panel's public_key. */
    static void register(Context ctx, JSONObject options, Callback callback) {
        CredentialManager manager = CredentialManager.create(ctx);
        CreatePublicKeyCredentialRequest request =
                new CreatePublicKeyCredentialRequest(options.toString());
        // The async API is typed on the base response; the public-key variant is
        // the only thing this request can produce.
        manager.createCredentialAsync(ctx, request, null, ctx.getMainExecutor(),
                new CredentialManagerCallback<CreateCredentialResponse, CreateCredentialException>() {
                    @Override
                    public void onResult(CreateCredentialResponse result) {
                        if (!(result instanceof CreatePublicKeyCredentialResponse)) {
                            callback.onError(new Exception("系统返回的凭据类型不正确"));
                            return;
                        }
                        try {
                            callback.onResult(new JSONObject(
                                    ((CreatePublicKeyCredentialResponse) result).getRegistrationResponseJson()));
                        } catch (Exception e) {
                            callback.onError(new Exception("系统返回的凭据无法解析"));
                        }
                    }

                    @Override
                    public void onError(CreateCredentialException e) {
                        callback.onError(translateCreate(e));
                    }
                });
    }

    /** Asserts an existing credential; {@code options} is the panel's public_key. */
    static void authenticate(Context ctx, JSONObject options, Callback callback) {
        CredentialManager manager = CredentialManager.create(ctx);
        GetCredentialRequest request = new GetCredentialRequest.Builder()
                .addCredentialOption(new GetPublicKeyCredentialOption(options.toString()))
                .build();
        manager.getCredentialAsync(ctx, request, null, ctx.getMainExecutor(),
                new CredentialManagerCallback<GetCredentialResponse, GetCredentialException>() {
                    @Override
                    public void onResult(GetCredentialResponse result) {
                        Credential credential = result.getCredential();
                        if (!(credential instanceof PublicKeyCredential)) {
                            callback.onError(new Exception("系统返回的凭据类型不正确"));
                            return;
                        }
                        try {
                            callback.onResult(new JSONObject(
                                    ((PublicKeyCredential) credential).getAuthenticationResponseJson()));
                        } catch (Exception e) {
                            callback.onError(new Exception("系统返回的凭据无法解析"));
                        }
                    }

                    @Override
                    public void onError(GetCredentialException e) {
                        callback.onError(translateGet(e));
                    }
                });
    }

    private static Exception translateCreate(CreateCredentialException e) {
        if (e instanceof CreateCredentialCancellationException) return new Cancelled("已取消通行密钥绑定");
        if (e instanceof CreateCredentialProviderConfigurationException) {
            return new Exception("本机缺少 Google Play 服务，无法使用通行密钥");
        }
        if (e instanceof CreateCredentialUnsupportedException) return new Exception("本机不支持通行密钥");
        if (e instanceof CreateCredentialInterruptedException) return new Exception("通行密钥绑定被中断，请重试");
        return new Exception(describe(e, "通行密钥绑定失败"));
    }

    private static Exception translateGet(GetCredentialException e) {
        if (e instanceof GetCredentialCancellationException) return new Cancelled("已取消通行密钥验证");
        if (e instanceof NoCredentialException) return new Exception("没有可用的通行密钥，请先在账户页绑定");
        if (e instanceof GetCredentialProviderConfigurationException) {
            return new Exception("本机缺少 Google Play 服务，无法使用通行密钥");
        }
        if (e instanceof GetCredentialUnsupportedException) return new Exception("本机不支持通行密钥");
        if (e instanceof GetCredentialInterruptedException) return new Exception("通行密钥验证被中断，请重试");
        return new Exception(describe(e, "通行密钥验证失败"));
    }

    /**
     * The association failure is the one worth naming: it means the panel does
     * not serve assetlinks.json for this app yet, which no amount of retrying
     * will fix.
     */
    private static String describe(Exception e, String fallback) {
        String message = e.getMessage() == null ? "" : e.getMessage();
        if (message.contains("rpId") || message.contains("origin") || message.contains("asset")
                || message.contains("privileged") || message.contains("association")) {
            return "该域名尚未与本 App 关联：请在面板「系统设置 → 通行密钥」配置 Android 资产链接";
        }
        return message.isEmpty() ? fallback : fallback + "：" + message;
    }
}
