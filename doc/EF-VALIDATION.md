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

