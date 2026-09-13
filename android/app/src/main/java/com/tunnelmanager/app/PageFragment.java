package com.tunnelmanager.app;

import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.fragment.app.Fragment;

/**
 * A console page.
 *
 * Pages build their views in code rather than in XML: every colour comes from
 * {@link Theme}, and threading a runtime palette through layout inflation means
 * writing the same value twice in two languages.
 */
public abstract class PageFragment extends Fragment {

    @Nullable
    @Override
    public final View onCreateView(@NonNull LayoutInflater inflater, @Nullable ViewGroup container,
                                   @Nullable Bundle savedInstanceState) {
        return build(inflater, container);
    }

    protected abstract View build(@NonNull LayoutInflater inflater, @Nullable ViewGroup container);

    /** Called when the page is brought back to the front. */
    void onShown() {
    }

    /** Console route this page corresponds to, for the shell's active state. */
    abstract String route();

    /**
     * False once the page has been detached.
     *
     * Every page repaints from a network callback, and those can land after the
     * operator has already navigated away. Building a view needs a context, so
     * without this the late callback crashes the app.
     */
    protected boolean alive() {
        return isAdded() && getContext() != null;
    }

    /** Navigates within the console, unless the page has already been left. */
    protected void openRoute(String route) {
        if (!alive()) return;
        console().open(route);
    }

    protected ConsoleActivity console() {
        return (ConsoleActivity) requireActivity();
    }
}
