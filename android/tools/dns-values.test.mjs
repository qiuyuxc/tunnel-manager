import { after, before, test } from 'node:test'
import assert from 'node:assert/strict'
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, resolve, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawnSync } from 'node:child_process'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const directory = mkdtempSync(join(tmpdir(), 'dns-values-'))
const executable = name => process.env.JAVA_HOME ? join(process.env.JAVA_HOME, 'bin', name) : name
const stubs = {
  'android/content/Context.java': `package android.content;
    public abstract class Context {
      public static final int MODE_PRIVATE = 0;
      public Context getApplicationContext() { return this; }
      public abstract SharedPreferences getSharedPreferences(String name, int mode);
    }`,
  'android/content/SharedPreferences.java': `package android.content;
    public interface SharedPreferences {
      java.util.Map<String, ?> getAll();
      String getString(String key, String fallback);
      Editor edit();
      void registerOnSharedPreferenceChangeListener(OnSharedPreferenceChangeListener listener);
      void unregisterOnSharedPreferenceChangeListener(OnSharedPreferenceChangeListener listener);
      interface OnSharedPreferenceChangeListener {
        void onSharedPreferenceChanged(SharedPreferences prefs, String key);
      }
      interface Editor {
        Editor putString(String key, String value);
        Editor remove(String key);
        void apply();
      }
    }`,
  'android/os/Looper.java': `package android.os;
    public final class Looper {
      private static final Looper MAIN = new Looper();
      static final java.util.Queue<Runnable> QUEUE = new java.util.concurrent.ConcurrentLinkedQueue<>();
      public static Looper getMainLooper() { return MAIN; }
      public static void drain() { Runnable next; while ((next = QUEUE.poll()) != null) next.run(); }
    }`,
  'android/os/Handler.java': `package android.os;
    public final class Handler {
      public Handler(Looper looper) {}
      public boolean post(Runnable action) { Looper.QUEUE.add(action); return true; }
    }`,
  'org/json/JSONObject.java': `package org.json;
    public class JSONObject {
      public JSONObject() {}
      public JSONObject(String raw) {}
      public boolean has(String key) { return false; }
      public String optString(String key, String fallback) { return fallback; }
      public boolean optBoolean(String key, boolean fallback) { return fallback; }
      public JSONArray optJSONArray(String key) { return null; }
      public String toString() { return "{}"; }
    }`,
  'org/json/JSONArray.java': `package org.json;
    public class JSONArray {
      public int length() { return 0; }
      public String optString(int index) { return ""; }
    }`,
  'org/json/JSONTokener.java': `package org.json;
    public final class JSONTokener {
      public JSONTokener(String raw) {}
      public Object nextValue() { return new JSONObject(); }
    }`,
  'com/tunnelmanager/app/MainActivity.java': `package com.tunnelmanager.app;
    final class MainActivity {
      static final String PREFS = "test", KEY_SERVER = "server", KEY_TOKEN = "token";
    }`,
  'com/tunnelmanager/app/Theme.java': `package com.tunnelmanager.app;
    final class Theme { static void init(android.content.Context context) {} }`,
}
before(() => {
  const sources = Object.entries(stubs).map(([path, content]) => {
    const target = join(directory, path)
    mkdirSync(dirname(target), { recursive: true })
    writeFileSync(target, content)
    return target
  })
  const result = spawnSync(executable('javac'), ['-encoding', 'UTF-8', '-d', directory,
    ...sources,
    resolve(root, 'app/src/main/java/com/tunnelmanager/app/DNSValues.java'),
    resolve(root, 'app/src/main/java/com/tunnelmanager/app/RequestScope.java'),
    resolve(root, 'app/src/main/java/com/tunnelmanager/app/Session.java'),
    resolve(root, 'app/src/main/java/com/tunnelmanager/app/Api.java'),
    resolve(root, 'tools/java/com/tunnelmanager/app/DNSValuesTest.java'),
    resolve(root, 'tools/java/com/tunnelmanager/app/DNSBoundaryTest.java'),
  ], { encoding: 'utf8' })
  assert.equal(result.status, 0, result.error?.message || result.stderr)
})
after(() => rmSync(directory, { recursive: true, force: true }))
for (const scenario of ['types', 'addresses', 'structured', 'preview', 'retry', 'session-boundary', 'host-boundary', 'inflight-unknown']) {
  test(`native DNS: ${scenario}`, () => {
    const result = spawnSync(executable('java'), ['-cp', directory,
      'com.tunnelmanager.app.DNSValuesTest', scenario,
    ], { encoding: 'utf8' })
    assert.equal(result.status, 0, result.error?.message || result.stderr)
  })
}
for (const scenario of ['api-snapshot', 'api-stale-401', 'api-old-response-401', 'api-unsent',
  'api-inflight-switch', 'api-inflight-destroy', 'api-send-race', 'session-aba', 'session-snapshot-atomic', 'api-compatible']) {
  test(`native DNS boundary: ${scenario}`, () => {
    const result = spawnSync(executable('java'), ['-cp', directory,
      'com.tunnelmanager.app.DNSBoundaryTest', scenario,
    ], { encoding: 'utf8', timeout: 15000 })
    assert.equal(result.status, 0, result.error?.message || result.stderr)
  })
}
