export const fontSans = '"MiSans", -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif'
export const fontMono = '"SFMono-Regular", Consolas, "Liberation Mono", "Noto Sans Mono CJK SC", monospace'

export const lightPalette = {
  canvas: '#EDF4F0', panel: '#FAFDFB', input: '#DCE8E2', text: '#1B2A27',
  secondary: '#4D5C58', muted: '#586660', accent: '#127250', onAccent: '#F6F9F7',
  hover: '#0E6244', line: '#D0DED6', strongLine: '#91A89A',
  success: '#127250', successSurface: '#DCEEE3',
  warning: '#8B5A10', warningSurface: '#F8EBD3', error: '#AE3842', errorSurface: '#F8E4E6',
}

export type Palette = typeof lightPalette

export const darkPalette: Palette = {
  canvas: '#0B1211', panel: '#151D1C', input: '#1E2625', text: '#E7ECEA',
  secondary: '#96A19F', muted: '#83908B', accent: '#6EDDAD', onAccent: '#042317',
  hover: '#8AE8BE', line: '#283330', strongLine: '#566860',
  success: '#6EDDAD', successSurface: '#1B3329',
  warning: '#E9BD73', warningSurface: '#332B1E', error: '#F09898', errorSurface: '#382326',
}

export const warmPalette: Palette = {
  canvas: '#F5F0E8', panel: '#FAF9F5', input: '#EFE9DE', text: '#141413',
  secondary: '#3D3D3A', muted: '#5F5D57', accent: '#A9583E', onAccent: '#FFFFFF',
  hover: '#8F4A34', line: '#E6DFD8', strongLine: '#C9C0B2',
  success: '#287847', successSurface: '#E3EFE3',
  warning: '#896019', warningSurface: '#F5EACF', error: '#AC3535', errorSurface: '#F7E2E0',
}

export const warmDarkPalette: Palette = {
  canvas: '#181715', panel: '#1F1E1B', input: '#252320', text: '#FAF9F5',
  secondary: '#B7B3AB', muted: '#A8A49C', accent: '#E08B67', onAccent: '#2C130A',
  hover: '#EBA98B', line: '#33312C', strongLine: '#69645B',
  success: '#A9D4B4', successSurface: '#25352A',
  warning: '#E8BC78', warningSurface: '#382D1D', error: '#EAA0A0', errorSurface: '#3B2525',
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
    ['[data-visual-theme="warm"]', warmPalette],
    ['[data-visual-theme="warm"][data-theme="dark"]', warmDarkPalette],
  ]
  const sheet = document.createElement('style')
  sheet.id = 'tunnel-design-tokens'
  sheet.textContent = themes.map(([selector, palette]) => `${selector}{${Object.entries(themeVariables(palette)).map(([key, value]) => `--${key}:${value}`).join(';')}}`).join('\n')
  document.head.appendChild(sheet)
}
