package service

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/slinxlink/node/internal/database"
)

type DynamicPolicy struct {
	Mode             string            `json:"mode"`
	SessionID        string            `json:"session_id"`
	TTL              int               `json:"ttl_seconds"`
	ExpiresAt        time.Time         `json:"expires_at"`
	UsernameTemplate string            `json:"username_template"`
	PasswordTemplate string            `json:"password_template"`
	URL              string            `json:"url"`
	Method           string            `json:"method"`
	Headers          map[string]string `json:"headers"`
	Body             string            `json:"body"`
	Mapping          map[string]string `json:"mapping"`
	RotateOnFailure  bool              `json:"rotate_on_failure"`
}

func ParseDynamic(ep database.Endpoint) (DynamicPolicy, error) {
	var p DynamicPolicy
	if ep.Dynamic == "" {
		return p, nil
	}
	dec := json.NewDecoder(strings.NewReader(ep.Dynamic))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return p, fmt.Errorf("动态配置: %w", err)
	}
	if p.Mode == "" || p.Mode == "fixed" {
		return p, nil
	}
	if ep.Type != "socks" && ep.Type != "http" {
		return p, fmt.Errorf("动态网关仅适用于 SOCKS / HTTP")
	}
	if p.Mode != "session" && p.Mode != "api" {
		return p, fmt.Errorf("动态模式必须为 fixed/session/api")
	}
	if p.TTL < 0 {
		return p, fmt.Errorf("会话有效期不可为负")
	}
	if p.Mode == "session" && p.UsernameTemplate == "" && p.PasswordTemplate == "" {
		return p, fmt.Errorf("会话模式需要认证模板，使用 {session} 占位符")
	}
	if p.Mode == "api" {
		u, err := url.Parse(p.URL)
		if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
			return p, fmt.Errorf("供应商 API URL 无效")
		}
		if p.Method != "" && p.Method != "GET" && p.Method != "POST" && p.Method != "PUT" {
			return p, fmt.Errorf("API 方法只支持 GET/POST/PUT")
		}
		for k := range p.Mapping {
			if k != "server" && k != "server_port" && k != "username" && k != "password" {
				return p, fmt.Errorf("不支持的 API 映射 %s", k)
			}
		}
	}
	return p, nil
}

func DynamicCredentials(ep database.Endpoint) (string, string, error) {
	p, err := ParseDynamic(ep)
	if err != nil {
		return "", "", err
	}
	user, pass := ep.Username, ep.Password
	if p.Mode == "session" {
		if p.SessionID == "" {
			return "", "", fmt.Errorf("会话尚未分配，请先轮换")
		}
		replace := func(t string) string {
			return strings.NewReplacer("{session}", p.SessionID, "{username}", ep.Username, "{password}", ep.Password).Replace(t)
		}
		if p.UsernameTemplate != "" {
			user = replace(p.UsernameTemplate)
		}
		if p.PasswordTemplate != "" {
			pass = replace(p.PasswordTemplate)
		}
	}
	return user, pass, nil
}

// RotateDynamic only allocates credentials. Application and real egress checks belong to the caller.
func RotateDynamic(ep database.Endpoint) (database.Endpoint, error) {
	p, err := ParseDynamic(ep)
	if err != nil {
		return ep, err
	}
	if p.Mode == "session" {
		b := make([]byte, 12)
		if _, err := rand.Read(b); err != nil {
			return ep, err
		}
		p.SessionID = hex.EncodeToString(b)
		if p.TTL > 0 {
			p.ExpiresAt = time.Now().Add(time.Duration(p.TTL) * time.Second)
		}
	} else if p.Mode == "api" {
		method := p.Method
		if method == "" {
			method = "POST"
		}
		req, err := http.NewRequest(method, p.URL, bytes.NewBufferString(p.Body))
		if err != nil {
			return ep, fmt.Errorf("供应商请求无效")
		}
		for k, v := range p.Headers {
			req.Header.Set(k, v)
		}
		client := &http.Client{Timeout: 20 * time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(r *http.Request, v []*http.Request) error { return http.ErrUseLastResponse }}
		resp, err := client.Do(req)
		if err != nil {
			return ep, fmt.Errorf("供应商接口连接失败")
		}
		defer resp.Body.Close()
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return ep, fmt.Errorf("供应商接口认证失败 HTTP %d", resp.StatusCode)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return ep, fmt.Errorf("供应商接口错误 HTTP %d", resp.StatusCode)
		}
		var result map[string]any
		if err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
			return ep, fmt.Errorf("供应商返回非 JSON 对象")
		}
		for dest, path := range p.Mapping {
			var value any = result
			for _, part := range strings.Split(path, ".") {
				m, ok := value.(map[string]any)
				if !ok {
					return ep, fmt.Errorf("供应商映射路径不存在: %s", dest)
				}
				value = m[part]
			}
			if value == nil {
				return ep, fmt.Errorf("供应商映射缺少 %s", dest)
			}
			switch dest {
			case "server":
				ep.Server, _ = value.(string)
			case "server_port":
				n, ok := value.(float64)
				if !ok || n != float64(int(n)) {
					return ep, fmt.Errorf("供应商端口无效")
				}
				ep.ServerPort = int(n)
			case "username":
				ep.Username, _ = value.(string)
			case "password":
				ep.Password, _ = value.(string)
			}
		}
	} else {
		return ep, fmt.Errorf("固定网关无需轮换")
	}
	data, _ := json.Marshal(p)
	ep.Dynamic = string(data)
	return ep, nil
}
