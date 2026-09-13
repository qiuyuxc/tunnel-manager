/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

// Cloudflare Turnstile widget (loaded explicitly from challenges.cloudflare.com).
interface TurnstileWidget {
  render(container: HTMLElement, options: Record<string, unknown>): string
  reset(widgetId?: string): void
  remove(widgetId?: string): void
}

interface Window {
  turnstile?: TurnstileWidget
  TMHost?: TunnelManagerHost
}

// Native bridge injected by the Android shell, see
// android/app/java/com/tunnelmanager/app/MainActivity.java. Only the console
// features that genuinely need the OS live here; everything else stays web.
interface TunnelManagerHost {
  /** JSON string of { enabled, permission, battery }. */
  notifyState(): string
  notifyEnable(): void
  notifyDisable(): void
  notifyTest(): void
  notifyBattery(): void
}
