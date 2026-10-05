package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"tunnel-manager/models"
)

func (runner *LabIPSelectorRunner) ProvisionDNS(ctx context.Context, domain, ownerID string, provider LabDNSProvider) (models.LabDNSProvisionResult, error) {
	var result models.LabDNSProvisionResult
	if !runner.ownerAllowed(ownerID) {
		return result, errors.New("请使用有效的管理员账号操作")
	}
	runner.cancelRun()
	runner.control.Lock()
	defer runner.control.Unlock()
	if !runner.ownerAllowed(ownerID) || !runner.store.GetAppSettings().ExperimentalFeatures {
		return result, errors.New("管理员身份已失效或实验性功能已关闭")
	}
	if provider == nil {
		return result, errors.New("当前账户未绑定可用的 DNS 账户，请手动添加 TXT")
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	domain, err := LabVerificationDomain(domain)
	if err != nil {
		return result, err
	}
	zone, err := provider.FindZone(ctx, domain)
	if err != nil {
		return result, err
	}
	if !runner.ownerAllowed(ownerID) || !runner.store.GetAppSettings().ExperimentalFeatures || ctx.Err() != nil {
		return result, errors.New("操作已取消，未写入 TXT")
	}
	security := runner.store.GetLabSecurity()
	verification := security.Verification
	check := verification
	check.VerifiedAt = 1
	reusable := security.InstanceID != "" && strings.HasPrefix(verification.Token, "tunnel-manager="+security.InstanceID+".") && verification.Domain == domain &&
		validateLabVerification(check, ownerID) == nil && (verification.VerifiedAt > 0 || (verification.IssuedAt > 0 && time.Now().Unix()-verification.IssuedAt <= 86400))
	if reusable {
		if err := runner.store.SuspendLab(); err != nil {
			return result, err
		}
	} else {
		verification, err = runner.newChallengeLocked(domain, ownerID)
		if err != nil {
			return result, err
		}
	}
	result.Verification = verification
	result.RecordCreated, err = provider.EnsureTXT(ctx, zone, verification.RecordName, verification.Token)
	if err != nil {
		return result, err
	}
	result.Verification, err = runner.verifyLocked(ctx, ownerID, true)
	result.Verified = err == nil
	result.Message = "TXT 已写入并通过 DNS 验证；定时任务仍关闭，不会自动开始扫描。"
	if err != nil {
		result.Message = "TXT 已提交到 Cloudflare，但尚未通过 DNS 检查：" + err.Error() + "。请等待解析生效后点击检查 TXT，不必重新生成记录。"
	}
	return result, nil
}
