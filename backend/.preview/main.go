package main

import (
	"fmt"
	"os"
	"path/filepath"

	"tunnel-manager/services"
)

type item struct {
	label string
	plain string
	html  string
}

func main() {
	p1, h1 := services.VerificationCodeEmail("注册", "935273", 10)
	p2, h2 := services.VerificationCodeEmail("密码重置", "482910", 10)
	p3, h3 := services.TestEmail()
	p4, h4 := services.AlertEmail("核心服务监控", "官网首页", "https://example.com", "ok", "down", 503, "dial tcp: connection refused", "2026-08-28 22:10:00")
	p5, h5 := services.AlertEmail("核心服务监控", "官网首页", "https://example.com", "down", "ok", 200, "", "2026-08-28 22:14:00")
	items := []item{
		{"① 注册验证码邮件", p1, h1},
		{"② 密码重置验证码邮件", p2, h2},
		{"③ SMTP 测试邮件", p3, h3},
		{"④ 告警邮件 · 服务异常", p4, h4},
		{"⑤ 告警邮件 · 服务恢复", p5, h5},
	}

	outDir := filepath.Join("..", "docs", ".vitepress", "dist", "email-preview")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	page := `<!doctype html><html><head><meta charset="utf-8"><title>邮件模板预览 · Tunnel Manager</title><style>
	body{background:#111318;color:#e5e7eb;font-family:-apple-system,PingFang SC,Microsoft YaHei,sans-serif;margin:0;padding:32px;}
	h1{font-size:20px;margin:0 0 4px;}
	.sub{color:#9ca3af;font-size:13px;margin:0 0 28px;}
	h2{font-size:15px;margin:28px 0 10px;color:#93c5fd;}
	iframe{width:100%;height:640px;border:1px solid #374151;border-radius:8px;background:#fff;}
	summary{cursor:pointer;color:#9ca3af;font-size:13px;margin:8px 0;}
	pre{background:#1f2937;color:#d1d5db;padding:12px;border-radius:6px;font-size:12px;white-space:pre-wrap;}
	</style></head><body>`
	page += `<h1>邮件模板预览</h1><p class="sub">以下均为实际模板代码的渲染结果，与真实邮件一致。</p>`
	for i, it := range items {
		fname := fmt.Sprintf("%d.html", i+1)
		if err := os.WriteFile(filepath.Join(outDir, fname), []byte(it.html), 0o644); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		page += `<h2>` + it.label + `</h2><iframe src="` + fname + `"></iframe><details><summary>纯文本版本（不支持 HTML 的客户端收到这样）</summary><pre>` + it.plain + `</pre></details>`
	}
	page += `</body></html>`
	if err := os.WriteFile(filepath.Join(outDir, "index.html"), []byte(page), 0o644); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("preview written to", outDir)
}
