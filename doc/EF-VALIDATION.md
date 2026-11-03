# SLINX-EF 实际交付验收（2026-11-03）

## 仓库、发布与来源

- Fork: https://github.com/yuexuan/SLINX-EF ，本地目录 `G:\Project\SLINX-EF`。
- 保留上游历史、MIT LICENSE、Go 模块路径、表名、`/etc/slinx` 布局及 `slinx` 管理命令。
- upstream: https://github.com/slinxlink/node ，基线提交 `6f68b1849d60f1a96aa0ef36c4ca02924cb40435`。
- 首个成功 Release: https://github.com/yuexuan/SLINX-EF/releases/tag/v0.1.0-ef.2
- 首次成功 Actions: https://github.com/yuexuan/SLINX-EF/actions/runs/41582849934
- 当前 Release: https://github.com/yuexuan/SLINX-EF/releases/tag/v0.1.0-ef.5
- 当前成功 Actions: https://github.com/yuexuan/SLINX-EF/actions/runs/41596087673
- 当前 Release 源码提交：`4af52732077336c71d6d9081306a72d6c836ac38`。
- 当前 AMD64 二进制 SHA256：`5932ab3e6ea9f3f5f6f77799318c7874f8f9203cb3bd65f8983fa720fd7ba204`。
- ARM64 二进制 SHA256：`84ce6266bef1503fee7cae4920207111cf87196a5c269db6ec97537cc7d661c0`。
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



## Active Node Extension 本轮验收（2026-11-03，已发布部署 ef.5）

本节及下方最终发布记录为本轮新增能力证据；前面的 ef.2/ef.3 章节保留历史，不替代本轮验收。

| 阶段 | 实现及可获得条件下验收 | 外部条件 |
| --- | --- | --- |
| P0 | PASS：Header 唯一主题入口；暗/亮切换、刷新、390px 窄屏 Chrome 实测 | 无 |
| P1 | PASS：SOCKS4/4a/5、HTTP/HTTPS、SSH、SS、VLESS、VMess、Trojan、Hysteria2、TUIC、AnyTLS、自定义受支持 JSON、WireGuard；固定 1.13.12 实际 check | SS/VLESS/VMess/Trojan/QUIC/AnyTLS/WireGuard 供应商真实连接 NOT VERIFIED |
| P2 | PASS：循环/缺失依赖拒绝、禁用阻断、删除中继禁用依赖；受控 HTTP→SSH 链路和导出再导入公网连接 | 真实 ISP 中继链缺少凭据，NOT VERIFIED |
| P3 | PASS：固定/会话/API 模式、模板、有效期、手动轮换、明确网关连接失败按需轮换；受控提供商运行验收 | 真实住宅/ISP 会话与 API 换 IP 缺少账号，NOT VERIFIED |
| P4 | PASS：实际节点出站 IP、ASN、地区与 16 项解锁结果关联；不可达不回退，节点历史隔离；MTR 保留服务器级 | 各供应商 IP 类型属于来源判断 |
| P5 | PASS：sing-box / SLINX-EF 依赖导出导入、Mihomo 真客户端检查、常见 URI、旧订阅隔离 | Surge 真客户端 NOT VERIFIED，当前 Windows 无对应客户端 |
| P6 | PASS：前端/Go/固定核心/双架构 Actions、正式 Release、服务器升级、真实阻断及故障回滚 | 供应商缺项如上；ARM64 未在实机运行 |

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


## 最终 Release、部署及回滚（ef.5）

- 正式版本：[v0.1.0-ef.5](https://github.com/yuexuan/SLINX-EF/releases/tag/v0.1.0-ef.5)。[Actions 41596087673](https://github.com/yuexuan/SLINX-EF/actions/runs/41596087673) 成功。
- BUILDINFO：`repository=yuexuan/SLINX-EF`，`commit=4af52732077336c71d6d9081306a72d6c836ac38`，`version=v0.1.0-ef.5`。后续证据/测试提交不改变此 Release 的二进制来源。
- 产物齐全：AMD64、ARM64、SHA256SUMS、BUILDINFO、install.sh。服务器安装后版本和 AMD64 SHA256 均与发布一致；核心仍为 1.13.12。
- ef.4（f072718）的生产验收发现旧库唯一索引阻止历史追加，具体错误为 `UNIQUE constraint failed: ip.source, ip.ip_version`。它已被 ef.5 替代，不作为最终可用版本。
- ef.5 启动迁移移除旧 IP/解锁唯一索引，建立节点历史普通索引；将旧记录 NULL NodeID 归为服务器直连，保留原记录。`TestUpgradeLegacyDetectionIndexesPreservesHistory` 验证迁移、重复执行、同节点重复与跨节点历史追加，PASS。
- ef.3→ef.4 故障注入：仅给该安装命令设置临时 PATH，让本机面板健康 curl 返回错误。安装器自动恢复 ef.3 及其数据，原 SHA256、配置与订阅检查通过。备份：`/etc/slinx/backups/20261103T071916Z-125173`。
- ef.4 首次成功升级备份：`/etc/slinx/backups/20261103T072044Z-125474`。
- ef.4→ef.5 再次故障注入：自动恢复 ef.4 二进制 `f4727ec4fee8fdaf0ba5734f05ab29404367eaab37be2b76bd5e47190a3fb34f` 和数据，服务与订阅复验通过。回滚备份：`/etc/slinx/backups/20261103T072814Z-126059`。
- 随后使用同一正式 Release 安装器成功升级 ef.5，备份：`/etc/slinx/backups/20261103T072834Z-126302`。未手工替换面板二进制，未升级核心版本，未重置现有节点或终止其他服务。
- 最终 systemd：MainPID `126432`，active/running；SSH active。临时构建 `/tmp` 文件已清理，可用空间回到 420MB；独立测试核心已停止。

### 部署后真实网络验收

- 既有入站 2、用户 2、节点 0、路由 0 的配置语义及 6 个原订阅响应状态/哈希与升级前快照一致。比对排除启动、核心版本和用户更新时间等运行元数据；旧删除入站接口会重新序列化用户绑定并更新 UpdatedAt，绑定 ID 的语义未变。
- 既有 443 Reality：升级前与升级后，Windows 的独立 1.13.12 客户端经公网连接并实际 HTTPS 查询均为 `38.55.195.102`，PASS。
- 正式 ef.5 创建临时认证 SOCKS5 出口后，节点 IP 检测返回 `38.55.195.102 / AS139659 / HK / ippure.com`，NodeID 正确；导出 SLINX-EF 并重新导入的节点也能真实连接，PASS。
- 对重新导入节点连续检测，历史保留两条，PASS；不是覆盖缓存或返回另一节点数据。
- 部署后的 16 项解锁请求均使用该节点，关联 IP=38.55.195.102。Netflix/ChatGPT/Bing 受限；Claude/Gemini/Disney/DAZN/Reddit 不可用；其余按本次探测返回值记录，未宣称全部平台可用。
- 创建临时 Reality 入站并绑定该 SOCKS5 出口：实际 HTTPS 请求成功；禁用出口后同一 Reality 请求失败；重新启用恢复成功；停止独立 SOCKS 服务后请求失败。两种故障均没有 Direct 回退，PASS。
- 正式 ef.5 创建临时 SSH 节点（现有部署身份、私钥、匹配的 Host Key 算法；SSH 服务位于本 VPS），实际节点检测返回 `38.55.195.102 / AS139659 LUCIDACLOUD LIMITED / ip-api.com`，PASS。结合本机→公网 VPS 的远程 SSH 实测，覆盖真实远程 SSH 及正式 API 原生映射。
- 临时出站、测试用户、测试入站、路由及对应检测记录均已清理，原配置/订阅再次复验通过。部署私钥测试输入文件已删除，凭据未进入 Git 或普通订阅。
- 本机 `TestLiveImportIntoFreshInstance`：从一个 SQLite 实例导出，再导入新的空实例，经真实节点请求两次得到 155.254.105.7，并保留两条节点历史，PASS。

### 协议及导出验收矩阵

“配置 PASS”指固定 1.13.12 的真实核心检查；不等于存在该协议的供应商真实连接。

| 协议 | 原生配置 | 兼容导出 | 真实连接 |
| --- | --- | --- | --- |
| SOCKS4 / 4a | PASS，TCP | sing-box / SLINX-EF；其他格式明确拒绝 | NOT VERIFIED，无服务器账号 |
| SOCKS5 | PASS | sing-box / SLINX-EF / Mihomo / Surge / URI | PASS，实际静态网关及正式 VPS 受控出口 |
| HTTP CONNECT | PASS | 同上 | PASS，实际静态网关 |
| HTTPS CONNECT | PASS，HTTP+TLS | sing-box / SLINX-EF / Mihomo / Surge https / URI | NOT VERIFIED，无 HTTPS 网关 |
| SSH | PASS，user/密码/私钥/私钥密码/Host Key/算法 | sing-box / SLINX-EF / Mihomo；Surge/URI 转换明确拒绝 | PASS，密码、私钥、加密私钥受控实测及真实公网 VPS 私钥认证 |
| Shadowsocks | PASS | sing-box / SLINX-EF / Mihomo / Surge / SIP002 URI | NOT VERIFIED，无出站账号 |
| VLESS | PASS | sing-box / SLINX-EF / Mihomo / URI | 原有 Reality 入口回归 PASS；远程 VLESS 供应商出口 NOT VERIFIED |
| VMess | PASS | sing-box / SLINX-EF / Mihomo / URI | NOT VERIFIED，无出站账号 |
| Trojan | PASS | sing-box / SLINX-EF / Mihomo / Surge TLS / URI | NOT VERIFIED，无出站账号 |
| Hysteria2 / TUIC | PASS，TLS/UDP 参数分开映射 | sing-box / SLINX-EF / Mihomo / 基础 URI；不兼容扩展字段拒绝 | NOT VERIFIED，无出站账号 |
| AnyTLS | PASS，独立会话字段 | sing-box / SLINX-EF / Mihomo / 基础 URI | NOT VERIFIED，无出站账号 |
| WireGuard / WARP | 原生 endpoint PASS（含可选 PSK/keepalive 等） | sing-box / SLINX-EF / 兼容单 peer Mihomo；不能转换的字段拒绝 | NOT VERIFIED，无本轮隧道实测 |
| 自定义 JSON | 清单内原生出站高级字段由核心校验 | sing-box / SLINX-EF；其他按实际类型和字段转换 | 不单独宣称新协议或连通 |

Mihomo 1.19.33 已真实解析测试；URI 回导产生的出站已实际核心检查。Surge 按官方文档转换，但当前缺少 macOS/iOS Surge 客户端，仍 NOT VERIFIED。多级依赖在 sing-box / SLINX-EF 完整保留；Mihomo 使用 dialer-proxy，Surge 支持的类型使用 underlying-proxy，URI 无依赖表达时明确拒绝。

### 尚未完成的外部验收

- 未提供真实住宅/ISP 会话供应商及换 IP API 凭据：真实供应商轮换前后不同公网 IP、真实 ISP 中继链仍 BLOCKED / NOT VERIFIED。
- 除已实测 HTTP/SOCKS5/SSH 外，没有各远程出站协议账号，不能把配置检查当真实网络 PASS。
- Surge 真客户端和 ARM64 实机未提供，分别 NOT VERIFIED；ARM64 编译与产物完整性已通过。
- 所有已发现的软件缺陷（包括旧唯一索引）已修复并复验；本轮没有留下已知的不可用服务。因外部验收缺项，不宣称所有供应商与所有客户端已经全部验收完成。

## v0.0.1 发布与 HK 升级（2026-11-03）

- 版本号改为标准 `vX.Y.Z`，从 `v0.0.1` 开始；上游原 `v0.0.1` 标签保留为 `upstream-v0.0.1`。未发布的 ef.6 工作流已取消并移除临时标签。
- Release：https://github.com/yuexuan/SLINX-EF/releases/tag/v0.0.1；Actions：https://github.com/yuexuan/SLINX-EF/actions/runs/41646251129，成功。
- 源码：`235b951393db182e185c222f331d74a2eb754c6d`。双架构二进制、安装器、BUILDINFO 均通过 SHA256 验证。
- HK 使用 Release 安装器升级成功；备份 `/etc/slinx/backups/20261103T134416Z-131242`。运行版本 `SLINX-EF v0.0.1 235b951393db182e185c222f331d74a2eb754c6d`；AMD64 SHA256 `8e50a0ed738eb5c3e2bcd587066cafb5a0e297843752cf9f082e53f51b1dd71e` 与 Release 一致。
- slinx/SSH active，NRestarts=0，面板 HTTP 200，固定核心 1.13.12 配置检查通过。数据库原入站、用户、出口、路由字段与备份一致，仅新增 Shadowsocks 字段；生成的核心 JSON 与升级前完全一致。
- 当前订阅可读取并解析出 Reality 节点；从 Windows 与服务器临时客户端经该订阅请求 HTTPS 均未通过（SSL EOF；Windows 核心记录 unknown version: 72）。不得宣称端到端连通性验收通过。本次保留现有 SSH 出口与核心配置，没有修改其凭据或路由。
