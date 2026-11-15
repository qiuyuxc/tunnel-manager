package com.tunnelmanager.app;

import android.content.Context;
import android.content.SharedPreferences;
import android.os.Looper;
import android.os.Bundle;
import org.json.JSONObject;
import org.json.JSONArray;
import java.nio.charset.StandardCharsets;
import java.io.*;
import java.net.*;
import java.util.*;
import java.util.concurrent.*;

final class FirstLoadTest {
    private static final List<String> sent = Collections.synchronizedList(new ArrayList<>());
    private static volatile CountDownLatch release = new CountDownLatch(1);
    private static volatile int responseStatus=200;
    private static volatile String responseLabel="server";
    private static final java.util.concurrent.atomic.AtomicInteger oldReadResponses=new java.util.concurrent.atomic.AtomicInteger();

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
        int expected = Integer.parseInt(args[1]);

        if (args[0].equals("SettingsFragment")) {
            Bundle beforeLoad=new Bundle(); owner.onSaveInstanceState(beforeLoad);
            check(!beforeLoad.containsKey("settingsConfig"), "an unloaded empty config cannot become a loaded restore snapshot");
            Bundle legacy=new Bundle(); legacy.putString("settingsConfig", "{}"); owner.onCreate(legacy);
        }
        enter.invoke(owner);
        await(() -> sent.size() > 0, "first page entry never dispatched an HTTP request");
        check((boolean) pending.invoke(owner), "initial read must remain pending before its HTTP response");
        if(args[0].equals("AccountFragment")){
            check(!invokeGate(type,owner,"beginWrite")&&!invokeGate(type,owner,"beginPasskeyWrite"),"an in-flight account read accepted a write");
            check(!invokeGate(type,owner,"canRefresh"),"pending account read exposed refresh");
        }
        refresh.invoke(owner);
        release.countDown();
        await(() -> { Looper.drain(); return !isPending(pending, owner); }, "page never finishes its initial load");
        check(sent.size() == expected, "duplicate in-flight reads or missing initial sections: " + sent);

        if(args[0].equals("AccountFragment")){
            check(invokeGate(type,owner,"beginWrite"),"an idle account rejected its write");
            int before=sent.size();refresh.invoke(owner);check(sent.size()==before&&!invokeGate(type,owner,"canRefresh"),"saving account dispatched a refresh");
            setFlag(type,owner,"busy",false);
            check(invokeGate(type,owner,"beginPasskeyWrite"),"an idle account rejected its passkey operation");
            refresh.invoke(owner);check(sent.size()==before&&!invokeGate(type,owner,"canRefresh"),"pending passkey operation dispatched a refresh");
            setFlag(type,owner,"pkBusy",false);
        }
        if (args[0].equals("SettingsFragment")) {
            owner.nameDraft="unsaved site name";
            if(expected==5){owner.smtpHost="unsaved.smtp.example";owner.oauthClientId="unsaved-client";owner.turnstileSiteKey="unsaved-site-key";owner.labMaxWorkers=17;}
            Bundle saved=new Bundle(); owner.onSaveInstanceState(saved);
            check(saved.containsKey("settingsConfig"), "completed config must be available for draft restoration");
            if(expected==5){
                check(saved.containsKey("settingsSystem"),"all completed system sections must be cached");
                JSONObject snapshot=new JSONObject(saved.getString("settingsSystem"));
                check(snapshot.has("settings")&&snapshot.has("smtp")&&snapshot.has("oauth")&&snapshot.has("key"),"system snapshot omitted a section");
                check(!snapshot.optJSONObject("smtp").has("password")&&!snapshot.optJSONObject("oauth").has("client_secret")&&!snapshot.optJSONObject("settings").has("turnstile_secret_key"),"snapshot persisted a secret");
            }
            Host restored=(Host)type.getDeclaredConstructor().newInstance(); restored.onCreate(saved); enter.invoke(restored);
            check(sent.size()==expected, "restoring an already loaded settings draft must not reload its configuration");
            check(restored.nameDraft.equals("unsaved site name"), "restored settings draft was overwritten");
            if(expected==5){
                check(restored.smtpHost.equals("unsaved.smtp.example")&&restored.oauthClientId.equals("unsaved-client")&&restored.turnstileSiteKey.equals("unsaved-site-key")&&restored.labMaxWorkers==17,"restored system drafts were overwritten");
                Bundle partial=new Bundle();partial.putString("settingsConfig",saved.getString("settingsConfig"));
                Host missing=(Host)type.getDeclaredConstructor().newInstance();missing.onCreate(partial);missing.nameDraft="preserved branding";enter.invoke(missing);
                await(()->{Looper.drain();return sent.size()==expected+4;},"a branding-only snapshot must fetch the four missing system sections");
                await(()->{Looper.drain();try{java.lang.reflect.Field loaded=type.getDeclaredField("sysLoaded");loaded.setAccessible(true);return loaded.getBoolean(missing);}catch(Exception e){throw new AssertionError(e);}},"system-only read did not finish");
                check(missing.nameDraft.equals("preserved branding"),"fetching missing system sections overwrote branding drafts");
                sent.subList(expected,sent.size()).clear();
            }
        }
        refresh.invoke(owner);
        await(() -> { Looper.drain(); return !isPending(pending, owner) && sent.size() >= expected * 2; }, "completed page cannot refresh");
        check(sent.size() == expected * 2, "refresh did not dispatch one complete read set");
        responseStatus=500;
        Host failed=(Host)type.getDeclaredConstructor().newInstance();
        int beforeFailure=sent.size(); enter.invoke(failed);
        await(() -> {Looper.drain();return !isPending(pending,failed)&&sent.size()>beforeFailure;}, "failed initial HTTP read did not clear its pending state");
        responseStatus=200;
        int beforeRetry=sent.size(); enter.invoke(failed);
        await(() -> {Looper.drain();return !isPending(pending,failed)&&sent.size()>=beforeRetry+expected;}, "initial failure could not retry through the build entry point");
        if(args[0].equals("SettingsFragment")&&expected==5){
            Host racing=(Host)type.getDeclaredConstructor().newInstance();
            CountDownLatch oldRead=new CountDownLatch(1);release=oldRead;responseLabel="old";
            int raceStart=sent.size();enter.invoke(racing);
            await(()->sent.size()==raceStart+5,"pending read did not reach all five sections");
            responseLabel="new";release=new CountDownLatch(0);
            java.lang.reflect.Method mutate=type.getDeclaredMethod("mutate");mutate.setAccessible(true);mutate.invoke(racing);
            await(()->{Looper.drain();return racing.smtpHost.equals("new.smtp.example");},"mutation did not refresh its four system sections while old GET was pending");
            check(sent.size()==raceStart+10,"write must send one PUT and a fresh four-section read: "+sent);
            oldRead.countDown();
            await(()->oldReadResponses.get()==5,"old HTTP read did not finish");
            await(()->Looper.pending()>0,"discarded read did not schedule its completion");Looper.drain();
            check(racing.smtpHost.equals("new.smtp.example"),"a late read overwrote the saved system configuration");
            check(!isPending(pending,racing),"discarded read left the page pending");
        }
        owner.active = false;
        int beforeDetach=sent.size(); refresh.invoke(owner);
        check(sent.size() == beforeDetach, "detached page dispatched a new read");
    }

    private static boolean invokeGate(Class<?> type,Host owner,String name)throws Exception{java.lang.reflect.Method method=type.getDeclaredMethod(name);method.setAccessible(true);return (boolean)method.invoke(owner);}
    private static void setFlag(Class<?> type,Host owner,String name,boolean value)throws Exception{java.lang.reflect.Field field=type.getDeclaredField(name);field.setAccessible(true);field.setBoolean(owner,value);}

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
        final CountDownLatch gate;final String label;
        Connection(URL url) { super(url);gate=release;label=responseLabel; }
        public void connect(){} public void disconnect(){} public boolean usingProxy(){return false;}
        public int getResponseCode() throws IOException {
            sent.add(url.getPath());
            try { check(gate.await(5, TimeUnit.SECONDS), "HTTP response gate timed out"); }
            catch (InterruptedException error) { throw new IOException(error); }
            return responseStatus;
        }
        public OutputStream getOutputStream(){return new ByteArrayOutputStream();}
        public InputStream getInputStream(){
            if(label.equals("old")&&getRequestMethod().equals("GET"))oldReadResponses.incrementAndGet();
            JSONObject body=new JSONObject();String path=url.getPath();
            if(path.equals("/api/config"))body.put("site_name","Fixture");
            if(path.equals("/api/admin/settings"))body.put("turnstile_site_key","server-site-key").put("turnstile_secret_key","never-cache").put("lab_max_workers",32);
            if(path.equals("/api/admin/smtp"))body.put("host",label+".smtp.example").put("password","never-cache").put("configured",true);
            if(path.equals("/api/admin/oauth"))body.put("client_id","server-client").put("client_secret","never-cache").put("has_client_secret",true);
            if(path.equals("/api/admin/encryption-key"))body.put("source","database");
            return new ByteArrayInputStream(body.toString().getBytes(StandardCharsets.UTF_8));
        }
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
        String turnstileSiteKey="",passkeyRPID="",passkeyOrigins="",androidPackage="",androidFingerprints="",androidFingerprintsEffective="",rateLimitPerAccount="",rateLimitPerIP="",rateLimitWindow="",rateLimitFamiliar="",oauthClientId="",oauthRedirect="",oauthScopes="",smtpHost="",smtpPort="",smtpUsername="",smtpFrom="",testMailTo="",smtpPassword="",oauthClientSecret="",turnstileSecret="";
        boolean turnstileEnabled,experimentalEnabled,passwordLoginDisabled,rateLimitEnabled,rateLimitNotify,turnstileHasSecret,smtpReady,oauthHasSecret,passkeyAdminReady;
        int smtpTls,labDailyRequestLimit,labRequestsPerSecond,labMaxWorkers;
        boolean landingEnabled; final List<String[]> presets=new ArrayList<>();
        final android.os.Handler ui=new android.os.Handler(Looper.getMainLooper()); final Runnable pollTask=()->{}; final long STATUS_INTERVAL=30000;
        public void onCreate(Bundle saved){} public void onSaveInstanceState(Bundle saved){} void capture(){}
        boolean alive(){return active;} void render(){} void loadAvatar(){} void loadIcon(){}
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
