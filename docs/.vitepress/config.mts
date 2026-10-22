import { defineConfig } from 'vitepress'
import labels from './labels.json'
import { sections } from './navigation'

const GITHUB = 'https://github.com/qiuyuxc/tunnel-manager'
function settings(language: 'zh-CN' | 'en') {
  const messages: Record<string, string> = labels[language]
  const prefix = language === 'en' ? '/en' : ''
  const link = (path: string) => `${prefix}/${path}`
  const sidebar = Object.fromEntries(sections.map(section => [link(section.path) + '/', section.groups.map(group => ({
    text: messages[`groups.${group.id}`],
    items: group.pages.map(page => {
      const path = page.startsWith('/') ? page.slice(1) : `${section.path}/${page}`
      return { text: messages[path], link: link(path) }
    }),
  }))]))
  return {
    label: language === 'en' ? 'English' : '简体中文',
    lang: language === 'en' ? 'en-US' : 'zh-CN',
    ...(language === 'en' ? { link: '/en/' } : {}),
    description: messages.description,
    themeConfig: {
      nav: ['guide/getting-started', 'faq', 'api/endpoints'].map(path => ({ text: messages[path === 'guide/getting-started' ? 'nav.guide' : path === 'api/endpoints' ? 'nav.api' : path], link: link(path), activeMatch: link(path.split('/')[0]) + '/' })).concat([{ text: 'GitHub', link: GITHUB, activeMatch: '' }]),
      sidebar,
      outline: { level: [2, 3] as [number, number], label: messages.outline },
      docFooter: { prev: messages.previous, next: messages.next },
      lastUpdated: { text: messages.updated },
      returnToTopLabel: messages.top,
      sidebarMenuLabel: messages.menu,
      darkModeSwitchLabel: messages.appearance,
      langMenuLabel: messages.language,
      footer: { message: messages.footer, copyright: 'Tunnel Manager' },
    },
  }
}
function search(language: 'zh-CN' | 'en') {
  const messages = labels[language]
  return { button: { buttonText: messages.search, buttonAriaLabel: messages.search }, modal: {
    noResultsText: messages.noResults, resetButtonTitle: messages.reset, displayDetails: messages.details,
    footer: { selectText: messages.select, navigateText: messages.navigate, closeText: messages.close },
  } }
}
export default defineConfig({
  title: 'Tunnel Manager',
  base: process.env.DOCS_BASE ?? '/',
  cleanUrls: true,
  lastUpdated: true,
  ignoreDeadLinks: false,
  themeConfig: {
    socialLinks: [{ icon: 'github', link: GITHUB }],
    search: { provider: 'local', options: { locales: { root: { translations: search('zh-CN') }, en: { translations: search('en') } } } },
  },
  locales: { root: settings('zh-CN'), en: settings('en') },
})
