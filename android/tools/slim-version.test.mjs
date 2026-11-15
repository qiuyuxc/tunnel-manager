import assert from 'node:assert/strict'
import { readFileSync, mkdtempSync, writeFileSync, rmSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { tmpdir } from 'node:os'
import { spawnSync } from 'node:child_process'
import test from 'node:test'

const root = resolve(import.meta.dirname, '..')
const gradle = readFileSync(join(root, 'app/build.gradle.kts'), 'utf8')
const slim = /versionName\s*=\s*"[^"\n]+-slim"/.test(gradle)
const executable = name => process.env.JAVA_HOME ? join(process.env.JAVA_HOME, 'bin', name) : name

function extractMethod(source, signature) {
  const start = source.indexOf(signature)
  assert.ok(start >= 0, `Missing ${signature}`)
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
  throw new Error(`Unclosed ${signature}`)
}

test('native slim update versions parse and compare actual release tags', { skip: !slim }, () => {
  const source = readFileSync(join(root, 'app/src/main/java/com/tunnelmanager/app/AboutFragment.java'), 'utf8')
  const directory = mkdtempSync(join(tmpdir(), 'native-slim-version-'))
  try {
    const code = `import java.util.regex.Matcher; import java.util.regex.Pattern;
      public final class SlimVersionTest {
        private static final String SLIM_SUFFIX = "-slim";
        ${extractMethod(source, '    private static int[] parse(String tag) {')}
        ${extractMethod(source, '    private static int compare(String a, String b) {')}
        ${extractMethod(source, '    private static boolean isSlimRelease(String tag, boolean draft) {')}
        public static void main(String[] args) {
          if (!isSlimRelease("v3.1.0-slim", false) || isSlimRelease("v3.1.0-slim", true)
              || isSlimRelease("v3.1.0", false) || isSlimRelease("v3.1.0-slim-other", false)
              || isSlimRelease(null, false))
            throw new AssertionError("Published slim tags or draft eligibility is incorrect");
          int[] parsed = parse("v3.1.0-slim");
          if (parsed == null || parsed[0] != 3 || parsed[1] != 1 || parsed[2] != 0)
            throw new AssertionError("The published slim release cannot be parsed");
          if (parse("3.1.0-slim") == null || parse("v3.1.0") == null)
            throw new AssertionError("Installed app or server version cannot be parsed");
          if (compare("v3.2.0-slim", "v3.1.99-slim") <= 0 || compare("v3.1.0-slim", "3.1.0-slim") != 0)
            throw new AssertionError("Slim release sorting or version equality is incorrect");
          if (parse("v3.1.0-slim-other") != null || parse("xv3.1.0-slim") != null || parse(null) != null)
            throw new AssertionError("Malformed or missing release tag was accepted");
        }
      }`
    const file = join(directory, 'SlimVersionTest.java')
    writeFileSync(file, code)
    const compiled = spawnSync(executable('javac'), ['-encoding', 'UTF-8', '-d', directory, file], { encoding: 'utf8', timeout: 15000 })
    assert.equal(compiled.status, 0, compiled.error?.message || compiled.stderr)
    const result = spawnSync(executable('java'), ['-cp', directory, 'SlimVersionTest'], { encoding: 'utf8', timeout: 15000 })
    assert.equal(result.status, 0, result.error?.message || result.stderr)
  } finally {
    rmSync(directory, { recursive: true, force: true })
  }
})
