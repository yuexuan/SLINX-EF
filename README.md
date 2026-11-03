# SLINX-EF

Everflame 的独立单 VPS 节点组件。基于 [slinxlink/node](https://github.com/slinxlink/node)，保留上游历史与 MIT LICENSE，感谢原作者。保持原有界面与单机架构，不包含计费、多机控制或代理池。

## 安装与更新

支持 systemd 环境的 Debian、Ubuntu、Rocky、AlmaLinux、Fedora、CentOS/RHEL（dnf），Linux AMD64 / ARM64。

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/yuexuan/SLINX-EF/main/install.sh)
```

指定版本：在上述命令末尾加 `v0.0.1`。重复执行即更新；面板内置更新同样只读取此 Fork，启动独立 systemd 临时单元执行 Release 内经 SHA256 验证的安装脚本。

保留 `/etc/slinx`、`slinx.service` 和 `slinx` 管理命令。安装后执行 `slinx` 查看面板端口、路径及账号信息，建议首次通过 SSH 隧道访问。脚本不会修改系统时区或其他 sing-box 服务。发现未标记的已有 SLINX 安装会退出，需人工检查后迁移。

面板从本仓库 Release 下载，检查 SHA256SUMS。核心固定使用上游 `sing-box-1.13.12`，压缩包 SHA256 固定在脚本中，不追踪 latest。更新前停止服务并备份二进制、数据库、证书、配置至 `/etc/slinx/backups/时间戳`；启动或配置检查失败自动恢复已有版本。首次安装失败保留诊断文件，不宣称成功。

## ISP 出口

端点页面选择 SOCKS5 或 HTTP CONNECT，填入标签、地址、端口、用户名和密码，然后使用现有路由界面绑定入站。多个入站可共用一个出口。HTTP CONNECT 仅支持 TCP。代理连接失败不会尝试 Direct；禁用保留绑定并阻断，重新启用恢复；删除改为 block，需要显式重新绑定。凭证仅写入服务端配置，不加入订阅，订阅仍按原有入站名称生成节点。

## 发布与来源核对

手动触发 Release 工作流只生成构建 Artifact；推送 `vX.Y.Z` 标签才发布。复用 Dockerfile.builder 和 Makefile 的 CGO 交叉编译及前端嵌入方式。两种架构全部构建成功后先创建草稿并上传资源，再公开 Release。资源包括两个面板二进制、SHA256SUMS、BUILDINFO、install.sh。

`slinx version`（二进制 `/etc/slinx/slinx version`）输出版本及源码提交；可与 BUILDINFO、标签提交及 Actions 记录核对。

## 排错、恢复与卸载

`systemctl status slinx`、`journalctl -u slinx -n 100`；核心检查：`cd /etc/slinx && bin/sing-box check -c data/sing-box.json`。更新日志：`journalctl -u 'slinx-ef-update-*'`。手工回滚时先停止服务，从备份还原 slinx、bin、data、cert 与 service，然后 daemon-reload 并启动。

卸载：先备份 `/etc/slinx`，执行 `systemctl disable --now slinx`，删除 `/etc/systemd/system/slinx.service` 和 `/usr/local/bin/slinx`，再 daemon-reload。数据目录由管理员确认后手动删除。

## 同步上游

`git fetch upstream` 后在独立分支合并上游 main，检查数据库迁移、核心配置兼容性、更新源和安装脚本，跑测试及双架构构建后发布新的 EF 标签。内部 Go 模块和表名保留上游标识。

实际交付验证记录见 `doc/EF-VALIDATION.md`；未通过真实部署前不视为 P4 完成。

真实部署已验证：建议 Reality 握手目标使用本次验证通过的 `www.cloudflare.com`。完整结果与限制见 [实际验收记录](doc/EF-VALIDATION.md)。
