---
layout: home

hero:
  name: Tunnel Manager
  text: Cloudflare Tunnel 可视化管理面板
  tagline: 通过 Web UI 管理隧道、绑定域名、配置 DNS 优选与回退源，提供服务可用性监控与可分享的公开状态页，并支持 Telegram Bot 远程管理和管理员双重身份验证。
  actions:
    - theme: brand
      text: 快速开始
      link: /guide/getting-started
    - theme: alt
      text: API 参考
      link: /api/endpoints
    - theme: alt
      text: GitHub 仓库
      link: https://github.com/qiuyuxc/tunnel-manager

features:
  - title: 隧道管理
    details: 新建、删除与列出 Cloudflare Tunnel，查看和编辑应用程序路由；新建隧道后直接展示 cloudflared 连接令牌与运行命令。
    link: /guide/tunnels
    linkText: 管理隧道与路由
  - title: 域名绑定双模式
    details: 简化直连只配置主域名 Tunnel 路由和代理 CNAME；优选模式走优选 CNAME 与 SaaS Custom Hostname 完整流程，支持批量绑定。
    link: /guide/domain-binding
    linkText: 查看绑定模式
  - title: DNS 管理
    details: 按 Zone 查询和管理 A、AAAA、CNAME、TXT、MX 记录，支持 TTL、代理状态、MX 优先级与多选批量操作。
    link: /guide/dns-management
    linkText: 进入 DNS 管理
  - title: 服务监控
    details: HTTP（GET/POST）、TCP、ICMP 三种探测方式，每个监控挂多个目标，仪表盘展示近 24 小时延迟柱图。
    link: /guide/monitors-status
    linkText: 配置监控
  - title: 公开状态页
    details: 免登录分享检测结果，系统令牌或自定义短路径任一有效；标题、公告、品牌图标与主题均可自定义。
    link: /guide/monitors-status
    linkText: 发布状态页
  - title: 安全与远程管理
    details: Argon2id 密码哈希、标准 TOTP 双因素验证与一次性恢复码；Telegram Bot 远程执行隧道、绑定与 DNS 操作。
    link: /guide/security
    linkText: 安全配置
---
