package core

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/slinxlink/node/internal/database"
	"github.com/slinxlink/node/internal/service"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/proxy"
)

// This optional test uses an existing reachable gateway and real HTTPS public-IP requests.
// The SSH fixture is an actual SSH server, not a simulated HTTP response.
func TestLiveManagedEgress(t *testing.T) {
	gateway := os.Getenv("EF_LIVE_PROXY_ADDR")
	if gateway == "" {
		t.Skip("EF_LIVE_PROXY_ADDR not configured; real provider verification not claimed")
	}
	old := Default.BinPath
	Default.BinPath = os.Getenv("EF_CORE_PATH")
	defer func() { Default.BinPath = old }()
	host, portString, err := net.SplitHostPort(gateway)
	if err != nil {
		t.Fatal(err)
	}
	port, _ := strconv.Atoi(portString)
	var publicIP string
	check := func(nodes []database.Endpoint, tag string) {
		t.Helper()
		cfg, err := NodeConfig(nodes, tag)
		if err != nil {
			t.Fatal(err)
		}
		if err = CheckConfig(cfg); err != nil {
			t.Fatal(err)
		}
		client, close, err := ProbeClient(cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer close()
		ip, err := service.NewDetector(client, 1, tag).PublicIP()
		if err != nil {
			t.Fatal(err)
		}
		if publicIP != "" && ip != publicIP {
			t.Fatalf("exit mismatch: %s != %s", ip, publicIP)
		}
		publicIP = ip
		t.Logf("REAL HTTPS egress via %s: %s", tag, ip)
	}
	for _, typ := range []string{"http", "socks"} {
		check([]database.Endpoint{{Enable: true, Tag: typ, Type: typ, Server: host, ServerPort: port}}, typ)
	}
	_, hostPrivate, _ := ed25519.GenerateKey(rand.Reader)
	hostSigner, _ := ssh.NewSignerFromKey(hostPrivate)
	userPublic, userPrivate, _ := ed25519.GenerateKey(rand.Reader)
	upstream, err := proxy.SOCKS5("tcp", gateway, nil, proxy.Direct)
	if err != nil {
		t.Fatal(err)
	}
	server := &ssh.ServerConfig{
		PasswordCallback: func(meta ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if meta.User() == "fixture" && string(password) == "fixture-password" {
				return nil, nil
			}
			return nil, fmt.Errorf("authentication failed")
		},
		PublicKeyCallback: func(meta ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			expected, _ := ssh.NewPublicKey(userPublic)
			if meta.User() == "fixture" && string(key.Marshal()) == string(expected.Marshal()) {
				return nil, nil
			}
			return nil, fmt.Errorf("key rejected")
		},
	}
	server.AddHostKey(hostSigner)
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	var forwarded atomic.Int32
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				session, channels, requests, err := ssh.NewServerConn(conn, server)
				if err != nil {
					conn.Close()
					return
				}
				defer session.Close()
				go ssh.DiscardRequests(requests)
				for incoming := range channels {
					if incoming.ChannelType() != "direct-tcpip" {
						incoming.Reject(ssh.UnknownChannelType, "unsupported")
						continue
					}
					var destination struct {
						Host       string
						Port       uint32
						OriginHost string
						OriginPort uint32
					}
					if ssh.Unmarshal(incoming.ExtraData(), &destination) != nil {
						incoming.Reject(ssh.ConnectionFailed, "invalid")
						continue
					}
					remote, err := upstream.Dial("tcp", net.JoinHostPort(destination.Host, strconv.Itoa(int(destination.Port))))
					if err != nil {
						incoming.Reject(ssh.ConnectionFailed, "dial failed")
						continue
					}
					channel, reqs, err := incoming.Accept()
					if err != nil {
						remote.Close()
						continue
					}
					forwarded.Add(1)
					go ssh.DiscardRequests(reqs)
					go func() {
						defer channel.Close()
						defer remote.Close()
						go io.Copy(channel, remote)
						io.Copy(remote, channel)
					}()
				}
			}()
		}
	}()
	sshPort := listener.Addr().(*net.TCPAddr).Port
	hostKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(hostSigner.PublicKey())))
	ep := database.Endpoint{Enable: true, Tag: "ssh", Type: "ssh", Server: "127.0.0.1", ServerPort: sshPort, Username: "fixture", Password: "fixture-password"}
	options := map[string]any{"host_key": []string{hostKey}}
	setOptions := func() { b, _ := json.Marshal(options); ep.Options = string(b) }
	setOptions()
	check([]database.Endpoint{ep}, "ssh")
	keyDER, _ := x509.MarshalPKCS8PrivateKey(userPrivate)
	ep.Password = ""
	options["private_key"] = string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}))
	setOptions()
	check([]database.Endpoint{ep}, "ssh")
	encrypted, err := ssh.MarshalPrivateKeyWithPassphrase(userPrivate, "test", []byte("passphrase"))
	if err != nil {
		t.Fatal(err)
	}
	options["private_key"] = string(pem.EncodeToMemory(encrypted))
	options["private_key_passphrase"] = "passphrase"
	setOptions()
	check([]database.Endpoint{ep}, "ssh")
	// Add an actual local HTTP relay in front of SSH, then verify full chain export/reimport.
	relayClient, relayClose, err := ProbeClient(map[string]any{"outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}}, "route": map[string]any{"final": "direct"}})
	if err != nil {
		t.Fatal(err)
	}
	defer relayClose()
	relayURL, _ := relayClient.Transport.(*http.Transport).Proxy(&http.Request{})
	relayPort, _ := strconv.Atoi(relayURL.Port())
	relay := database.Endpoint{Enable: true, Tag: "relay", Type: "http", Server: relayURL.Hostname(), ServerPort: relayPort}
	ep.Detour = "relay"
	check([]database.Endpoint{relay, ep}, "ssh")
	bundle, err := ExportNodes([]database.Endpoint{relay, ep}, []string{"ssh"}, "sing-box")
	if err != nil {
		t.Fatal(err)
	}
	imported, err := ImportNodes(bundle)
	if err != nil {
		t.Fatal(err)
	}
	check(imported, "ssh")
	if forwarded.Load() < 5 {
		t.Fatal("requests did not traverse SSH fixture")
	}
	relay.Enable = false
	if _, err := NodeConfig([]database.Endpoint{relay, ep}, "ssh"); err == nil {
		t.Fatal("disabled relay did not block")
	}
	// Explicit wrong host pin must fail; never return the default network's public IP.
	ep.Detour = ""
	_, badPrivate, _ := ed25519.GenerateKey(rand.Reader)
	badSigner, _ := ssh.NewSignerFromKey(badPrivate)
	options["host_key"] = []string{strings.TrimSpace(string(ssh.MarshalAuthorizedKey(badSigner.PublicKey())))}
	setOptions()
	cfg, _ := NodeConfig([]database.Endpoint{ep}, "ssh")
	client, close, err := ProbeClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer close()
	if ip, err := service.NewDetector(client, 1, "ssh").PublicIP(); err == nil {
		t.Fatalf("bad host pin unexpectedly connected: %s", ip)
	}
	t.Log("wrong SSH host pin: connection failed without direct fallback")
}
