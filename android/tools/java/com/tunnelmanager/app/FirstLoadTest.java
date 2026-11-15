package com.tunnelmanager.app;

import android.content.Context;
import android.content.SharedPreferences;
import android.os.Looper;
import android.os.Bundle;
import org.json.JSONObject;
import org.json.JSONArray;
import java.io.*;
import java.net.*;
import java.util.*;
import java.util.concurrent.*;

final class FirstLoadTest {
    private static final List<String> sent = Collections.synchronizedList(new ArrayList<>());
    private static volatile CountDownLatch release = new CountDownLatch(1);
    private static volatile int responseStatus=200;

    public static void main(String[] args) {
        try { run(args); System.exit(0); }
        catch (Throwable failure) { failure.printStackTrace(); System.exit(1); }
    }

    private static void run(String[] args) throws Exception {
        URL.setURLStreamHandlerFactory(protocol -> new URLStreamHandler() {
            protected URLConnection openConnection(URL url) { return new Connection(url); }
        });
        Preferences preferences = new Preferences();
        Context context = new Context() { public SharedPreferences getSharedPreferences(String key, int mode) { return preferences; } };
        Session.init(context);
        Session.save(context, "https://fixture.invalid", "fixture-token", "fixture-admin", "admin");
        Class<?> type = Class.forName("com.tunnelmanager.app." + args[0] + "Load");
        Host owner = (Host) type.getDeclaredConstructor().newInstance();
        java.lang.reflect.Method enter = type.getDeclaredMethod("enter"), pending = type.getDeclaredMethod("pending"), refresh=type.getDeclaredMethod("refresh");
        enter.setAccessible(true); pending.setAccessible(true); refresh.setAccessible(true);
        java.lang.reflect.Method write=type.getDeclaredMethod("gateWrite", boolean.class), writing=type.getDeclaredMethod("writing"), eligible=type.getDeclaredMethod("eligible");
        write.setAccessible(true); writing.setAccessible(true); eligible.setAccessible(true);
        int expected = Integer.parseInt(args[1]);

        if (args[0].equals("SettingsFragment")) {
            Bundle beforeLoad=new Bundle(); owner.onSaveInstanceState(beforeLoad);
            check(!beforeLoad.containsKey("settingsConfig"), "an unloaded empty config cannot become a loaded restore snapshot");
            Bundle legacy=new Bundle(); legacy.putString("settingsConfig", "{}"); owner.onCreate(legacy);
        }
        enter.invoke(owner);
        await(() -> sent.size() > 0, "first page entry never dispatched an HTTP request");
        check((boolean) pending.invoke(owner), "initial read must remain pending before its HTTP response");
        check(!(boolean)eligible.invoke(owner), "pending read must disable toolbar refresh eligibility");
        check(!(boolean)write.invoke(owner,false), "ordinary write must be rejected while the earlier GET is pending");
        if (args[0].equals("AccountFragment")) check(!(boolean)write.invoke(owner,true), "passkey write must be rejected while GET is pending");
        refresh.invoke(owner);
        release.countDown();
        await(() -> { Looper.drain(); return !isPending(pending, owner); }, "page never finishes its initial load");
        check(sent.size() == expected, "duplicate in-flight reads or missing initial sections: " + sent);

        if (args[0].equals("SettingsFragment")) {
            owner.nameDraft="unsaved site name";
            Bundle saved=new Bundle(); owner.onSaveInstanceState(saved);
            check(saved.containsKey("settingsConfig"), "completed config must be available for draft restoration");
            Host restored=(Host)type.getDeclaredConstructor().newInstance(); restored.onCreate(saved); enter.invoke(restored);
            check(sent.size()==expected, "restoring an already loaded settings draft must not reload its configuration");
            check(restored.nameDraft.equals("unsaved site name"), "restored settings draft was overwritten");
        }
        refresh.invoke(owner);
        await(() -> { Looper.drain(); return !isPending(pending, owner) && sent.size() >= expected * 2; }, "completed page cannot refresh");
        check(sent.size() == expected * 2, "refresh did not dispatch one complete read set");
        check((boolean)eligible.invoke(owner), "completed load must permit toolbar refresh");
        for (boolean passkey : args[0].equals("AccountFragment") ? new boolean[]{false,true} : new boolean[]{false}) {
            release=new CountDownLatch(1);
            int beforeWrite=sent.size();
            check((boolean)write.invoke(owner,passkey), "idle page must permit a write");
            await(() -> sent.size()==beforeWrite+1, "guarded write never dispatched real HTTP");
            check(sent.get(beforeWrite).equals("/api/fixture/write"), "write must use its own HTTP request");
            check(!(boolean)eligible.invoke(owner), "ordinary/passkey write must disable toolbar refresh eligibility");
            refresh.invoke(owner);
            check(!(boolean)pending.invoke(owner), "write busy state must reject a new load before entering loading");
            check(!(boolean)write.invoke(owner,false), "concurrent ordinary write must be rejected");
            if (args[0].equals("AccountFragment")) check(!(boolean)write.invoke(owner,true), "concurrent passkey write must be rejected");
            check(sent.size()==beforeWrite+1, "busy write must not dispatch refresh or another write");
            release.countDown();
            await(() -> {Looper.drain();return !isPending(writing,owner);}, "HTTP completion never leaves its ordinary/passkey busy state");
            check((boolean)eligible.invoke(owner), "completed write must restore toolbar refresh eligibility");
            refresh.invoke(owner);
            await(() -> {Looper.drain();return !isPending(pending,owner)&&sent.size()==beforeWrite+1+expected;}, "page cannot refresh after its write completes");
        }
        responseStatus=500;
        Host failed=(Host)type.getDeclaredConstructor().newInstance();
        int beforeFailure=sent.size(); enter.invoke(failed);
        await(() -> {Looper.drain();return !isPending(pending,failed)&&sent.size()>beforeFailure;}, "failed initial HTTP read did not clear its pending state");
        responseStatus=200;
        int beforeRetry=sent.size(); enter.invoke(failed);
        await(() -> {Looper.drain();return !isPending(pending,failed)&&sent.size()>=beforeRetry+expected;}, "initial failure could not retry through the build entry point");
        owner.active = false;
        int beforeDetach=sent.size(); refresh.invoke(owner);
        check(sent.size() == beforeDetach, "detached page dispatched a new read");
    }

    private static boolean isPending(java.lang.reflect.Method method, Host owner) {
        try { return (boolean) method.invoke(owner); } catch (Exception error) { throw new AssertionError(error); }
    }
    private static void await(java.util.function.BooleanSupplier condition, String message) throws InterruptedException {
        long until = System.nanoTime() + TimeUnit.SECONDS.toNanos(5);
        while (!condition.getAsBoolean() && System.nanoTime() < until) Thread.sleep(1);
        check(condition.getAsBoolean(), message);
    }
    private static void check(boolean value, String message) { if (!value) throw new AssertionError(message); }

    private static final class Connection extends HttpURLConnection {
        Connection(URL url) { super(url); }
        public void connect(){} public void disconnect(){} public boolean usingProxy(){return false;}
        public int getResponseCode() throws IOException {
            sent.add(url.getPath());
            try { check(release.await(5, TimeUnit.SECONDS), "HTTP response gate timed out"); }
            catch (InterruptedException error) { throw new IOException(error); }
            return responseStatus;
        }
        public OutputStream getOutputStream(){return new ByteArrayOutputStream();}
        public InputStream getInputStream(){return new ByteArrayInputStream("{site_name:Fixture}".getBytes());}
        public InputStream getErrorStream(){return getInputStream();}
    }

    /** UI-only host methods are no-ops; actual load fields, Api and Session are unchanged. */
    static class Host {
        boolean active = true, statusOk, bannerError;
        String statusMessage="",cfError="",nicknameDraft="",avatarUrl="",avatarBitmapUrl="",encKeySource="",banner="",statusError="";
        Object avatarBitmap;
        JSONObject config=new JSONObject(),me=new JSONObject(),tf=new JSONObject(),cf=new JSONObject(),pk=new JSONObject(),settings=new JSONObject(),status=new JSONObject();
        JSONArray users=new JSONArray(),groups=new JSONArray(),invites=new JSONArray();
        String nameDraft="",descriptionDraft="",iconDraft="",panelHostDraft="",preferredDraft="",fallbackDraft="";
        boolean landingEnabled; final List<String[]> presets=new ArrayList<>();
        final android.os.Handler ui=new android.os.Handler(Looper.getMainLooper()); final Runnable pollTask=()->{}; final long STATUS_INTERVAL=30000;
        public void onCreate(Bundle saved){} public void onSaveInstanceState(Bundle saved){} void capture(){}
        boolean alive(){return active;} boolean canRefresh(){return true;} void render(){} void loadAvatar(){} void loadIcon(){}
        void applyConfig(JSONObject value){} void applySettings(JSONObject value){} void applySettingsDrafts(){}
        void applySmtp(JSONObject value){} void applyOauth(JSONObject value){} void applySysPayload(JSONObject value){}
        JSONObject orEmpty(JSONObject value){return value==null?new JSONObject():value;}
        JSONArray array(JSONObject value,String key){return new JSONArray();}
        Console console(){return new Console();}
    }
    static final class Console { void rebuildNav(){} }
    private static final class Preferences implements SharedPreferences {
        final Map<String,String> values=new HashMap<>();
        final List<OnSharedPreferenceChangeListener> listeners=new ArrayList<>();
        public Map<String,?> getAll(){return new HashMap<>(values);}
        public String getString(String key,String fallback){return values.getOrDefault(key,fallback);}
        public void registerOnSharedPreferenceChangeListener(OnSharedPreferenceChangeListener listener){listeners.add(listener);}
        public Editor edit(){return new Editor(){
            final Map<String,String> updates=new HashMap<>();
            public Editor putString(String key,String value){updates.put(key,value);return this;}
            public Editor remove(String key){updates.put(key,null);return this;}
            public void apply(){updates.forEach((key,value)->{if(value==null)values.remove(key);else values.put(key,value);});
                updates.keySet().forEach(key->listeners.forEach(listener->listener.onSharedPreferenceChanged(Preferences.this,key)));}
        };}
    }
}
