package services

import (
	"fmt"
	"html"
)

// emailShell wraps an email body into a simple branded card layout.
func emailShell(kicker, title, inner string) string {
	return "<!doctype html><html><body style=\"margin:0;padding:24px;background:#f4f5f7;font-family:-apple-system,Segoe UI,PingFang SC,Microsoft YaHei,sans-serif;\">" +
		"<div style=\"max-width:520px;margin:0 auto;background:#ffffff;border-radius:12px;padding:32px;border:1px solid #e5e7eb;\">" +
		"<p style=\"margin:0 0 4px;font-size:12px;color:#9ca3af;letter-spacing:2px;\">" + html.EscapeString(kicker) + "</p>" +
		"<h2 style=\"margin:0 0 16px;font-size:20px;color:#111827;\">" + html.EscapeString(title) + "</h2>" +
		inner +
		"<p style=\"margin:24px 0 0;font-size:12px;color:#9ca3af;\">此邮件由系统自动发送，请勿回复。</p>" +
		"</div></body></html>"
}

// VerificationCodeEmail renders the register / password-reset code mail.
func VerificationCodeEmail(kind, code string, minutes int) (string, string) {
	plain := fmt.Sprintf("您的%s验证码是：%s\n\n%d 分钟内有效。若非本人操作请忽略本邮件。", kind, code, minutes)
	inner := fmt.Sprintf(
		"<p style=\"margin:0 0 12px;font-size:14px;color:#374151;\">您的%s验证码：</p>"+
			"<div style=\"font-size:32px;font-weight:700;letter-spacing:10px;color:#111827;font-family:Menlo,Consolas,monospace;background:#f9fafb;border:1px solid #e5e7eb;border-radius:8px;padding:16px;text-align:center;\">%s</div>"+
			"<p style=\"margin:16px 0 0;font-size:14px;color:#6b7280;\">%d 分钟内有效。若非本人操作请忽略本邮件。</p>",
		html.EscapeString(kind), html.EscapeString(code), minutes)
	return plain, emailShell("Tunnel Manager", kind+"验证码", inner)
}

// TestEmail renders the SMTP test mail.
func TestEmail() (string, string) {
	plain := "这是一封来自 Tunnel Manager 的测试邮件，收到即表示 SMTP 配置有效。"
	inner := "<p style=\"margin:0;font-size:14px;color:#374151;\">这是一封来自 Tunnel Manager 的测试邮件，收到即表示 SMTP 配置有效。</p>"
	return plain, emailShell("Tunnel Manager", "SMTP 测试邮件", inner)
}

// AlertEmail renders the monitor state-change notification.
func AlertEmail(monitorName, targetName, targetURL, previous, current string, httpCode int, errMsg, when string) (string, string) {
	recovered := current == "ok"
	stateLabel := "异常"
	color := "#dc2626"
	if recovered {
		stateLabel = "已恢复"
		color = "#059669"
	}
	plain := fmt.Sprintf("监控项目：%s\n探测目标：%s（%s）\n状态变化：%s -> %s\n", monitorName, targetName, targetURL, previous, current)
	if httpCode > 0 {
		plain += fmt.Sprintf("HTTP 状态码：%d\n", httpCode)
	}
	if errMsg != "" {
		plain += fmt.Sprintf("错误信息：%s\n", errMsg)
	}
	plain += fmt.Sprintf("时间：%s", when)
	codeRow := ""
	if httpCode > 0 {
		codeRow = fmt.Sprintf("<tr><td style=\"padding:6px 0;color:#6b7280;\">HTTP 状态码</td><td style=\"padding:6px 0;color:#111827;\">%d</td></tr>", httpCode)
	}
	errRow := ""
	if errMsg != "" {
		errRow = fmt.Sprintf("<tr><td style=\"padding:6px 0;color:#6b7280;\">错误信息</td><td style=\"padding:6px 0;color:#111827;\">%s</td></tr>", html.EscapeString(errMsg))
	}
	inner := fmt.Sprintf(
		"<table style=\"border-collapse:collapse;font-size:14px;width:100%%;\">"+
			"<tr><td style=\"padding:6px 0;color:#6b7280;\">监控项目</td><td style=\"padding:6px 0;color:#111827;\">%s</td></tr>"+
			"<tr><td style=\"padding:6px 0;color:#6b7280;\">探测目标</td><td style=\"padding:6px 0;color:#111827;\">%s<br><span style=\"color:#9ca3af;font-size:12px;\">%s</span></td></tr>"+
			"<tr><td style=\"padding:6px 0;color:#6b7280;\">状态变化</td><td style=\"padding:6px 0;\"><span style=\"color:#6b7280;\">%s</span> → <strong style=\"color:%s;\">%s</strong></td></tr>"+
			codeRow+errRow+
			"<tr><td style=\"padding:6px 0;color:#6b7280;\">时间</td><td style=\"padding:6px 0;color:#111827;\">%s</td></tr>"+
			"</table>",
		html.EscapeString(monitorName), html.EscapeString(targetName), html.EscapeString(targetURL),
		html.EscapeString(previous), color, html.EscapeString(stateLabel), html.EscapeString(when))
	return plain, emailShell("Tunnel Manager", "服务状态"+stateLabel, inner)
}
