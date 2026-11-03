package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// ProbeClient starts an isolated core with only the selected chain and a loopback HTTP inbound.
// No service-wide route or default HTTP client is modified.
func ProbeClient(cfg map[string]any) (*http.Client, func(), error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, nil, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	cfg["inbounds"] = []any{map[string]any{"type": "http", "tag": "probe", "listen": "127.0.0.1", "listen_port": port}}
	cfg["log"] = map[string]any{"disabled": true}
	dir, err := os.MkdirTemp("", "slinx-probe-")
	if err != nil {
		return nil, nil, err
	}
	path := dir + "/config.json"
	data, _ := json.Marshal(cfg)
	if err = os.WriteFile(path, data, 0600); err != nil {
		os.RemoveAll(dir)
		return nil, nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, Default.BinPath, "run", "-c", path)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err = cmd.Start(); err != nil {
		cancel()
		os.RemoveAll(dir)
		return nil, nil, err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	addr := "127.0.0.1:" + strconv.Itoa(port)
	proxy, _ := url.Parse("http://" + addr)
	transport := &http.Transport{Proxy: http.ProxyURL(proxy), ForceAttemptHTTP2: true, ResponseHeaderTimeout: 15 * time.Second}
	cleanup := func() { transport.CloseIdleConnections(); cancel(); <-done; os.RemoveAll(dir) }
	for i := 0; i < 100; i++ {
		select {
		case <-done:
			cancel()
			os.RemoveAll(dir)
			return nil, nil, fmt.Errorf("检测核心启动失败")
		default:
		}
		conn, e := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if e == nil {
			conn.Close()
			return &http.Client{Transport: transport, Timeout: 20 * time.Second}, cleanup, nil
		}
		time.Sleep(30 * time.Millisecond)
	}
	cleanup()
	return nil, nil, fmt.Errorf("检测入口启动超时")
}
