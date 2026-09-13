/**
 * Thin wrapper over the Android shell's TMHost bridge.
 *
 * Everything returns null/does nothing when the console is running in a plain
 * browser, so callers only need one check to decide whether to render the
 * native-only controls.
 */

export interface HostNotifyState {
  /** Background alert polling is switched on. */
  enabled: boolean
  /** The OS granted POST_NOTIFICATIONS. */
  permission: boolean
  /** The app is exempt from battery optimisation. */
  battery: boolean
}

function bridge() {
  return typeof window === 'undefined' ? undefined : window.TMHost
}

export function isAppShell(): boolean {
  return !!bridge()
}

export function readHostNotifyState(): HostNotifyState | null {
  const host = bridge()
  if (!host) return null
  try {
    const raw = JSON.parse(host.notifyState())
    return {
      enabled: !!raw.enabled,
      permission: !!raw.permission,
      battery: !!raw.battery,
    }
  } catch {
    return null
  }
}

export function enableHostNotify() {
  bridge()?.notifyEnable()
}

export function disableHostNotify() {
  bridge()?.notifyDisable()
}

export function sendHostTestNotify() {
  bridge()?.notifyTest()
}

export function openBatterySettings() {
  bridge()?.notifyBattery()
}
