# 域名绑定模式

Web 端的单个与批量域名绑定支持两种模式；批量绑定时每组可独立选择绑定模式、转发地址与优选 CNAME。

## 简化直连

只填写访问域名与转发地址。系统执行两步：

1. 为访问域名添加 Tunnel ingress 路由。
2. 创建一条代理开启的 CNAME 记录，指向 `<tunnel-id>.cfargotunnel.com`。

不需要辅助域名，也不会创建 SaaS Custom Hostname。适合不需要优选加速的常规站点。

## 优选模式

优选模式使用两条相互独立的链路：访问域名负责接入优选线路，辅助域名负责回源到 Tunnel。系统会执行以下配置：

1. 访问域名创建灰云 CNAME，指向本次选择的优选 CNAME。
2. 辅助域名创建橙云 CNAME，指向 `<tunnel-id>.cfargotunnel.com`。
3. 在辅助域名所在 Zone 创建 SaaS Custom Hostname，主机名为访问域名，Custom Origin 为辅助域名。
4. 为访问域名和辅助域名添加 Tunnel ingress 路由。

访问域名与辅助域名不能相同。辅助域名应使用未承载其他服务的独立主机名，避免保存时覆盖已有 DNS 或 ingress 配置。

优选 CNAME 可以从常用 CNAME 组中选择，也可以手动填写。留空时使用全局默认值。

适合需要走自定义优选线路的场景。

| 资源 | 简化直连 | 优选模式 |
| --- | --- | --- |
| 访问域名 | 橙云 CNAME 到 Tunnel | 灰云 CNAME 到优选 CNAME |
| 辅助域名 | 不需要 | 橙云 CNAME 到 Tunnel |
| Custom Hostname | 不创建 | Custom Origin 指向辅助域名 |
| ingress | 访问域名 | 访问域名与辅助域名 |

公开状态页的自定义域名复用同一套优选配置，另有 Host 隔离和手动检查说明，见[服务监控与公开状态页](/guide/monitors-status#自定义域名)。

::: tip 兼容行为
为兼容旧客户端，API 请求未提交 `mode` 字段时仍按优选模式处理。
:::

## 回退源

使用 SaaS Custom Hostname 时，可在面板中一键将回退源（fallback origin）指向指定的服务地址。

## 删除联动

删除隧道路由时可选择连带清理对应的 DNS 记录，避免留下悬空的 CNAME。
