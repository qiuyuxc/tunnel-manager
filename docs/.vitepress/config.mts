import { defineConfig } from 'vitepress'

export default defineConfig({
  lang: 'zh-CN',
  title: 'Tunnel Manager',
  description: 'Cloudflare Tunnel 可视化管理面板：隧道管理、域名绑定、DNS、服务监控与公开状态页',
  // 默认根路径（本地预览、未来绑定自定义域名）；GitHub Pages 项目子路径由 CI 注入 DOCS_BASE 覆盖
  base: process.env.DOCS_BASE ?? '/',
  cleanUrls: true,
  lastUpdated: true,
  ignoreDeadLinks: false,
  themeConfig: {
    nav: [
      { text: '指南', link: '/guide/getting-started', activeMatch: '/guide/' },
      { text: 'FAQ', link: '/faq/', activeMatch: '/faq/' },
      { text: 'API', link: '/api/endpoints', activeMatch: '/api/' },
      { text: 'GitHub', link: 'https://github.com/qiuyuxc/tunnel-manager' }
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
    socialLinks: [{ icon: 'github', link: 'https://github.com/qiuyuxc/tunnel-manager' }],
    outline: { level: [2, 3], label: '本页目录' },
    docFooter: { prev: '上一页', next: '下一页' },
    lastUpdated: { text: '最后更新于' },
    returnToTopLabel: '回到顶部',
    sidebarMenuLabel: '菜单',
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
    },
    footer: { message: 'Cloudflare Tunnel 可视化管理面板', copyright: 'Tunnel Manager' }
  }
})
