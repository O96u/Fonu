package ddns

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Credentials struct {
	Provider string `json:"provider,omitempty"`
	Token    string `json:"token,omitempty"`
	TokenID  string `json:"token_id,omitempty"`
	Secret   string `json:"secret,omitempty"`
}

func CredentialsFromSave(in SaveInput) Credentials {
	provider := strings.TrimSpace(in.Provider)
	if provider == "" {
		provider = "cloudflare"
	}
	switch provider {
	case "dnspod":
		return Credentials{Provider: "dnspod", TokenID: strings.TrimSpace(in.APITokenID), Token: strings.TrimSpace(in.APIToken)}
	case "alidns":
		return Credentials{Provider: "alidns", Token: strings.TrimSpace(in.APIToken), Secret: strings.TrimSpace(in.APISecret)}
	case "tencentcloud":
		return Credentials{Provider: "tencentcloud", Token: strings.TrimSpace(in.APIToken), Secret: strings.TrimSpace(in.APISecret)}
	default:
		return Credentials{Provider: "cloudflare", Token: strings.TrimSpace(in.APIToken)}
	}
}

func (c Credentials) HasValues() bool {
	switch c.Provider {
	case "dnspod":
		return c.TokenID != "" && c.Token != ""
	case "alidns", "tencentcloud":
		return c.Token != "" && c.Secret != ""
	default:
		return c.Token != ""
	}
}

func (c Credentials) Validate(provider string) error {
	switch provider {
	case "dnspod":
		if c.TokenID == "" || c.Token == "" {
			return fmt.Errorf("请填写 DNSPod ID 和 Token")
		}
	case "alidns":
		if c.Token == "" || c.Secret == "" {
			return fmt.Errorf("请填写阿里云 AccessKey ID 和 Secret")
		}
	case "tencentcloud":
		if c.Token == "" || c.Secret == "" {
			return fmt.Errorf("请填写腾讯云 SecretId 和 SecretKey")
		}
	default:
		if c.Token == "" {
			return fmt.Errorf("请填写 API Token")
		}
	}
	return nil
}

func (c Credentials) LoginToken() string {
	return c.TokenID + "," + c.Token
}

func (c Credentials) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

func ParseCredentials(raw string) (Credentials, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Credentials{}, fmt.Errorf("凭证未配置")
	}
	if strings.HasPrefix(raw, "{") {
		var cred Credentials
		if err := json.Unmarshal([]byte(raw), &cred); err != nil {
			return Credentials{}, err
		}
		if cred.Provider == "" {
			cred.Provider = "cloudflare"
		}
		return cred, nil
	}
	return Credentials{Provider: "cloudflare", Token: raw}, nil
}
