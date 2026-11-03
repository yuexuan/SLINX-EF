package sub

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/slinxlink/node/internal/database"
	"github.com/slinxlink/node/internal/util"
)

func shadowsocksCredentials(user database.User, inbound database.Inbound) (string, string) {
	method := inbound.ShadowsocksMethod
	return method, util.ShadowsocksClientPassword(method, inbound.ShadowsocksPassword, user.Token)
}

func shadowsocksURI(user database.User, host string, inbound database.Inbound) string {
	method, password := shadowsocksCredentials(user, inbound)
	credential := base64.RawURLEncoding.EncodeToString([]byte(method + ":" + password))
	address := net.JoinHostPort(host, strconv.Itoa(inbound.Port))
	return "ss://" + credential + "@" + address + "#" + url.PathEscape(inbound.Name)
}

func shadowsocksClash(user database.User, host string, inbound database.Inbound) string {
	method, password := shadowsocksCredentials(user, inbound)
	var sb strings.Builder
	fmt.Fprintf(&sb, "  - name: %q\n", inbound.Name)
	fmt.Fprintf(&sb, "    type: ss\n")
	fmt.Fprintf(&sb, "    server: %q\n", host)
	fmt.Fprintf(&sb, "    port: %d\n", inbound.Port)
	fmt.Fprintf(&sb, "    cipher: %s\n", method)
	fmt.Fprintf(&sb, "    password: %q\n", password)
	fmt.Fprintf(&sb, "    udp: true\n")
	return sb.String()
}

func shadowsocksSurge(user database.User, host string, inbound database.Inbound) string {
	method, password := shadowsocksCredentials(user, inbound)
	// Surge supports both AEAD and 2022 AES, but not 2022 ChaCha20.
	if method == "2022-blake3-chacha20-poly1305" {
		return ""
	}
	// Surge's comma-separated proxy format cannot represent these values.
	if strings.ContainsAny(inbound.Name, ",\r\n=") || strings.ContainsAny(password, ",\r\n") {
		return ""
	}
	return fmt.Sprintf("%s = ss, %s, %d, encrypt-method=%s, password=%s, udp-relay=true\n", inbound.Name, host, inbound.Port, method, password)
}

func shadowsocksSingBox(user database.User, host string, inbound database.Inbound) string {
	method, password := shadowsocksCredentials(user, inbound)
	value := map[string]any{
		"type":        "shadowsocks",
		"tag":         "proxy",
		"server":      host,
		"server_port": inbound.Port,
		"method":      method,
		"password":    password,
	}
	b, _ := json.MarshalIndent(value, "        ", "    ")
	return string(b)
}
