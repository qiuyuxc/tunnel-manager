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
        // Building the request and reaching Credential Manager can both fail
        // before any callback exists. Letting that escape leaves the caller
        // waiting forever with its busy flag set, and the UI goes quiet instead
        // of saying what went wrong.
        CredentialManager manager;
        CreatePublicKeyCredentialRequest request;
        try {
            manager = CredentialManager.create(ctx);
            request = new CreatePublicKeyCredentialRequest(options.toString());
        } catch (Exception e) {
            callback.onError(new Exception(I18n.text("m_c372e2203bda") + describe(e, I18n.text("m_47a2243e45b3"))));
            return;
        }
        manager.createCredentialAsync(ctx, request, null, ctx.getMainExecutor(),
                new CredentialManagerCallback<CreateCredentialResponse, CreateCredentialException>() {
                    @Override
                    public void onResult(CreateCredentialResponse result) {
                        if (!(result instanceof CreatePublicKeyCredentialResponse)) {
                            callback.onError(new Exception(I18n.text("m_945d6aff9026")));
                            return;
                        }
                        try {
                            callback.onResult(new JSONObject(
                                    ((CreatePublicKeyCredentialResponse) result).getRegistrationResponseJson()));
                        } catch (Exception e) {
                            callback.onError(new Exception(I18n.text("m_e10479baa666")));
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
        CredentialManager manager;
        GetCredentialRequest request;
        try {
            manager = CredentialManager.create(ctx);
            request = new GetCredentialRequest.Builder()
                    .addCredentialOption(new GetPublicKeyCredentialOption(options.toString()))
                    .build();
        } catch (Exception e) {
            callback.onError(new Exception(I18n.text("m_7e2fac887b1c") + describe(e, I18n.text("m_47a2243e45b3"))));
            return;
        }
        manager.getCredentialAsync(ctx, request, null, ctx.getMainExecutor(),
                new CredentialManagerCallback<GetCredentialResponse, GetCredentialException>() {
                    @Override
                    public void onResult(GetCredentialResponse result) {
                        Credential credential = result.getCredential();
                        if (!(credential instanceof PublicKeyCredential)) {
                            callback.onError(new Exception(I18n.text("m_945d6aff9026")));
                            return;
                        }
                        try {
                            callback.onResult(new JSONObject(
                                    ((PublicKeyCredential) credential).getAuthenticationResponseJson()));
                        } catch (Exception e) {
                            callback.onError(new Exception(I18n.text("m_e10479baa666")));
                        }
                    }

                    @Override
                    public void onError(GetCredentialException e) {
                        callback.onError(translateGet(e));
                    }
                });
    }

    private static Exception translateCreate(CreateCredentialException e) {
        if (e instanceof CreateCredentialCancellationException) return new Cancelled(I18n.text("m_f951ae00348b"));
        if (e instanceof CreateCredentialProviderConfigurationException) {
            return new Exception(I18n.text("m_64740dcb4882"));
        }
        if (e instanceof CreateCredentialUnsupportedException) return new Exception(I18n.text("m_7d33037e3d4f"));
        if (e instanceof CreateCredentialInterruptedException) return new Exception(I18n.text("m_4fe45e47037a"));
        return new Exception(describe(e, I18n.text("m_b5624c708f47")));
    }

    private static Exception translateGet(GetCredentialException e) {
        if (e instanceof GetCredentialCancellationException) return new Cancelled(I18n.text("m_db900a083e3e"));
        if (e instanceof NoCredentialException) return new Exception(I18n.text("m_b0c5651bf54f"));
        if (e instanceof GetCredentialProviderConfigurationException) {
            return new Exception(I18n.text("m_64740dcb4882"));
        }
        if (e instanceof GetCredentialUnsupportedException) return new Exception(I18n.text("m_7d33037e3d4f"));
        if (e instanceof GetCredentialInterruptedException) return new Exception(I18n.text("m_434dcc2c864f"));
        return new Exception(describe(e, I18n.text("m_17c96c45fc4c")));
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
            return I18n.text("m_058ecbbb9805");
        }
        return message.isEmpty() ? fallback : fallback + "：" + message;
    }
}
