# 单用户精简版（slim）

单用户精简版是 Tunnel Manager 面向**单管理员自托管**场景的裁剪构建，对应仓库的 `slim/single-user` 分支。它去掉了多人协作相关的全部能力，只保留一个管理员自己使用的隧道 / DNS / 监控能力，代码量约为完整版的一半。

需要多用户、用户组权限、管理后台或 Telegram 远程控制的团队，请使用完整版（`main` 分支）。

## 适合谁

- 只有你一个人使用，不需要给别人开账号
- 想要更小的攻击面与更少的运维项（没有注册入口、没有审计表、没有远程控制 Bot）
- 仍然需要通行密钥 / 双因素、Cloudflare OAuth、服务监控与告警、公开状态页

## 相比完整版的差异

| 移除 | 保留 |
| --- | --- |
| 多用户注册、邮箱验证码、邀请码 | 单个管理员（安装引导时设置，无注册入口） |
| 用户组与权限体系 | 密码 + 通行密钥（WebAuthn）+ TOTP 双因素登录 |
| 管理后台（用户 / 用户组 / 邀请码管理） | 登录限流、Turnstile 人机验证 |
| 管理员操作审计日志 | Cloudflare OAuth（PKCE，多账户）与静态 Token |
| Telegram Bot **远程控制**（指令操作隧道 / DNS） | Telegram **通知**（仅对外推送告警，与远程控制是两套独立 Token） |

原管理后台里的系统配置卡片——SMTP、Cloudflare OAuth 客户端、应用加密密钥、通行密钥依赖方 / 来源、登录限流、Turnstile、实验性功能开关——都迁移到了面板内的**全局设置**页，端点不变，只是不再需要管理员专属入口。

隧道管理、域名绑定（含批量）、DNS 管理、IP 优选实验室、服务监控与公开状态页、邮件告警均与完整版一致。

## 部署

与完整版一样，只是安装脚本跟踪的是 `slim/single-user` 分支。

```bash
git clone -b slim/single-user https://github.com/qiuyuxc/tunnel-manager.git
cd tunnel-manager
./install.sh
```

`install.sh` 会自动克隆并在后续更新时拉取 `slim/single-user` 分支（不会拉 `main`）。首次打开面板进入安装引导：选择 SQLite 或 PostgreSQL，然后设置管理员用户名与密码——这个账户就是整个面板。

其它方式：

- **Docker 镜像**：精简版镜像标签为 `:slim`（以及 `:<版本>-slim`），例如 `ghcr.io/qiuyuxc/tunnel-manager:slim`。
- **二进制**：从 [GitHub Releases](https://github.com/qiuyuxc/tunnel-manager/releases) 下载 `vX.Y.Z-slim` 标签对应的 pre-release 资产。

其余部署细节见 [Docker Compose 部署详解](/guide/docker-compose) 与 [二进制部署](/guide/binary-deploy)。

## 版本与更新检查

精简版的发布独立于完整版：

- 版本号带 `-slim` 后缀（如 `v2.6.0-slim`），在 GitHub 上以 **pre-release** 发布。
- 面板「关于」页与 App 的更新检查**只与带 `-slim` 的发布比较**——因此完整版发新版时，精简版不会误报「可更新」，反之亦然。
- 因为精简版的发布始终标记为 pre-release，它不会成为仓库的 `releases/latest`，也就不会影响完整版的更新提示。

## 与完整版保持同步

`slim/single-user` 是 `main` 的长期精简子集。共享代码（隧道、DNS、监控、Cloudflare OAuth、通行密钥）的修复请从 `main` **cherry-pick** 对应提交到精简版分支：

```bash
git checkout slim/single-user
git cherry-pick <main 上的提交>
```

不要对精简版整体 `git merge main`——那会尝试把已删除的多用户 / 审计 / 远程控制代码重新引入，在每个被删文件上产生冲突。
