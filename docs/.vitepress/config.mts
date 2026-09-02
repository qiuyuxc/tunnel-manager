import { defineConfig } from 'vitepress'

const GITHUB = 'https://github.com/qiuyuxc/tunnel-manager'

export default defineConfig({
  title: 'Tunnel Manager',
  // 默认根路径（本地预览、未来绑定自定义域名）；GitHub Pages 项目子路径由 CI 注入 DOCS_BASE 覆盖
  base: process.env.DOCS_BASE ?? '/',
  cleanUrls: true,
  lastUpdated: true,
  ignoreDeadLinks: false,

  themeConfig: {
    socialLinks: [{ icon: 'github', link: GITHUB }],
    search: {
      provider: 'local',
      options: {
        locales: {
          root: {
            translations: {
              button: { buttonText: '搜索文档', buttonAriaLabel: '搜索文档' },
              modal: {
                noResultsText: '未找到相关结果',
                resetButtonTitle: '清除查询条件',
                displayDetails: '显示详细列表',
                footer: { selectText: '选择', navigateText: '切换', closeText: '关闭' }
              }
            }
          }
        }
      }
    }
  },

  locales: {
    root: {
      label: '简体中文',
      lang: 'zh-CN',
      description: 'Cloudflare Tunnel 可视化管理面板：隧道管理、域名绑定、DNS、服务监控与公开状态页',
      themeConfig: {
        nav: [
          { text: '指南', link: '/guide/getting-started', activeMatch: '/guide/' },
          { text: 'FAQ', link: '/faq/', activeMatch: '/faq/' },
          { text: 'API', link: '/api/endpoints', activeMatch: '/api/' },
          { text: 'GitHub', link: GITHUB }
        ],
        sidebar: {
          '/guide/': [
            {
              text: '部署与升级',
              items: [
                { text: '快速开始', link: '/guide/getting-started' },
                { text: 'Docker Compose 部署详解', link: '/guide/docker-compose' },
                { text: '二进制部署', link: '/guide/binary-deploy' },
                { text: '升级、备份与恢复', link: '/guide/upgrade-backup' }
              ]
            },
            {
              text: '账号与安全',
              items: [
                { text: '多用户与管理后台', link: '/guide/multi-user' },
                { text: '邮件服务与监控告警', link: '/guide/email-alerts' },
                { text: 'Cloudflare OAuth 连接', link: '/guide/cloudflare-oauth' },
                { text: '管理员认证与双因素', link: '/guide/security' }
              ]
            },
            {
              text: '功能使用',
              items: [
                { text: '隧道管理与路由', link: '/guide/tunnels' },
                { text: '域名绑定模式', link: '/guide/domain-binding' },
                { text: '批量域名绑定', link: '/guide/batch-binding' },
                { text: 'DNS 记录管理', link: '/guide/dns-management' },
                { text: '站点品牌与全局设置', link: '/guide/site-branding' },
                { text: '服务监控与公开状态页', link: '/guide/monitors-status' },
                { text: 'Telegram Bot 远程管理', link: '/guide/telegram-bot' }
              ]
            }
          ],
          '/api/': [
            {
              text: 'API 参考',
              items: [
                { text: '接口列表', link: '/api/endpoints' },
                { text: '调用示例', link: '/api/examples' }
              ]
            }
          ],
          '/reference/': [
            {
              text: '参考',
              items: [
                { text: '架构与技术栈', link: '/reference/architecture' },
                { text: '版本历史', link: '/changelog' }
              ]
            }
          ],
          '/faq/': [
            { text: '支持', items: [{ text: '常见问题 FAQ', link: '/faq/index' }] }
          ]
        },
        outline: { level: [2, 3], label: '本页目录' },
        docFooter: { prev: '上一页', next: '下一页' },
        lastUpdated: { text: '最后更新于' },
        returnToTopLabel: '回到顶部',
        sidebarMenuLabel: '菜单',
        darkModeSwitchLabel: '外观',
        langMenuLabel: '切换语言',
        footer: { message: 'Cloudflare Tunnel 可视化管理面板', copyright: 'Tunnel Manager' }
      }
    },
    en: {
      label: 'English',
      lang: 'en-US',
      link: '/en/',
      description: 'A web control panel for Cloudflare Tunnel: tunnels, domain binding, DNS, service monitoring and public status pages',
      themeConfig: {
        nav: [
          { text: 'Guide', link: '/en/guide/getting-started', activeMatch: '/en/guide/' },
          { text: 'FAQ', link: '/en/faq/', activeMatch: '/en/faq/' },
          { text: 'API', link: '/en/api/endpoints', activeMatch: '/en/api/' },
          { text: 'GitHub', link: GITHUB }
        ],
        sidebar: {
          '/en/guide/': [
            {
              text: 'Deploy & upgrade',
              items: [
                { text: 'Quick start', link: '/en/guide/getting-started' },
                { text: 'Docker Compose deployment', link: '/en/guide/docker-compose' },
                { text: 'Binary deployment', link: '/en/guide/binary-deploy' },
                { text: 'Upgrade, backup & restore', link: '/en/guide/upgrade-backup' }
              ]
            },
            {
              text: 'Accounts & security',
              items: [
                { text: 'Multi-user & admin console', link: '/en/guide/multi-user' },
                { text: 'Email service & alerts', link: '/en/guide/email-alerts' },
                { text: 'Cloudflare OAuth connection', link: '/en/guide/cloudflare-oauth' },
                { text: 'Admin auth & two-factor', link: '/en/guide/security' }
              ]
            },
            {
              text: 'Using the panel',
              items: [
                { text: 'Tunnels & routing', link: '/en/guide/tunnels' },
                { text: 'Domain binding modes', link: '/en/guide/domain-binding' },
                { text: 'Batch domain binding', link: '/en/guide/batch-binding' },
                { text: 'DNS records', link: '/en/guide/dns-management' },
                { text: 'Site branding & global settings', link: '/en/guide/site-branding' },
                { text: 'Monitoring & status pages', link: '/en/guide/monitors-status' },
                { text: 'Telegram bot', link: '/en/guide/telegram-bot' }
              ]
            }
          ],
          '/en/api/': [
            {
              text: 'API reference',
              items: [
                { text: 'Endpoints', link: '/en/api/endpoints' },
                { text: 'Examples', link: '/en/api/examples' }
              ]
            }
          ],
          '/en/reference/': [
            {
              text: 'Reference',
              items: [
                { text: 'Architecture & stack', link: '/en/reference/architecture' },
                { text: 'Changelog', link: '/en/changelog' }
              ]
            }
          ],
          '/en/faq/': [
            { text: 'Support', items: [{ text: 'FAQ', link: '/en/faq/index' }] }
          ]
        },
        outline: { level: [2, 3], label: 'On this page' },
        footer: { message: 'A web control panel for Cloudflare Tunnel', copyright: 'Tunnel Manager' }
      }
    }
  }
})
