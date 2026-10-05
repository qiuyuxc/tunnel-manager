package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/idna"
	"golang.org/x/net/publicsuffix"
	"tunnel-manager/models"
)

type LabTXTResolver interface {
	LookupTXT(context.Context, string) ([]string, error)
}

func NormalizeLabHostname(value string) (string, error) {
	value = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ".")
	host, err := idna.Lookup.ToASCII(value)
	if err != nil || len(host) > 253 || net.ParseIP(host) != nil || !strings.Contains(host, ".") {
		return "", errors.New("Host/SNI 必须是完整域名，不能填写 IP、端口或 URL")
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", errors.New("域名格式不正确")
		}
		for _, character := range label {
			if !(character >= 'a' && character <= 'z' || character >= '0' && character <= '9' || character == '-') {
				return "", errors.New("域名含有不允许的字符")
			}
		}
	}
	if _, err := publicsuffix.EffectiveTLDPlusOne(host); err != nil {
		return "", errors.New("不能使用公共后缀作为探测域名")
	}
	return host, nil
}

func LabVerificationDomain(requested string) (string, error) {
	if strings.TrimSpace(requested) == "" {
		return "", errors.New("请填写你管理的验证域名，验证域名与探测 Host/SNI 独立")
	}
	domain, err := NormalizeLabHostname(requested)
	if err != nil {
		return "", fmt.Errorf("验证域名无效：%w", err)
	}
	return domain, nil
}

func labDomainContains(domain, host string) bool {
	return host == domain || strings.HasSuffix(host, "."+domain)
}

func NormalizeLabProbe(settings *models.LabIPSelectorSettings) error {
	host, err := NormalizeLabHostname(settings.Host)
	if err != nil {
		return err
	}
	sni := settings.SNI
	if strings.TrimSpace(sni) == "" {
		sni = host
	}
	sni, err = NormalizeLabHostname(sni)
	if err != nil {
		return err
	}
	path := strings.TrimSpace(settings.Path)
	if path == "" {
		path = "/"
	}
	parsed, err := url.ParseRequestURI(path)
	if err != nil || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || parsed.Host != "" {
		return errors.New("探测路径必须是站内路径，不能包含完整 URL")
	}
	for _, character := range path {
		if character <= 32 || character == 127 {
			return errors.New("探测路径不能含空格或控制字符")
		}
	}
	settings.Host, settings.SNI, settings.Path = host, sni, path
	return nil
}

func ValidateLabPublicTargets(targets []string) error {
	for _, target := range targets {
		address, err := netip.ParseAddr(target)
		if err != nil || !labPublicAddress(address.Unmap()) {
			return fmt.Errorf("禁止探测本机、私网或保留地址：%s", target)
		}
	}
	return nil
}

var labBlockedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"), netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"), netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("3fff::/20"),
}

func labPublicAddress(address netip.Addr) bool {
	if !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast() {
		return false
	}
	if address.Is6() && !netip.MustParsePrefix("2000::/3").Contains(address) {
		return false
	}
	for _, prefix := range labBlockedPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

func newLabToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func validateLabVerification(verification models.LabDomainVerification, ownerID string) error {
	if ownerID == "" || verification.OwnerID != ownerID || verification.Token == "" || verification.Domain == "" || verification.VerifiedAt == 0 {
		return errors.New("请先使用当前管理员账号完成 DNS TXT 域名验证")
	}
	domain, err := LabVerificationDomain(verification.Domain)
	if err != nil || domain != verification.Domain {
		return errors.New("验证域名无效，请重新生成 TXT")
	}
	if verification.RecordName != "_tunnel-manager-verify."+verification.Domain {
		return errors.New("TXT 验证记录不匹配，请重新生成")
	}
	return nil
}

func lookupLabVerification(ctx context.Context, resolver LabTXTResolver, verification models.LabDomainVerification) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	records, err := resolver.LookupTXT(ctx, verification.RecordName+".")
	if err != nil {
		return errors.New("DNS TXT 查询失败，已停止任务；请检查解析后重试")
	}
	for _, record := range records {
		if record == verification.Token {
			return nil
		}
	}
	return errors.New("DNS TXT 验证值未找到或已移除，已停止任务")
}
