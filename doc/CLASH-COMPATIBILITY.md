# Clash 传统版、Mihomo 与 Shadowsocks 订阅兼容

SLINX-EF 在**导出阶段**按客户端内核生成配置。传统 Clash 和 Mihomo 共用已存在的 sing-box 服务端、入站和本地用户，不改变已有出口路由。

## 添加入站与用户绑定

1. 在「入站」页面，选择 **添加入站 → Shadowsocks**。版本选择 **普通 SS**（传统 Clash）或 **SS2022**（Mihomo / sing-box）。
2. 普通 SS 可选 `aes-128-gcm`、`aes-256-gcm`、`chacha20-ietf-poly1305`、`xchacha20-ietf-poly1305`；传统 Clash 兼容导出只使用前三种。SS2022 可选 `2022-blake3-aes-128-gcm`、`2022-blake3-aes-256-gcm`，由后台生成严格长度的 Base64 密钥。
3. **保存入站后**在「用户」中关联该入站。没有关联启用的本地用户时，sing-box **不监听**该 SS 入站，以免意外成为共享密钥节点。
4. 服务器和云侧防火墙开放入站端口的 TCP/UDP。客户端复制该用户的对应订阅地址，按客户端习惯手动切换节点。
5. 用户凭据由「入站服务端密钥 + 用户 Token」经 HMAC-SHA256 稳定派生：普通 SS 客户端只拿用户密码；SS2022 客户端拿 `serverKey:userKey` 组合密码。**不会覆盖现有 User.Password**（Trojan/AnyTLS/Hysteria 的认证密码）；修改服务端密钥或用户 Token 会使旧 SS 订阅密码失效，需要重新订阅。

## 其他传统入站

「传统 Clash 入站」快捷入口生成 VMess / RAW(TCP) / 不启用 TLS；如需使用 TLS，请配置正确证书。普通 Shadowsocks 入站也兼容旧版 Clash for Windows。

## 订阅导出

| 订阅类型 | 地址后缀 | 包含的节点 |
| --- | --- | --- |
| 传统 Clash | `/clash-legacy` | 普通 SS（经典 AEAD）、VMess、Trojan；不包含 SS2022/Reality 等现代协议，不包含 GEOSITE 规则 |
| Mihomo 现代 Clash | `/clash` | 普通 SS、SS2022 AES、VLESS/Reality、Hysteria2、TUIC 等 |
| 原始订阅 | 无后缀 | `ss://` SIP002 与现有 URI |
| Surge | `/surge` | 普通 SS、SS2022 AES、原有兼容的协议 |
| sing-box | 用户「复制链接」旁的单节点配置下载 | 包含与服务器一致的 SS 密码 |

如果用户没有与目标客户端兼容的入站，传统 Clash 订阅返回明确提示，不会通过改写协议类型生成无效节点。

### 固定核心版本兼容性

使用本地保存的 **sing-box 1.13.12** Windows 核心进行 `sing-box check`：

- 普通 SS：AES-128-GCM、AES-256-GCM、ChaCha20-IETF-Poly1305、XChaCha20-IETF-Poly1305 **均通过**。
- SS2022：`2022-blake3-aes-128-gcm`、`2022-blake3-aes-256-gcm` **通过**。
- `2022-blake3-chacha20-poly1305` 的**多用户入站**返回 `initialize inbound: invalid argument`，故此处**不开放**该入站算法；已有出站节点支持不受影响。后续升级核心时可重新验证。
- 当前 SS 入站使用本地用户；**SSPanel 外部同步用户**尚未接入 SS 凭据格式，不应将其绑定到 SS 入站。本地用户实现可使用不同协议的同一用户关系。

VLESS + Reality 的 `xtls-rprx-vision` 必须搭配 RAW/TCP，不能搭配 WebSocket。旧版 Clash for Windows 的原版 Clash 核心不能直接连接 VLESS + Reality。

上述为本地源码与配置层的校验；真实远端服务器端口、客户端握手与 UDP 转发需在部署后验证。
