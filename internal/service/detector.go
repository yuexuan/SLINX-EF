package service

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type Detector struct {
	client     *http.Client
	noRedirect *http.Client
	nodeID     uint
	nodeTag    string
	started    time.Time
}

func NewDetector(client *http.Client, id uint, tag string) *Detector {
	noRedirect := *client
	noRedirect.CheckRedirect = func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
	return &Detector{client: client, noRedirect: &noRedirect, nodeID: id, nodeTag: tag, started: time.Now()}
}

func DirectClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second, Transport: &http.Transport{Proxy: nil}}
}

func (d *Detector) PublicIP() (string, error) {
	resp, err := d.client.Get("https://api.ipify.org")
	if err != nil {
		return "", fmt.Errorf("出口连接失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("IP 目标站限制 HTTP %d（不认定代理失效）", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 256))
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(b))
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("IP 目标未返回公网地址")
	}
	return ip, nil
}
