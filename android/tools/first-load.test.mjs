import { after, before, test } from 'node:test'
import assert from 'node:assert/strict'
import { existsSync, readFileSync, mkdtempSync, mkdirSync, writeFileSync, rmSync } from 'node:fs'
import { join, resolve, dirname } from 'node:path'
import { tmpdir } from 'node:os'
import { spawnSync } from 'node:child_process'
import { writeI18nStub } from './i18n-stub.mjs'

const root = resolve(import.meta.dirname, '..')
const directory = mkdtempSync(join(tmpdir(), 'native-first-load-'))
const executable = name => process.env.JAVA_HOME ? join(process.env.JAVA_HOME, 'bin', name) : name
const names = ['AccountFragment', 'SettingsFragment', 'AdminFragment', 'TelegramFragment']
  .filter(name => existsSync(resolve(root, `app/src/main/java/com/tunnelmanager/app/${name}.java`)))
const slim = /versionName\s*=\s*"[^"\n]+-slim"/.test(readFileSync(resolve(root, 'app/build.gradle.kts'), 'utf8'))

// Compile the actual field initialization and complete load method verbatim.
// Only the Android rendering host is replaced; the real Api/Session perform
// HTTP dispatch against an in-memory URL handler with controlled completion.
function method(source, signature = '    private void load() {') {
  const start = source.indexOf(signature)
  assert.ok(start >= 0)
  let depth = 0, quoted = false, escaped = false
  for (let index = source.indexOf('{', start); index < source.length; index++) {
    const character = source[index]
    if (quoted) {
      if (escaped) escaped = false
      else if (character === '\\') escaped = true
      else if (character === '"') quoted = false
    } else if (character === '"') quoted = true
    else if (character === '{') depth++
    else if (character === '}' && --depth === 0) return source.slice(start, index + 1)
  }
  throw new Error('Unclosed load method')
}
const stubs = {
  'android/content/Context.java': `package android.content; public abstract class Context {
    public static final int MODE_PRIVATE=0; public Context getApplicationContext(){return this;}
    public abstract SharedPreferences getSharedPreferences(String key,int mode); }`,
  'android/content/SharedPreferences.java': `package android.content; public interface SharedPreferences {
    java.util.Map<String,?> getAll(); String getString(String key,String fallback); Editor edit();
    void registerOnSharedPreferenceChangeListener(OnSharedPreferenceChangeListener listener);
    interface OnSharedPreferenceChangeListener { void onSharedPreferenceChanged(SharedPreferences prefs,String key); }
    interface Editor { Editor putString(String key,String value); Editor remove(String key); void apply(); } }`,
  'android/os/Looper.java': `package android.os; public final class Looper {
    static final java.util.Queue<Runnable> queue=new java.util.concurrent.ConcurrentLinkedQueue<>();
    public static Looper getMainLooper(){return new Looper();}
    public static void drain(){Runnable action;while((action=queue.poll())!=null)action.run();} }`,
  'android/os/Handler.java': `package android.os; public final class Handler {
    public Handler(Looper looper){} public boolean post(Runnable action){Looper.queue.add(action);return true;}
    public boolean postDelayed(Runnable action,long delay){return true;} }`,
  'android/os/Bundle.java': `package android.os; public final class Bundle {
    private final java.util.Map<String,Object> values=new java.util.HashMap<>();
    public boolean containsKey(String key){return values.containsKey(key);}
    public String getString(String key){return getString(key,null);}
    public String getString(String key,String fallback){Object value=values.get(key);return value instanceof String?(String)value:fallback;}
    public boolean getBoolean(String key,boolean fallback){Object value=values.get(key);return value instanceof Boolean?(Boolean)value:fallback;}
    public void putString(String key,String value){values.put(key,value);} public void putBoolean(String key,boolean value){values.put(key,value);}
    public void putStringArrayList(String key,java.util.ArrayList<String> value){values.put(key,value);}
    @SuppressWarnings("unchecked") public java.util.ArrayList<String> getStringArrayList(String key){return (java.util.ArrayList<String>)values.get(key);}
  }`,
  'org/json/JSONObject.java': `package org.json; public class JSONObject {
    private final java.util.Map<String,Object> values=new java.util.HashMap<>();
    public JSONObject(){} public JSONObject(String raw){if(raw!=null&&raw.contains("site_name"))values.put("site_name","Fixture");} public JSONObject put(String key,Object value){values.put(key,value);return this;}
    public boolean has(String key){return values.containsKey(key);} public String optString(String key){return optString(key,"");}
    public String optString(String key,String fallback){Object value=values.get(key);return value instanceof String?(String)value:fallback;}
    public boolean optBoolean(String key,boolean fallback){return fallback;}
    public JSONArray optJSONArray(String key){return null;}
    public JSONObject optJSONObject(String key){Object value=values.get(key);return value instanceof JSONObject?(JSONObject)value:null;}
    public String toString(){return values.containsKey("site_name")?"{site_name:Fixture}":"{}";} }`,
  'org/json/JSONArray.java': `package org.json; public class JSONArray { public int length(){return 0;} public String optString(int index){return "";} }`,
  'org/json/JSONTokener.java': `package org.json; public class JSONTokener { private final String raw; public JSONTokener(String raw){this.raw=raw;} public Object nextValue(){return new JSONObject(raw);} }`,
  'com/tunnelmanager/app/MainActivity.java': `package com.tunnelmanager.app; final class MainActivity { static final String PREFS="fixture",KEY_SERVER="server",KEY_TOKEN="token"; }`,
  'com/tunnelmanager/app/Theme.java': `package com.tunnelmanager.app; final class Theme { static void init(android.content.Context context){} }`,
}
before(() => {
  const sources = Object.entries(stubs).map(([path, content]) => {
    const target = join(directory, path); mkdirSync(dirname(target), { recursive: true }); writeFileSync(target, content); return target
  })
  for (const name of names) {
    const source = readFileSync(resolve(root, `app/src/main/java/com/tunnelmanager/app/${name}.java`), 'utf8')
    const field = source.match(/private boolean loading(?:\s*=\s*(?:true|false))?;/)?.[0]
    assert.ok(field)
    const busy = source.match(/private boolean busy(?:\s*=\s*(?:true|false))?;/)?.[0]
    assert.ok(busy)
    const passkeyBusy = source.match(/private boolean pkBusy(?:\s*=\s*(?:true|false))?;/)?.[0] || ''
    const canRefresh = method(source, '    @Override boolean canRefresh() {')
    const beginWrite = method(source, '    private boolean beginWrite() {')
    const beginPasskeyWrite = name === 'AccountFragment' ? method(source, '    private boolean beginPasskeyWrite() {') : ''
    const build = method(source, source.match(/    (?:@Override )?protected View build[^\n]+/)?.[0])
    const tail = build.slice(build.lastIndexOf('        render();'), build.lastIndexOf('        return '))
    assert.ok(tail.includes('load();'), `${name}: build entry point must include its load decision`)
    const loaded = source.match(/private boolean configLoaded(?:\s*=\s*(?:true|false))?;/)?.[0] || ''
    const restore = name === 'SettingsFragment' ? method(source, '    @Override public void onCreate(Bundle saved) {') : ''
    const save = name === 'SettingsFragment' ? method(source, '    @Override public void onSaveInstanceState(@NonNull Bundle saved) {').replace('@NonNull ', '') : ''
    const apply = name === 'SettingsFragment' ? 'void applyConfig(JSONObject payload) { ' + method(source, '    private void applyConfig(JSONObject payload) {').match(/configLoaded\s*=\s*(?:true|false);/)[0] + ' config=payload; }' : ''
    const target = join(directory, `com/tunnelmanager/app/${name}Load.java`)
    writeFileSync(target, `package com.tunnelmanager.app; import org.json.*; import android.os.Bundle;
      final class ${name}Load extends FirstLoadTest.Host { ${field} ${busy} ${passkeyBusy} ${loaded}
        ${method(source)} ${restore} ${save} ${apply} ${canRefresh} ${beginWrite} ${beginPasskeyWrite}
        void enter(){${tail}} void refresh(){load();} boolean pending(){return loading;}
        boolean eligible(){return canRefresh();}
        boolean gateWrite(boolean passkey){
          boolean started = ${name === 'AccountFragment' ? 'passkey ? beginPasskeyWrite() : beginWrite()' : 'beginWrite()'};
          if (!started) return false;
          Api.async(() -> Api.put("/api/fixture/write", new JSONObject()),
            result -> { busy=false; ${name === 'AccountFragment' ? 'pkBusy=false;' : ''} },
            failure -> { busy=false; ${name === 'AccountFragment' ? 'pkBusy=false;' : ''} });
          return true;
        }
        boolean writing(){return ${name === 'AccountFragment' ? 'busy || pkBusy' : 'busy'};}

      }`)
    sources.push(target)
  }
  const result = spawnSync(executable('javac'), ['-encoding', 'UTF-8', '-d', directory,
    writeI18nStub(directory), ...sources,
    ...['Api.java', 'Session.java', 'RequestScope.java', 'WebOrigin.java', 'NativeRequestPolicy.java'].map(file => resolve(root, 'app/src/main/java/com/tunnelmanager/app', file)),
    resolve(root, 'tools/java/com/tunnelmanager/app/FirstLoadTest.java')], { encoding: 'utf8' })
  assert.equal(result.status, 0, result.error?.message || result.stderr)
})
after(() => rmSync(directory, { recursive: true, force: true }))
for (const name of names) {
  test(`${name}: first entry, refresh merging, write/passkey exclusion and recovery use real HTTP`, () => {
    const expected = name === 'SettingsFragment' ? (slim ? 5 : 1) : name === 'AccountFragment' ? 4 : name === 'AdminFragment' ? 7 : 2
    const result = spawnSync(executable('java'), ['-cp', directory, 'com.tunnelmanager.app.FirstLoadTest', name, String(expected)], { encoding: 'utf8', timeout: 15000 })
    assert.equal(result.status, 0, result.error?.message || result.stderr)
  })
}
