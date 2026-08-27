# 域名绑定模式

Web 端的单个与批量域名绑定支持两种模式；批量绑定时每组可独立选择绑定模式、转发地址与优选 CNAME。

## 简化直连

只填写主域名与转发地址。系统执行两步：

1. 为主域名添加 Tunnel Ingress 路由；
2. 创建一条代理开启的 CNAME 记录，指向 `<tunnel-id>.cfargotunnel.com`。

不需要辅助域名，也不会创建 SaaS Custom Hostname。适合不需要优选加速的常规站点。

## 优选模式

沿用完整的 Cloudflare for SaaS 流程：配置优选 CNAME、辅助域名与 Custom Hostname。优选 CNAME 留空时使用全局默认值（可在设置中维护常用 CNAME 组供直接选择）。

适合需要走自定义优选线路的场景。

::: tip 兼容行为
为兼容旧客户端，API 请求未提交 `mode` 字段时仍按优选模式处理。
:::

## 回退源

使用 SaaS Custom Hostname 时，可在面板中一键将回退源（fallback origin）指向指定的服务地址。

## 删除联动

删除隧道路由时可选择连带清理对应的 DNS 记录，避免留下悬空的 CNAME。
