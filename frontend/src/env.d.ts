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
}
