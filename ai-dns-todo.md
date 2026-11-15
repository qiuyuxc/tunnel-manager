# AI 默认参数与 DNS 扩展交接

更新：2026-10-04。已按用户要求续接会话 `01a104e7-b16b-73e3-a32c-00a39f54c023`，完成本地整合验收，并修复验收发现的两端批量请求会话切换问题。当前改动保留在工作区，**尚未提交、推送、发布或部署**。没有调用真实模型、写入线上 DNS 或安装 App；真实服务商、真机和版本发布门禁仍待处理。

## 已完成的本地实现

- AI：创建监控省略检测频率时默认 60 秒，仍不公开、不告警；新增 DNS 省略 TTL / 代理时默认自动 TTL（1）和关闭代理。工具定义、系统提示和服务端规范化同步，实际值写入待确认任务。省略不等于显式无效值；必要信息仍需澄清，不猜资源 ID，不自动执行。
- AI DNS：工具类型扩至 A / AAAA / CNAME / TXT / MX / NS / SRV / CAA / PTR，SRV / CAA 使用结构化 `data`；复用 DNS 后端校验。Web / Android 确认卡保留数据，新增中文字段标签。
- DNS 管理：Web / Android 支持上述九类新增和编辑；仅 A / AAAA / CNAME 可代理，MX 保留优先级，SRV / CAA 字段往返保留，未知类型只读保护。
- 批量新增：A / AAAA / TXT / MX / NS / PTR，同一区域、类型和名称下逐行填值，最多 100 个非空行；行号校验、去重、预览、串行提交和逐条结果。成功行不重发；结果未知时停止队列，先核对再明确确认重试。CNAME / SRV / CAA 使用单条新增。
- 续接修复：Web 出站请求校验批次原会话，切换账户或页面卸载后停止未发送行；Android 绑定原子会话快照与代次，防止服务器/令牌混配，窗口关闭或宿主销毁后停止队列。旧批次 401 不会清除新会话；已在途结果仍按成功、明确失败或未知处理，不误报为未发送。
- 文档：中英文 AI 和 DNS 指南、DNS API 段落已更新；原一键 TXT 的本地收尾见 `ip-selector-todo.md`。

## 核心文件

- `backend/handlers/assistant.go`、`assistant_tools.go`、`assistant_defaults_test.go`、`assistant_dns_test.go`
- `backend/models/dns.go`、`dns_test.go`、`types.go`；`backend/handlers/dns.go`、`dns_test.go`；`backend/services/cloudflare.go`
- `frontend/src/views/DNSManagement.vue`、`frontend/src/components/DNSRecordFields.vue`、`DNSBatchCreate.vue`、`AssistantTasks.vue`
- `frontend/src/api/index.ts`、`frontend/src/utils/dnsValidation.ts` 及测试；`frontend/tests/dns-browser.mjs`、`assistant-browser.mjs`
- `frontend/src/utils/dnsSession.test.ts`：出站校验、异步发送前切换和旧 401 回归。
- `android/app/src/main/java/com/tunnelmanager/app/DNSFragment.java`、`DNSValues.java`、`DNSJson.java`、`DNSRecordEditor.java`、`DNSBatchCreate.java`、`AssistantTaskSelection.java`
- `android/tools/dns-values.test.mjs`、`android/tools/java/com/tunnelmanager/app/DNSValuesTest.java` 及助手标签测试
- `android/app/src/main/java/com/tunnelmanager/app/RequestScope.java`、`Api.java`、`Session.java`；`android/tools/java/com/tunnelmanager/app/DNSBoundaryTest.java`：凭据快照、并发切换、发送前竞态及失效回调隔离。

## 已收取的验证结果

| 检查 | 结果与边界 |
| --- | --- |
| 后端全量测试、构建、vet | 本次运行 `go test ./... -count=1 && go build ./... && go vet ./...` 全部通过 |
| AI 回归 | 默认值、无效值、权限、持久化、结构化 DNS、确认后原样执行及禁止重复执行均通过 |
| AI / TXT 浏览器 | 本次最终复跑通过，含多尺寸/主题、默认值和结构化 DNS 确认卡、TXT 写入模拟和权限；抽查截图 |
| DNS 后端定向测试 | 子代理报告通过 |
| Web 单测和生产构建 | 14/14 通过；整合源码副本与工作区生产构建均通过，不依赖旧 dist |
| DNS 浏览器 | 开发模式及最新生产包 preview 均通过：九类新增编辑、只读、去重/上限、部分失败、未知结果门禁、切换会话、页面卸载及 375px 布局 |
| Android DNS helper 和 Java 编译 | DNS / 请求边界 18/18 通过；主代理最终重跑全部非版本测试 37/37 通过，`:app:assembleDebug` 通过 |
| Android 助手选择与标签 | 5/5 通过 |
| Android 既有版本门禁 | 本次单独重跑仍为 3 项失败：2.6.1 vs 2.7.0、260 vs 261、变更记录 2.6.0 vs 2.7.0；未修改版本字段 |
| 文档构建 | 合并最终 AI / DNS 指南后 `npm run build` 通过 |
| 最终空白检查 | `git diff --check` 通过 |

浏览器全部拦截 API 并阻断外部请求，没有发真实资源写入。最终截图位于 `$TMPDIR/dns-final-screenshots`、`$TMPDIR/dns-production-final-screenshots`、`$TMPDIR/assistant-final-screenshots`、`$TMPDIR/lab-final-screenshots`。Web 隔离验收副本为 `$TMPDIR/tunnel-web-final.Iay9ev/frontend`：复制已跟踪及未跟踪源码，复用工作区 node_modules，未声称全新依赖安装。

APK：`android/app/build/outputs/apk/debug/app-debug.apk`，已在后续问题处理中更新，生成时间 **2026-10-04 20:01:20 +0800**，包含 DNS / 会话保护、绑定运行反馈、优选诊断和页面动效，SHA-256 为 `4fef37c3d889f57dd299a3dc38a1a11d974bd27de6175df7068d86f7df45e3ba`。它仍是现有版本号的调试包，不是发布附件。`adb devices -l` 无设备，构建及 Java 回归不代表真机交互验收。

新增缺陷先失败后通过：Web 浏览器原本切换会话后仍发送 3 条（期望只发在途的 1 条）；旧批次 401 原本会清空新 token。Android 初始边界回归 5 项失败，修复后 18 项 DNS 测试通过。

## 后续三项问题处理（2026-10-04）

- **绑定运行反馈**：Web 原有按钮内旋转提示，App 单条 / 批量绑定没有 pending 状态。现双端增加运行卡、已等待秒数、长等待提示和防重复点击；请求期间锁定表单，结束后恢复。Android 保存转发地址也有反馈，离开视图后计时器释放；减少动态效果时仍有文字和计时，不显示旋转。
- **页面动效**：Web 私有路由短淡入与 6px 位移，退出淡出；Android 页面进入与返回使用短淡入 / 小幅位移。尊重系统或应用的减少动态效果设置，不增加等待弹层。
- **优选判定**：按本机留存的 `Host/SNI=www.dkg.us.ci`、路径 `/`、接受 `200` 对用户指定的 `43.175.131.30` 做单点只读 GET，TLS 校验通过但返回 `403`；同一 Host 普通解析访问返回 `200`。因此在这套探针配置下未命中是预期筛选结果，不代表 IP 不可达。未修改接受状态、TLS 验证、TXT 授权或线上 DNS；本机配置和网络不一定等于用户线上环境，不能将这一结果泛化为全部场景。
- **ITDOG 对照补证（2026-10-04）**：用户补充实际测试域名为绑定至该 IP 的 `am.kukie.cn`，并提供 HTTP/2 200 响应。20:25（UTC+8）本机固定连接 `43.175.131.30:443`，使用扫描器 User-Agent、GET `/`、正常证书校验且不跟随跳转对照：`am.kukie.cn` 的 HTTP/1.1、HTTP/2 均为 200，`www.dkg.us.ci` 两种协议均为 403；`am.kukie.cn` 普通 DNS 请求的远端 IP 也为该 IP，并返回 200。随后用隔离临时程序直接调用当前 `ScanLabIPs`：前者命中并选中该 IP，后者返回 403 且未命中。本次差异随 Host/SNI 域名变化，未复现 HTTP/2 兼容或扫描器错误过滤 200 的问题。本机只读确认留存 Host/SNI 仍为 `www.dkg.us.ci`，不能代表线上当前配置；仍需对照线上失败任务参数。未调用完整任务执行器、TXT 验证或 DNS 写入，未修改扫描算法、证书策略、应用配置或授权流程。
- **线上探针域名补证（2026-10-04 20:30–20:31，UTC+8）**：用户确认线上路径 `/`、目标域名 `ss.dkg.us.ci`，并允许对照未绑定该 IP 的 `tm.kukie.cn`。本机令 Host/SNI 同为各自域名，固定连接 `43.175.131.30:443`，使用扫描器请求头、正常 TLS 校验且不跟随跳转：`ss.dkg.us.ci` 在 HTTP/1.1、HTTP/2 下均为 403，`tm.kukie.cn` 均为 200；实际 `ScanLabIPs` 同样分别未命中、命中。关键补证：前者响应含 `Cf-Mitigated: challenge`，正文标题 `Just a moment...`，与 Cloudflare 官方挑战响应标志一致；Ray ID 为 `a45433f3dd37d604-HKG`。因此本次复现的直接原因是探针请求收到挑战页，而非 IP 不可达、HTTP/2 不兼容或把 200 错判为未命中。普通 DNS 请求分别连接 `172.67.139.69`、`103.199.103.204`，均为 200，不能替代固定 IP 测试。尚未读取线上安全事件，不能确定具体触发规则，也不能声称线上历史请求已查实。当前扫描器只读状态行，新增诊断会显示 HTTP 403，但尚不能自动标注挑战页。没有修改生产配置、DNS、WAF、TXT 授权或接受码。
- **诊断缺失修复**：执行器此前丢弃未命中逐项响应，失败任务只能看到汇总。新历史保存本次 `probe` 参数、`rejected` 总数和最多 100 条 `diagnostics`；双端展示 HTTP 状态 / TLS / 连接错误，Web 手机端分行展示完整原因。旧历史没有详情，需重新执行或单独复测。未命中段提示改为先核对原因，不直接建议剔除。
- **新增验证**：执行器诊断回归先失败后通过，包含失败任务保留原因、100 条上限、历史参数快照和不改变选中结果。后端全量 test / build / vet 通过；Web 15 项单测与最新生产构建通过；Android 37 项非版本 helper 测试与最新 `assembleDebug` 通过；文档构建通过。
- **浏览器验证**：新增 `frontend/tests/binding-browser.mjs`，验证延迟响应下旋转 / 计时 / 禁重 / 恢复、路由动效及减少动态效果。实验室、DNS、AI 回归均通过；实验室额外验证 403 / TLS 超时详情与 375px 不截断。截图见 `$TMPDIR/binding-screenshots`、`$TMPDIR/lab-motion-screenshots/375-diagnostics.png`、`$TMPDIR/dns-motion-regression`、`$TMPDIR/assistant-motion-regression`，已抽查。
- **仍未完成**：没有 Android 真机，不能声称原生动画观感已验收。仍未提交、推送、部署或安装 APK，未做真实 DNS 写入；既有三项版本门禁未改动。若线上 Host / SNI / 路径已换，需按实际配置及部署网络再次核对示例 IP。

## 下次从这里继续

- **预览登录后续调整（2026-10-04 20:50，UTC+8）**：按用户要求，仅在隔离预览副本重置现有 `123456` 账户密码，随后将该账户提升为管理员，撤销旧会话并重启。当前执行会话为 `75110`，替代原会话；地址和隔离目录不变。管理员登录与用户管理接口实测 200；优选设置接口因副本尚未开启实验功能返回 404，未调整功能开关。接口验证使用前端实际的 `X-Auth-Token` 请求头，最新验证会话退出后确认返回 401。原库同名账户仍为普通用户，原管理员及线上账户未改动。

- **用户验收预览已启动（2026-10-04 20:43，UTC+8）**：地址 `http://127.0.0.1:18085`，执行会话 `81327`；当前源码编译的后端直接提供最新 `frontend/dist`。隔离目录 `/data/data/com.termux/files/usr/tmp/tm-preview.T3as5i`，SQLite 通过只读备份复制本机 `backend/data/tm.db`，原库未改动。副本停用自动优选、默认 DNS 更新、Telegram 机器人和通知，未加载旧监控任务，清空复制的登录会话；账户和其余配置保留。首页、静态资源 GET、健康检查和登录配置接口通过，实验室接口未登录返回 401。用户手动绑定 / DNS 提交仍可能通过保留的 Cloudflare 连接产生真实操作，应使用测试域名。仅供用户验收，尚未授权提交或推送；收到明确验收确认后再处理发布。

- [x] 复查并修复批量新增会话切换边界；部分成功、未知结果和后端严格校验回归通过。
- [x] Web 14 项单测、最新生产构建与隔离源码副本构建通过。
- [x] Android DNS / 助手及全部非版本 helper 测试通过，生成最新 debug APK。
- [x] 最终文档构建和后端全量测试 / 构建 / vet 通过。
- [x] 最终复跑 DNS、AI 和 TXT 浏览器脚本；DNS 另验生产 preview。复跑可共用独立 CDP 19224，开发 Vite 18085、生产 preview 18086；按脚本指定环境变量并串行运行，避免模拟登录互相影响。
- [ ] Android 实机交互和真实模型体验仍待验证；真实 Cloudflare 写入需要明确测试区域授权。
- [ ] 不在本次顺带解决既有版本门禁；提交 / 推送 / 发布 / 部署另等用户授权。新增文件较多，不能只看 `git diff --stat` 漏掉未跟踪文件。

原会话的两名子代理和本次 Android 子代理均已关闭。验收用 Vite / preview / 独立 Chromium 已停止；保留截图、隔离源码和临时目录，没有清理源码或业务数据。
