package models

import (
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"strings"
)

type DNSRecordData struct {
	Priority *int    `json:"priority,omitempty"`
	Weight   *int    `json:"weight,omitempty"`
	Port     *int    `json:"port,omitempty"`
	Target   string  `json:"target,omitempty"`
	Flags    *int    `json:"flags,omitempty"`
	Tag      string  `json:"tag,omitempty"`
	Value    *string `json:"value,omitempty"`
	Service  string  `json:"service,omitempty"`
	Proto    string  `json:"proto,omitempty"`
	Name     string  `json:"name,omitempty"`
}

var dnsLabel = regexp.MustCompile(`^[A-Za-z0-9_](?:[A-Za-z0-9_-]*[A-Za-z0-9_])?$`)
var caaTag = regexp.MustCompile(`^[A-Za-z0-9]{1,15}$`)

func EditableDNSRecordType(recordType string) bool {
	switch recordType {
	case "A", "AAAA", "CNAME", "TXT", "MX", "NS", "SRV", "CAA", "PTR":
		return true
	}
	return false
}

func DNSProxyEligible(recordType string) bool {
	return recordType == "A" || recordType == "AAAA" || recordType == "CNAME"
}

func dnsHostname(value string) bool {
	value = strings.TrimSuffix(value, ".")
	if value == "" || len(value) > 253 {
		return false
	}
	if _, err := netip.ParseAddr(value); err == nil {
		return false
	}
	for _, label := range strings.Split(value, ".") {
		if len(label) > 63 || !dnsLabel.MatchString(label) {
			return false
		}
	}
	return true
}

func dnsUint16(value *int) bool {
	return value != nil && *value >= 0 && *value <= 65535
}

func ValidateDNSRecordRequest(payload *DNSRecordRequest) error {
	payload.Name = strings.TrimSuffix(strings.TrimSpace(payload.Name), ".")
	payload.Type = strings.ToUpper(strings.TrimSpace(payload.Type))
	if !EditableDNSRecordType(payload.Type) {
		return errors.New("unsupported DNS record type (read-only)")
	}
	name := strings.TrimPrefix(payload.Name, "*.")
	if payload.Name != "@" && payload.Name != "*" && !dnsHostname(name) {
		return errors.New("name must be a DNS name, @ or wildcard (use Punycode for international names)")
	}
	if payload.TTL != 1 && (payload.TTL < 60 || payload.TTL > 86400) {
		return errors.New("ttl must be automatic (1) or between 60 and 86400 seconds")
	}
	if !DNSProxyEligible(payload.Type) {
		payload.Proxied = false
	}
	if payload.Type != "MX" {
		payload.Priority = nil
	}
	if payload.Type == "SRV" || payload.Type == "CAA" {
		if payload.Data == nil {
			return fmt.Errorf("%s requires structured data", payload.Type)
		}
		if payload.Content != "" {
			return errors.New("SRV/CAA use data instead of content")
		}
		data := payload.Data
		if payload.Type == "SRV" {
			if data.Flags != nil || data.Tag != "" || data.Value != nil {
				return errors.New("CAA fields are not valid for SRV")
			}
			if !dnsUint16(data.Priority) || !dnsUint16(data.Weight) || !dnsUint16(data.Port) {
				return errors.New("SRV data.priority, weight and port must be integers between 0 and 65535")
			}
			data.Target = strings.TrimSpace(data.Target)
			if data.Target != "." && !dnsHostname(data.Target) {
				return errors.New("SRV data.target must be a hostname or .")
			}
			labels := strings.SplitN(payload.Name, ".", 3)
			if len(labels) != 3 || !strings.HasPrefix(labels[0], "_") || len(labels[0]) < 2 || !strings.HasPrefix(labels[1], "_") || len(labels[1]) < 2 {
				return errors.New("SRV name must include _service._protocol.name")
			}
			if data.Service != "" {
				data.Service = labels[0]
			}
			if data.Proto != "" {
				data.Proto = labels[1]
			}
			if data.Name != "" {
				data.Name = labels[2]
			}
		} else {
			if data.Priority != nil || data.Weight != nil || data.Port != nil || data.Target != "" || data.Name != "" || data.Service != "" || data.Proto != "" {
				return errors.New("SRV fields are not valid for CAA")
			}
			if data.Flags == nil || *data.Flags < 0 || *data.Flags > 255 || !caaTag.MatchString(data.Tag) || data.Value == nil || len(*data.Value) > 4096 || strings.ContainsAny(*data.Value, "\x00\r\n") {
				return errors.New("CAA requires data.flags (0-255), tag (1-15 letters/digits) and value (up to 4096 bytes)")
			}
		}
		return nil
	}
	if payload.Data != nil {
		return errors.New("data is only supported for SRV/CAA")
	}
	if payload.Type != "TXT" {
		payload.Content = strings.TrimSpace(payload.Content)
	}
	if strings.TrimSpace(payload.Content) == "" || len(payload.Content) > 4096 || strings.ContainsRune(payload.Content, '\x00') {
		return errors.New("content is required and must not exceed 4096 bytes")
	}
	switch payload.Type {
	case "A", "AAAA":
		address, err := netip.ParseAddr(payload.Content)
		if err != nil || address.Zone() != "" || (payload.Type == "A" && !address.Is4()) || (payload.Type == "AAAA" && !address.Is6()) {
			return fmt.Errorf("invalid %s address", payload.Type)
		}
		payload.Content = address.String()
	case "CNAME", "MX", "NS", "PTR":
		if !(payload.Type == "MX" && payload.Content == ".") && !dnsHostname(payload.Content) {
			return fmt.Errorf("%s content must be a hostname", payload.Type)
		}
		if payload.Type == "MX" && (!dnsUint16(payload.Priority) || (payload.Content == "." && *payload.Priority != 0)) {
			return errors.New("MX priority must be between 0 and 65535 (null MX requires 0)")
		}
	}
	return nil
}
