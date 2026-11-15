# Tunnel Manager 3.0 代码审查

审查日期：2026-10-23。基线：`1b0951d`，应用版本 `3.0.0`。重点覆盖 Go API 的身份与权限、多用户隔离、OIDC / Passkey、Cloudflare 连接、Telegram、AI 任务执行、监控与公开状态页，以及 Android WebView 的会话传递。

本轮确认 **11 项问题：6 项 P1、4 项 P2、1 项 P3**。优先处理 Android 令牌外泄、OIDC 账户预先接管和 Telegram 权限绕过。结论是静态审查与本机隔离复现的结果，不代表已经验证生产部署可被攻击；每项的前提见下文。正式业务代码未修改，现有未提交笔记与实验目录保留。

P1：应优先修复的安全或全站可用性问题；P2：应修复的条件性信息泄露、限流缺陷或功能错误；P3：影响显示的一般缺陷。

## 验证与复现

- 现有 `go test ./...`、`go vet ./...` 通过；运行环境为 Go 1.27.0 android/arm64，仓库声明 Go 1.26。
- 前端 `npm test` 的 24 项测试通过；`npm run build` 的类型检查与生产构建通过。构建有大于 500 kB 的 chunk 提示。
- 新增 10 个隔离复现，使用临时 SQLite 数据库、HTTP 模拟服务和测试令牌，无真实 Cloudflare / Telegram 操作或邮件发送。运行 `python3 tools/security-review/run.py`。
- **复现用例的 PASS 表示问题被成功复现，不能解释为产品安全测试通过。** 测试代码放在 `tools/security-review/repro/`，通过 Go overlay 临时装入测试包，避免混入常规测试。
- Android 项为静态代码路径核查，没有执行真机令牌窃取；Go 依赖的 `govulncheck` 因下载工具时网络超时未完成，不能据此断言 Go 依赖没有已知漏洞。

## 1. [P1] Android WebView 向跨源页面注入面板会话令牌

位置：`android/app/src/main/java/com/tunnelmanager/app/WebPageFragment.java:80`、`:99`；同类路径位于 `MainActivity.java:369`、`:1278`。正常触发入口包括 `frontend/src/components/OIDCAccount.vue:83` 的第三方身份绑定跳转。

`onPageFinished` 无条件调用 `injectSession`，后者把 `Session.token()` 写入**当前页面源**的 `localStorage.auth_token`。WebView 没有实现限制来源的导航拦截，也没有在注入前比较 URL 与面板的 scheme、host、port。用户在已登录的 `/account` 中跳转到 OIDC / Cloudflare 授权页面，或跟随外部链接与重定向时，外部页面会收到面板令牌。外部页面自己的脚本随后可读取或监听该存储值，攻击者无需先突破面板页面的同源限制。

影响：外部页面运营方或受控制的第三方页面可获得该用户的面板会话，管理员会话同样受影响。MainActivity 还向所有加载页面暴露 `TMHost`，扩大了外部页面可触发的原生操作范围。

建议：在每次脚本注入前严格核对面板来源；将第三方导航送入外部浏览器 / Custom Tabs，单独设计认证回调；仅向可信面板页面开放原生桥。来源校验应覆盖重定向和回调时页面已再次导航的情况。

证据：静态追踪了 account 页面 → `window.location.assign(authorization_url)` → WebView 页面完成 → 会话注入。未进行 Android 真机攻击测试。

## 2. [P1] 未验证的邮箱更换与 OIDC 自动合并构成账户预先接管

位置：`backend/handlers/admin.go:707`、`backend/store/oidc.go:182`、`backend/handlers/oidc.go:530`。

`ChangeEmail` 只确认当前本站密码，随后调用 `SetUserEmail(userID, email, email != "")`，任意非空新地址立即被标记为已验证，没有向新邮箱发送验证码。OIDC 首次登录会按邮箱寻找本地账户；只要第三方提供了该邮箱的有效证明，就将 OIDC 身份合并到已有账户，而且保留原本站密码与会话。

复现：攻击者先持有自己的普通账户，把邮箱改成尚未被占用的受害者地址；受害者随后首次通过返回 `email_verified=true` 的 OIDC 身份登录，实际登录到了攻击者建立的账户。攻击者原密码仍然有效，可继续访问受害者之后授权的 Cloudflare 连接和数据。已注册且占用该邮箱的受害者不会被这条路径覆盖。

建议：新邮箱先保存为待验证，确认归属后再变更；OIDC 与已有本站账户的合并应同时证明当前本站账户所有权。不能仅依靠第三方邮箱证明，把第三方身份加入仍保留他人登录凭据的账户。修复时应一并检查免邮箱验证注册造成的同类合并风险。

证据：`TestSecurityReviewEmailClaimPreHijacksOIDCAccount` 通过实际邮箱更新与 OIDC Complete 处理器复现；OIDC claims 在测试中作为已验证的上游结果注入，不涉及绕过 ID Token 签名。

## 3. [P1] Telegram 命令绕过用户组权限与账户禁用

位置：`backend/services/telegram.go:370`、`:398`；写操作示例位于 `backend/services/telegram_commands.go:379`；生命周期位于 `backend/services/user_telegram.go:111`。

Bot 只检查来信者 TG ID 是否在配置的操作者名单内，命令分发和业务执行均没有检查面板账户状态或 `HasPerm`。`Reconcile` 也不排除禁用账户。因此，即使管理员从用户组中移除 tunnels / DNS / domain_bind 权限，用户仍能通过已配置 Bot 执行这些操作；禁用账户、删除其 Web 会话也不会停止 Bot 访问原有 Cloudflare 授权。

影响：Web / AI 接口设置的权限边界在 Telegram 上失效。仍使用该账户自己的 Cloudflare 凭据，复现未发现由此跨到其他用户的连接。

建议：每条命令执行前重新读取 Bot 所属面板账户状态与当前权限；为命令和别名统一映射权限；禁用或删除账户时停止 Bot。删除确认等延迟操作须在最终执行时再次校验。

证据：`TestSecurityReviewTelegramBypassesGroupAndDisabledStatus` 使用真实用户所属的模拟 OAuth 连接；缺少 tunnels 权限时 `/tunnels` 仍发出 Cloudflare 请求，账户禁用后下一条命令依然发出请求。未执行真实 DNS 写操作，写操作的同类路径由代码确认。

## 4. [P1] 普通监控权限可触发服务器内网 HTTP / TCP 请求

位置：`backend/handlers/monitors.go:475`、`:529`、`:649`；`backend/services/probe.go:24`、`:42`、`:84`。

目标校验只限制协议与字段格式，没有检查目的地址。普通用户可创建 HTTP GET / POST、TCP 目标，随后通过 CheckNow 或定时探测，让服务器访问回环、私网、链路本地与元数据地址。默认 HTTP 客户端还会跟随重定向，因此只校验最初输入的主机也不足以修复。

影响：至少可进行内网端口 / HTTP 状态探测，触发内部无鉴权 GET / 空 body POST 接口。已复现普通 monitor-only 用户向本机 HTTP 服务发送 POST。现有响应只暴露状态和错误，没有返回完整内部响应体，本轮不据此声称已读取云元数据凭据或实现 RCE。

建议：按用户权限区分公网探测和显式授权的内网监控；默认拒绝元数据、链路本地、回环与未授权私网，允许管理员按需配置内部网段。对 DNS 解析后的实际拨号地址、每次重定向和 TCP 目标统一检查，避免 DNS rebinding。共享部署还需对手动探测和并发数设置上限。

证据：`TestSecurityReviewMonitorReachesLoopback` 经过 Auth / RequirePerm / Create / AddTarget / CheckNow 的实际处理器，模拟内网服务收到 POST。

## 5. [P1] 密码重置没有使已签发的二次验证挑战失效

位置：`backend/handlers/auth.go:443`、`backend/handlers/management.go:139`、`backend/handlers/admin.go:896`、`:929`。

忘记密码与管理员重置只更新密码并删除已有 sessions，没有清理或使旧登录 challenge 失效。Challenge 只绑定 userID 和内存 authEpoch，未绑定该账户的密码版本；这两条重置路径也不增加该 epoch。重置前通过旧密码取得的 challenge，重置后仍可用有效 TOTP / 恢复码换取新 session。

影响：密码重置宣称的会话撤销边界不完整。利用者仍需持有一个未过期 challenge 和有效第二因素；**本问题不绕过 TOTP**。与本站 `ChangePassword` 不同，那条路径还会调用 `revokeAllAuthState`。

建议：引入按账户的认证版本，将挑战与会话签发绑定到取得第一因素时的版本；密码重置、禁用、关键凭据变更应原子更新版本并撤销该账户的待完成认证。签发 session 的最终步骤须再次核对该版本。

证据：`TestSecurityReviewPasswordResetLeavesOldChallenge`：重置成功后，用重置前 challenge 与有效 TOTP 获得 HTTP 200，并验证新 session 可用。

## 6. [P1] 匿名 Passkey Begin 未计费，可耗尽全站认证状态容量

位置：`backend/handlers/passkey.go:355`、`:47`；`backend/services/passkey.go:32`、`:381`。

LoginBegin 调用 `guardPasskey` 检查预算，但成功开始 ceremony 后不调用 `chargePasskey`。只有部分完成失败路径才计费。因此同一个 IP 可以连续创建全部 256 个全局 ceremony，使之后所有用户的登录 / 注册 ceremony 因容量用尽失败。状态保留 5 分钟，可通过持续请求维持。

建议：在分配状态之前原子获取并计入开始请求预算，同时按来源 / 账户限制待完成 ceremony 数量；容量控制应避免一个匿名来源占满所有登录槽位。

证据：`TestSecurityReviewAnonymousPasskeyBeginExhaustsCapacity`：配置每 IP 额度为 1，同一 IP 仍成功创建 256 个 ceremony，第 257 次收到全局容量错误，而非限流。

## 7. [P2] 客户端可伪造 X-Forwarded-For 绕过 IP 限流

位置：`backend/handlers/turnstile.go:43`；共享调用位于 `backend/handlers/ratelimit.go:68`、`:103`。

`clientIP` 无条件取 XFF 最左端，不检查 RemoteAddr 是否是可信代理，也不验证地址。客户端可以改变这个字段来切换 IP 计数，同时伪造审计来源和已记住的熟悉 IP。即使经过 Cloudflare，官方文档说明已有 XFF 会被保留并追加代理地址，最左端仍不能直接当作可信来源。

影响：每 IP 限额可绕过，熟悉来源的账号限额可被伪装后放宽。默认每账号限额仍独立存在，不能仅凭这一点声称单账户密码猜测无限制。

建议：明确可信代理配置；直连用 RemoteAddr，代理场景从可信链右侧解析，Cloudflare 场景只有确认流量来自可信入口后才接受其原始 IP 头。保存熟悉来源前统一规范化、校验 IP。

证据：`TestSecurityReviewForwardedIPChangesRateBudget`：保持 RemoteAddr 和 XFF 链尾相同，只改最左端即可从预算耗尽变为允许。

参考：[Cloudflare HTTP headers](https://developers.cloudflare.com/fundamentals/reference/http-headers/)，检索时页面更新日期为 2026-10-09。

## 8. [P2] Passkey 二次验证开始成功后永久占用同一登录挑战

位置：`backend/handlers/passkey.go:432`、`:443`、`:456`；`backend/handlers/admin.go:902`。

TwoFactorBegin 调用 claimChallenge 把 `inUse` 置为 true，仅失败分支释放。成功返回后，TwoFactorFinish 再次 claim 相同 challenge，立即因 `inUse=true` 返回 401，根本不会验证 WebAuthn assertion。改用有效 TOTP 也会被同一锁拒绝，只能重新开始登录。

建议：开始阶段使用可释放的检查，在返回 options 前释放；将 ceremony 与该 challenge 绑定，最终完成阶段原子独占并消费。覆盖“开始成功 → 完成成功”与“用户取消 → 回到 TOTP”两条端到端流程。

证据：`TestSecurityReviewPasskeyFactorLocksChallenge`：Begin=200，Finish 在断言验证前=401，正确 TOTP 回退=401，challenge 仍 inUse。

## 9. [P2] 公开状态页通过错误文本泄露隐藏目标和 URL 参数

位置：`backend/handlers/monitors.go:1072`、`:1105`；`backend/services/probe.go:157`。

`link_enabled=false` 只隐藏 link，未隐藏 error。Go HTTP transport 错误可能带完整请求 URL；`ProbeErrorText` 仅在文本超过 80 字符时尝试截短，`publicErrorText` 又只从最后一个冒号截断，不能可靠移除主机、路径、查询参数。

影响：公开监控失败时，匿名访问者可能看到原本关闭链接的内部地址和短 URL 中的 token / key。不同错误与 URL 长度的结果不同，不是所有 transport 错误都会泄露完整 URL。

建议：公开接口只输出结构化错误类别或经过明确脱敏的固定文本；内部详细错误与匿名返回分开。不要用字符串长度和冒号位置代替 URL 脱敏。

证据：`TestSecurityReviewPublicFailureLeaksHiddenURL` 使用正常取消请求产生的短 transport 错误，未登录 public API 返回 `review-secret`，目标的 link_enabled=false。

## 10. [P2] OAuth 状态实际路由遗漏 Cloudflare 客户端，账户列表恒为空

位置：`backend/main.go:362`、`backend/handlers/middleware.go:132`、`backend/handlers/cloudflare_oauth.go:78`。

Status 从 `UserCF(r)` 读取 Cloudflare 客户端，但路由使用的 SessionOnly 只附加用户，不附加 CF context。实际请求总是跳过 ListAccounts，返回 connected=true、accounts=[]，账户页据此隐藏 Cloudflare account 选择器。现有测试手动注入了 withCF，掩盖路由问题。

建议：保持 session-only 要求的同时提供账户所属客户端，或由 Status 明确 `h.cf.ForUser(uid)`；增加经实际 middleware 的状态查询测试。

证据：`TestSecurityReviewOAuthStatusDropsCFContext` 使用实际 SessionOnly wrapper，已连接但列表为空，而且模拟上游调用次数为 0。

## 11. [P3] 自定义状态页域名遗漏字体资源白名单

位置：`backend/handlers/status_domain.go:82`；`frontend/index.html:9`、`:11`。

自定义域名只允许 assets / uploads 等路径，拒绝 `/fonts/fonts.css` 与字体 woff2。3.0 HTML 直接引用这些资源，因此主域名显示正常的字体在自定义状态页上均为 404，导致回退字体和排版变化。

建议：仅将实际需要的字体目录纳入公共静态资源白名单，并覆盖字体 CSS 与 woff2 请求。不要因此开放全部 SPA 或 API 路由。

证据：`TestSecurityReviewCustomStatusDomainBlocksFont`：已发布状态域名请求 `/fonts/fonts.css`，静态处理器尚未执行就返回 404。

## 依赖与其他观察

直接向 npm 官方安全公告 bulk API 查询并按 `frontend/package-lock.json` 版本匹配，7 个包共匹配 24 条公告：`@vue/server-renderer 3.5.39`、`axios 1.18.1`、`follow-redirects 1.16.0`、`markdown-it 15.0.0`、`nanoid 3.3.15`、`postcss 8.5.16`、`source-map-js 1.2.1`。

匹配版本不等于当前应用可被利用。前端是浏览器 SPA，未发现服务端 Vue SSR；多条 Axios / follow-redirects 公告涉及 Node adapter，nanoid / PostCSS 公告也需要特定输入或构建场景。markdown-it 在 About 页解析官方 GitHub release body，html=false、linkify=true，输入截为 20,000 字符；默认 reference/linkify 解析的复杂度公告与使用路径相关，但未构造页面 DoS 复现。

建议在安全修复后单独刷新依赖锁并验证构建，优先评估 [markdown-it 默认解析复杂度公告](https://github.com/advisories/GHSA-gfr2-rv44-r23c) 和 [linkify 公告](https://github.com/advisories/GHSA-253c-mchw-3w2r)。按当前公告范围，markdown-it 15.0.3、Axios 1.20.0 是对应已列公告的范围外版本；这些数字不表示覆盖未来公告。

还观察到前端普通 401 响应会自动退出：`frontend/src/api/index.ts:37` 仅在配置 validateSession 的请求上比较当前 token。普通慢请求的旧 401 可能清除新会话，账户页密码确认错误也会触发退出。此项未进行浏览器复现，未计入上述 11 项确认结论。

本轮 AI 任务执行路径已有权限检查、账户状态检查、Cloudflare 连接标记、资源快照与显式确认；OIDC 的 state / nonce / PKCE 和浏览器 cookie 绑定也已有实现。本轮未发现可直接绕过这些检查的已确认问题，但这些局部结论不能抵消上述入口与账户合并风险。

## 建议修复顺序

1. 封住 Android 跨源会话注入、邮箱 / OIDC 预先接管和 Telegram 越权。
2. 为监控建立网络访问边界，修复密码重置认证版本与匿名 Passkey 状态计费。
3. 修复可信 IP 解析、Passkey 二次验证锁、公开错误脱敏、OAuth context 与状态页字体。
4. 刷新依赖并回归上述复现涉及的完整登录和权限流程，再进行浏览器、Android 真机与 PostgreSQL 场景验证。
