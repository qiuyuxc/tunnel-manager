# 3.0 审查复现

审查结论见仓库根目录的 [security-review-v3.md](../../security-review-v3.md)。

运行：

```sh
python3 tools/security-review/run.py
```

脚本使用 Go overlay 临时向 `handlers`、`services` 包提供复现代码，不修改正式代码或现有测试。数据库和模拟 HTTP 服务都创建在本机临时目录中，测试不访问真实 Cloudflare / Telegram 账户，不发送真实邮件，不扫描生产网络。

**这些复现的 PASS 表示当前问题被成功重现，不表示安全检查通过。** 修复后对应复现可能失败，应将用例改写为验证修复行为的正常回归测试。Android 的跨源注入结论来自静态代码路径核查，不包含真机攻击测试。

修复后的正常回归测试已经放入 `backend/handlers/`、`backend/services/` 和 `backend/store/`，随 `cd backend && go test ./...` 运行；原 `repro/` 保留为修复前证据。最新修复结果见仓库根目录 `security-fixes-v3.md`。
