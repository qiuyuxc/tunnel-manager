export const fontSans = '"MiSans", -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif'
export const fontMono = '"SFMono-Regular", Consolas, "Liberation Mono", "Noto Sans Mono CJK SC", monospace'

export const lightPalette = {
  canvas: '#F5F5F5', panel: '#FFFFFF', input: '#F1F1F1', text: '#171717',
  secondary: '#525252', muted: '#666666', accent: '#1A1A1A', onAccent: '#FFFFFF',
  hover: '#333333', line: '#E0E0E0', strongLine: '#A3A3A3',
  success: '#333333', successSurface: '#EBEBEB',
  warning: '#906313', warningSurface: '#F1F1F1', error: '#C13F3F', errorSurface: '#F1F1F1',
}

export type Palette = typeof lightPalette

export const darkPalette: Palette = {
  canvas: '#0F0F0F', panel: '#181818', input: '#232323', text: '#F5F5F5',
  secondary: '#B5B5B5', muted: '#999999', accent: '#EDEDED', onAccent: '#171717',
  hover: '#D1D1D1', line: '#303030', strongLine: '#6B6B6B',
  success: '#D4D4D4', successSurface: '#292929',
  warning: '#E4BA6A', warningSurface: '#252525', error: '#ED9A9A', errorSurface: '#252525',
}

export function themeVariables(palette: Palette): Record<string, string> {
  return {
    'color-canvas': palette.panel,
    'color-canvas-soft': palette.canvas,
    'color-canvas-soft-2': palette.input,
    'color-canvas-raised': palette.panel,
    'color-ink': palette.text,
    'color-body': palette.secondary,
    'color-mute': palette.muted,
    'color-hairline': palette.line,
    'color-hairline-strong': palette.strongLine,
    'color-link': palette.accent,
    'color-link-hover': palette.hover,
    'color-focus': palette.accent,
    'color-success': palette.success,
    'color-error': palette.error,
    'color-warning': palette.warning,
    'color-info': palette.accent,
    'color-sidebar': palette.canvas,
    'color-sidebar-hover': palette.panel,
    'color-sidebar-active': palette.accent,
    'color-sidebar-active-bg': palette.successSurface,
    'color-sidebar-text': palette.secondary,
    'color-sidebar-text-active': palette.accent,
    'color-sidebar-hover-text': palette.text,
    'color-sidebar-divider': palette.line,
    'color-brand-icon': palette.accent,
    'color-header-bg': palette.canvas,
    'color-header-border': palette.line,
    'color-status-healthy-bg': palette.successSurface,
    'color-status-healthy-border': palette.successSurface,
    'color-status-healthy-text': palette.success,
    'color-status-down-bg': palette.errorSurface,
    'color-status-down-border': palette.errorSurface,
    'color-status-down-text': palette.error,
    'color-status-degraded-bg': palette.warningSurface,
    'color-status-degraded-border': palette.warningSurface,
    'color-status-degraded-text': palette.warning,
    'color-btn-primary-bg': palette.accent,
    'color-btn-primary-text': palette.onAccent,
    'color-btn-primary-hover': palette.hover,
    'color-btn-secondary-border': palette.line,
    'color-btn-secondary-hover-border': palette.strongLine,
    'color-btn-ghost-hover': palette.input,
    'color-banner-warning-bg': palette.warningSurface,
    'color-banner-warning-border': palette.warningSurface,
    'color-banner-warning-text': palette.warning,
    'color-banner-info-bg': palette.successSurface,
    'color-banner-info-border': palette.successSurface,
    'color-banner-info-text': palette.accent,
    'color-result-error-bg': palette.errorSurface,
    'color-result-error-border': palette.errorSurface,
    'color-result-error-text': palette.error,
    'color-result-success-bg': palette.successSurface,
    'color-result-success-border': palette.successSurface,
    'color-result-success-text': palette.success,
  }
}

export function installDesignTokens() {
  const themes: [string, Palette][] = [
    [':root', lightPalette],
    ['[data-theme="dark"]', darkPalette],
  ]
  const sheet = document.createElement('style')
  sheet.id = 'tunnel-design-tokens'
  sheet.textContent = themes.map(([selector, palette]) => `${selector}{${Object.entries(themeVariables(palette)).map(([key, value]) => `--${key}:${value}`).join(';')}}`).join('\n')
  document.head.appendChild(sheet)
}
