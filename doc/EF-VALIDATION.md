# SLINX-EF 实际交付验收（2026-11-03）

## 仓库、发布与来源

- Fork: https://github.com/yuexuan/SLINX-EF ，本地目录 `G:\Project\SLINX-EF`。
- 保留上游历史、MIT LICENSE、Go 模块路径、表名、`/etc/slinx` 布局及 `slinx` 管理命令。
- upstream: https://github.com/slinxlink/node ，基线提交 `6f68b1849d60f1a96aa0ef36c4ca02924cb40435`。
- 首个成功 Release: https://github.com/yuexuan/SLINX-EF/releases/tag/v0.1.0-ef.2
- 首次成功 Actions: https://github.com/yuexuan/SLINX-EF/actions/runs/41582849934
- 当前 Release: https://github.com/yuexuan/SLINX-EF/releases/tag/v0.1.0-ef.3
- 当前成功 Actions: https://github.com/yuexuan/SLINX-EF/actions/runs/41584128961
- 当前版本源码提交：`8d7361eaaf715cba28794df41cc5ef89de1e16c0`。
- 当前 AMD64 二进制 SHA256：`e72d3ab73cfbf89d2a08ab447117118f6f71af1a05cbd5ac4f67a010908ee64f`。
- ARM64 二进制 SHA256：`a0e636c8cdd076ee9746e5a1f50d274a96740aade57acd251be5b1dc8cf5a831`。
- Actions 验证实际 AMD64 二进制 `version` 输出与标签、Fork 提交一致；服务器再次核对版本与 Release 哈希。
- Release 包含双架构二进制、SHA256SUMS、BUILDINFO、install.sh，仅使用 GITHUB_TOKEN 发布。
- 早期失败构建未发布不完整 Release；`ef.1` 没有公开 Release。

## 核心与构建

- 复用上游 Dockerfile.builder、Makefile、CGO 交叉编译及前端嵌入方式。
- 前端类型检查、Vite 构建、Go 测试、双架构编译和版本来源检查通过。
- 核心固定使用上游 `sing-box-1.13.12`；压缩包的两种架构 SHA256 固定在安装脚本中。
- CI 使用固定核心实际检查生成的 SOCKS5 / HTTP 配置及禁用、删除时的阻断路由。
- AMD64 已实际部署运行；ARM64 已编译，未在 ARM64 实机运行。

## VPS 与真实连接

- 服务器：38.55.195.102，Debian 13.7，AMD64。
- 安装前未发现 SLINX、sing-box 或 Xray 安装；现有 SSH、iperf3 与系统服务保留。
- 用公开 raw URL 的正式安装脚本安装 ef.2，未使用手工编译面板替代。
- 面板、固定核心下载均通过 SHA256 校验，systemd active/running，核心配置检查通过。
- 创建真实 VLESS + Reality 入站、测试用户，订阅按入站名称导出独立节点。
- 本机官方 sing-box 1.13.12 客户端经公网连接 443 节点，HTTPS 查询公网出口：`38.55.195.102`。
- 已验证握手目标：`www.cloudflare.com`。`www.microsoft.com` 在本次环境中握手失败，未将其作为交付配置。
- 第二个 8443 入站在 VPS 内可真实连接，但本机到其公网端口超时；服务器未发现 INPUT 防火墙阻断，公网链路原因尚未定位。该测试入站已停用，交付订阅仅包含公网实测通过的 443 节点。
- 管理账号、Reality 私钥、用户 UUID、订阅 token、部署私钥均未写入仓库；管理访问可使用 SSH 隧道。

## ISP 行为

- 使用现有 Endpoint 表单与绑定组件实现 SOCKS5、HTTP CONNECT，保留 WireGuard / WARP 路径。
- 创建、编辑持久化、绑定、配置重载、启停与删除均已操作验证。
- 两个真实 Reality 入站绑定同一 SOCKS5 出口：两个客户端均因出口不可达而失败；显式解除绑定后均恢复直连。
- SOCKS5 与 HTTP CONNECT 各自验证：不可达时请求失败；禁用保留绑定并生成 block；重启用恢复出口引用；删除将路由目标改为 block；均没有自动回落 Direct。
- 检查真实订阅：保留各入站名称，未出现测试代理用户名或密码。
- 未提供真实 ISP 账户：实际 ISP 公网出口 IP 尚未验证。不可达代理用于故障阻断测试，不作为可用 ISP 的证明。
- Direct 已实测；WireGuard / WARP 代码路径保留，未进行真实隧道或 Cloudflare 注册流量测试。

## 更新、备份与恢复

- 使用正式安装器从 ef.2 尝试升级 ef.3，故意让面板健康探测失败。
- 安装器返回非零，自动恢复 ef.2 的二进制及配置；恢复后 SHA256 与 ef.2 Release 一致，systemd active，用户、入站、出口及路由快照一致。
- 该回滚备份：`/etc/slinx/backups/20261103T053455Z-113687`。
- 随后调用面板内置更新接口，成功从自有 Releases 升级 ef.2 → ef.3；配置快照一致，真实 Reality 连接及出口 IP 仍正常。
- 成功升级备份：`/etc/slinx/backups/20261103T053537Z-113946`。
- 发现并移除上游全局 `pkill -f sing-box`。ef.3 实测受管核心重启前后，独立 sing-box 测试客户端 PID 均为 114157，连接保持可用。
- 进程隔离及流量测试所用临时 VPS 客户端服务已停止；故障注入 curl 仅通过单次命令 PATH 生效，不影响正常更新环境。

## 安装命令

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/yuexuan/SLINX-EF/main/install.sh)
```

管理：`slinx`；状态：`systemctl status slinx`；日志：`journalctl -u slinx`；版本：`/etc/slinx/slinx version`。

补充复验：再次执行公开一键脚本成功，版本保持 ef.3，配置快照一致，公网 Reality 请求仍返回 38.55.195.102。重复安装备份：`/etc/slinx/backups/20261103T053855Z-114340`；最终 systemd MainPID=114477，active/running。



## Active Node Extension 本轮验收（2026-11-03，发布前）

本节为本轮新增能力的证据；上方 ef.3 的历史验收不代表本轮新代码已经部署。

| 阶段 | 实现及可获得条件下验收 | 外部条件 |
| --- | --- | --- |
| P0 | PASS：Header 唯一主题入口；暗/亮切换、刷新、390px 窄屏 Chrome 实测 | 无 |
| P1 | PASS：SOCKS4/4a/5、HTTP/HTTPS、SSH、SS、VLESS、VMess、Trojan、Hysteria2、TUIC、AnyTLS、自定义受支持 JSON、WireGuard；固定 1.13.12 实际 check | SS/VLESS/VMess/Trojan/QUIC/AnyTLS/WireGuard 供应商真实连接 NOT VERIFIED |
| P2 | PASS：循环/缺失依赖拒绝、禁用阻断、删除中继禁用依赖；受控 HTTP→SSH 链路和导出再导入公网连接 | 真实 ISP 中继链缺少凭据，NOT VERIFIED |
| P3 | PASS：固定/会话/API 模式、模板、有效期、手动轮换、明确网关连接失败按需轮换；受控提供商运行验收 | 真实住宅/ISP 会话与 API 换 IP 缺少账号，NOT VERIFIED |
| P4 | PASS：实际节点出站 IP、ASN、地区与 16 项解锁结果关联；不可达不回退，节点历史隔离；MTR 保留服务器级 | 各供应商 IP 类型属于来源判断 |
| P5 | PASS：sing-box / SLINX-EF 依赖导出导入、Mihomo 真客户端检查、常见 URI、旧订阅隔离 | Surge 真客户端 NOT VERIFIED，当前 Windows 无对应客户端 |
| P6 | 前端构建/类型检查、Go、核心矩阵与真实网络已通过；最终 Release/部署待下方发布记录 | 供应商缺项如上 |

### 命令与真实结果

- `npm run build --prefix web`：类型检查和 Vite 构建 PASS。
- 本机 `go test -tags nomsgpack ./...`（CGO + GCC，EF_CORE_PATH 指向本机完整核心）：PASS。API 生命周期、失败阻断、循环、原子导入及订阅凭据隔离纳入测试。
- Windows 测试辅助核心由官方 `github.com/sagernet/sing-box/cmd/sing-box@v1.13.12` 构建，`go version -m` 确认模块版本与校验和；启用 v2ray_api / gvisor / quic / wireguard / utls / clash_api。正式服务器仍使用既有上游固定 Linux 1.13.12 包，未替换核心版本。
- 官方 Windows 1.13.12 标准包缺少 v2ray_api，不能用于检查面板完整配置；初次完整检查明确失败后改用上述同版本完整构建。WireGuard 还需 with_gvisor，已补齐并复验 PASS。
- `TestNativeProtocolMatrix`：14 组实际核心检查 PASS；`TestWireGuardNativeRoundtrip`：原生 endpoint（含 keepalive）检查/导入导出 PASS。
- Mihomo v1.19.33：`-t -d <temp> -f config.yaml` 对支持协议实际解析 PASS（含 SSH、AnyTLS、WireGuard）。SOCKS4/4a 及不能无损映射的字段明确拒绝。
- `TestLiveManagedEgress`：静态 HTTP/SOCKS5、真实受控 SSH 服务密码/私钥/加密私钥、HTTP→SSH 链路、完整 sing-box 导出再导入均通过 HTTPS 请求得到 `155.254.105.7`；错误 SSH Host Key 请求失败，没有 Direct 回退。受控 SSH 是实际 SSH 服务与公网请求，不是供应商 SSH 账号验收。
- `TestLiveRemoteSSH`：连接已配置 VPS 的真实 SSH，私钥 + Host Key + ssh-ed25519 算法钉扎后，HTTPS 公网 IP 为 `38.55.195.102`，PASS。初次只钉扎 ed25519 Host Key、未限制协商算法时服务器选择 ECDSA，正确报 Host Key mismatch；未通过跳过验证绕开，已明确配置匹配算法后通过。
- 本地完整运行面板的节点检测：`QA-HTTP / NodeID=1 / 155.254.105.7 / AS38136 / HK / Hong Kong / ippure.com`，首次耗时 1086ms。16 项检测均关联 NodeID=1、IP=155.254.105.7；该结果不是 VPS 的 38.55.195.102。
- 实测结果：Netflix、Disney、Prime、YouTube Premium、DAZN、iQiyi、Claude、ChatGPT 等探测返回可用；Gemini、Steam、TVBAnywhere、Wikipedia 返回不可用，Bing 受限。仅记录本次现有探测器返回值，不把所有平台写为 PASS。
- QA-Unreachable（127.0.0.1:9）：IP 接口返回 500、无 data，节点历史中没有服务器默认 IP，PASS。
- QA-Session：会话 ID 轮换、有效期、连接刷新、前后实际 IP 查询 PASS；前后都是 155.254.105.7，正确显示“实际 IP 未变化”。
- QA-API：受控本地 HTTP API 返回 JSON 网关映射后应用并查询实际出口，PASS；前后 IP 相同，不作为真实供应商换 IP 证明。

### 构建中发现的真实问题及处理

- 初次 Actions 41591280074 因 AnyTLS 扩展会话字段不能放入通用 URI 的测试预期错误失败；保留导出拒绝语义，修正测试为基础兼容字段后通过，未生成假 URI。
- Actions 41592461206：双架构构建、固定上游核心校验与 Go 测试成功（源码 d58280b），该次为无标签构建，不是最终 Release。
- 早期在 1GB VPS 上的验证构建导致资源紧张及 `/tmp` 链接空间不足，没有用于替换线上二进制。随后转移本机/CI。SSH 在约 14:27–15:04 期间握手超时，本机和 GitHub runner 都失败；恢复后确认无构建进程，slinx ef.3、sing-box 与 SSH 服务正常。未重启主机或结束无关服务。
- 临时 GitHub SSH 检查工作流与临时仓库密钥仅用于排查，结束后清理；不保留新的部署授权体系。
