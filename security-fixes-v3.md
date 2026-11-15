# Tunnel Manager 3.0 安全修复记录

日期：2026-10-23。对应 `security-review-v3.md` 的 11 项确认问题，基线 `1b0951d`。修复已写入当前工作区，未提交、未推送、未部署。原审查笔记与隔离复现保留；原复现的 PASS 仍表示漏洞存在，正常回归使用业务包内的测试。

| 审查项 | 修复结果 |
| --- | --- |
| 1. Android 跨源会话注入 | 两处面板 WebView 的 Java 与注入脚本均检查协议、主机、端口；异步回调再次核对页面。外部页面使用独立 WebView，保留一次性认证 cookie，但无会话注入、无原生桥。MainActivity 改为读取顶层页面路径，移除向所有 frame 暴露的 TMHost。禁用混合内容、文件及 content 访问。 |
| 2. 邮箱与 OIDC 预先接管 | 邮箱更新需当前密码与新邮箱验证码，验证前原邮箱不变；验证码绑定账户和认证版本。Web 与 Android 增加发送、输入验证码流程。OIDC 不再按同邮箱自动合并，需登录原账户后确认密码主动绑定；免邮件验证注册也不能触发合并。已有稳定身份继续按 Provider / Issuer / Subject 登录。 |
| 3. Telegram 越权 | 每条命令重新读取账户状态与用户组，中文命令及别名统一映射权限；DNS 删除执行前再次校验。管理端禁用、删除账户后立即 Reconcile 停止 Bot。 |
| 4. 监控内网请求 | 普通账户仅允许公网 HTTP / TCP / ICMP，解析所有 DNS 结果后检查，拨号与 ping 使用已检查的数字 IP；HTTP 每次重定向受同一拨号检查，不使用环境代理。管理员所属监控允许私网和回环，仍拒绝链路本地、常见元数据地址及特殊网段。定时探测核对账户状态与当前权限。全局探测并发 32、手动检测每账户每分钟 5 次。 |
| 5. 重置后旧挑战可用 | 新增持久化 auth_version，SQLite / PostgreSQL migration 16。密码更新、管理员及 CLI 重置、账户禁用原子更新版本并撤销会话；挑战创建、领取和最终签发会话检查版本，覆盖已领取挑战与重置并发。保存失败回滚版本、密码和会话。 |
| 6. 匿名 Passkey 状态耗尽 | Begin 在分配状态前原子计入独立请求预算；单来源、单账户最多 8 个未完成 ceremony，过期释放。Begin 用完预算后允许当前 Finish 完成。高配置额度使用固定大小计数器。 |
| 7. 伪造来源 IP | 默认只用 TCP peer；新增 TRUSTED_PROXY_CIDRS，核对直接代理后从 XFF 右侧解析到首个不可信地址。IP 规范化供限流、熟悉来源、审计及 Turnstile 使用。 |
| 8. Passkey 二次验证锁死 | Begin 返回时释放挑战锁；ceremony 绑定精确挑战和用途。有效签名可完成，取消可回退 TOTP；其他挑战及匿名 Finish 无法复用。 |
| 9. 公开页错误泄露 | 匿名接口对新旧详细错误统一返回固定“探测失败”，登录后的监控保留详细诊断。 |
| 10. OAuth 账户为空 | Status 自行获取当前用户所属 Cloudflare 客户端；经实际 SessionOnly middleware 的回归验证返回账户列表。 |
| 11. 自定义状态域名字体 404 | 公共静态白名单加入 /fonts/，覆盖 CSS 和 woff2；保留状态域名其他路径限制。 |

额外修复：普通旧 API 请求的 401 不再清除新会话；账户密码确认失败不触发自动退出。

依赖按 package.json 已声明范围更新：Axios 1.20.0、markdown-it 15.0.3、@vue/server-renderer 3.5.44、follow-redirects 1.16.1、PostCSS 8.5.31、source-map-js 1.2.2、nanoid 3.3.20。官方 npm registry 的 audit 返回 0 项漏洞，此结果仅对应当前公告与锁文件。

## 验证

- 后端 `go test ./...` 与 `go vet ./...` 通过（Go 1.27.0 android/arm64）。
- 新增业务回归覆盖漏洞阻断、邮箱验证码、认证版本持久化/回滚、Telegram 权限/停机、HTTP 重定向/各协议网络边界、Passkey 状态公平与并发预算。
- 使用测试生成的 P-256 凭据和真实 ECDSA 签名验证 Passkey Begin → Finish → 可用会话、取消 → TOTP、跨挑战/匿名 Finish 拒绝和重放拒绝，无外部认证器或生产凭据。
- 前端 25 项测试通过；更新依赖后的类型检查与 Vite 生产构建通过，仍有大于 500 kB 的 chunk 提示。
- Android 与共享翻译共 45 项测试通过；Java 来源测试覆盖外部主机、端口、协议、userinfo、非 HTTP URL 和 IPv6。
- Android `gradle :app:assembleDebug` 成功，`apksigner verify` 通过。APK：`android/app/build/outputs/apk/debug/app-debug.apk`。
- 文档 26 个双语页面生成检查、VitePress 构建通过；Git whitespace 检查通过。

## 验证范围与升级行为

- Android 外部认证窗口及重定向尚未真机端到端测试；应在发布前验证实际 OIDC / Cloudflare 身份提供方的交互。
- PostgreSQL 的 auth_version SQL 会通过现有 dialect 转成 BIGINT，但临时 PostgreSQL 实例在当前嵌套 proot 环境初始化超时，未完成数据库实测。
- govulncheck 因下载 golang.org/x/vuln 超时未运行成功，不能宣称 Go 依赖无已知漏洞。
- 升级后反向代理部署需显式配置 TRUSTED_PROXY_CIDRS；否则 IP 限流以代理 peer 为来源。配置示例见 .env.example 与安全文档。
- 同邮箱但尚未绑定 OIDC 的现有账号需先登录本站再绑定；新邮箱需 SMTP 验证；普通用户内网监控改为拒绝，管理员所属监控继续支持内网。
- 已经在修复前绑定的身份或标记验证的邮箱不会被自动解绑/撤销；本次修复阻止新的不安全合并和邮箱更新，不替代对既有异常账户的核查。
