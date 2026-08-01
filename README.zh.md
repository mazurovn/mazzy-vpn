# Mazzy VPN — 中文指南

项目作者与维护者：
[Nik m (@mazurovn)](https://github.com/mazurovn)。

Mazzy VPN 是一个 Linux VPN 管理器，统一支持 AmneziaWG、WireGuard、OpenVPN
以及 NetworkManager L2TP/IPsec。它提供交互式 TUI、可自动化的 CLI、安全的
配置导入、连接检测，以及带自动回滚的事务式实连测试。

[英文架构与工作流程图](docs/ARCHITECTURE.en.md) ·
[俄文架构图](docs/ARCHITECTURE.ru.md)

## 安装与语言

```bash
git clone https://github.com/mazurovn/mazzy-vpn.git
cd mazzy-vpn
sudo ./install.sh
mazzy-vpn
```

交互式安装开始时可以选择六种语言。无人值守安装示例：

```bash
sudo ./install.sh --lang zh --yes
sudo ./install.sh --lang zh --config-dir ~/VPN-configs
```

安装后可通过菜单第 16 项或命令立即切换语言：

```bash
mazzy-vpn language
mazzy-vpn language zh
```

支持的语言代码：`ru`、`en`、`de`、`zh`、`ja`、`ko`。

## 仪表板与快速连接

仪表板显示在 TUI 菜单上方，也可单独运行：

```bash
mazzy-vpn dashboard
mazzy-vpn quick
```

仪表板会显示真实的隧道与互联网状态、当前位置、协议、默认配置、网络接口、
握手时间、公网 IP、自动启动、健康监控、备用 VPN 和各协议配置数量。

`mazzy-vpn quick` 无需再次选择即可连接已保存的默认配置。如果尚未设置默认
配置，Mazzy VPN 会打开配置选择界面，并把本次选择保存为默认值。

### 桌面仪表板与托盘

Tauri 桌面应用在现代窗口和系统托盘中提供快速连接、重新连接、断开、
刷新和自我诊断。Linux 版本可与已安装的 CLI 正常配合，并提供 AppImage、
DEB 和 RPM。macOS 与 Windows 目前仅为界面预览，原生 VPN 后端尚未实现。
Desktop 0.3 已作为未签名 preview 发布。Issue #31 已通过审核的 upstream
`glib` backport、精确源码来源验证以及通过的 RustSec、Dependabot 和 CodeQL
检查关闭。GUI 不读取配置文件或密钥。详情见
[Desktop guide (English)](docs/DESKTOP.en.md)。

## 常用命令

```bash
mazzy-vpn
mazzy-vpn list
mazzy-vpn quick
sudo mazzy-vpn connect amneziawg 1
sudo mazzy-vpn disconnect
mazzy-vpn diagnose
mazzy-vpn validate all
mazzy-vpn probe all --timeout 3 --jobs 4
sudo mazzy-vpn test openvpn 1 --timeout 60
sudo mazzy-vpn test-all all --timeout 30
sudo mazzy-vpn emergency --timeout 20
mazzy-vpn self-test
sudo mazzy-vpn doctor --fix
sudo mazzy-vpn autostart on
```

## 自动监控与修复

VPN 进程意外退出后，systemd 会自动重启它。独立的健康检查定时器约每 20 秒检查
目标状态、服务、VPN 接口以及通过该接口的真实 HTTPS 访问。当
`DESIRED=up` 而服务未运行时会立即启动；连续两次流量检查失败会触发重新连接。
`sudo mazzy-vpn doctor --fix` 可启用监控，并在默认配置有效时修复自动启动。

## 配置文件夹

```bash
mazzy-vpn init-config-dir ~/MazzyConfigs
mazzy-vpn import-dir ~/VPN-configs --dry-run
sudo mazzy-vpn import-dir ~/VPN-configs
```

创建的结构包含 `amneziawg/`、`wireguard/`、`openvpn/` 和 `l2tp/`。
Mazzy VPN 按内容识别协议，在复制前验证配置，并以 `600` 权限安装。包含可执行
hook、嵌套 OpenVPN 配置或缺少必要字段的文件会被拒绝。

## 测试、诊断与回滚

`validate all` 在不连接的情况下验证全部配置；`probe all` 以有限并发检查完整位置
列表并报告可达性、延迟和当前活动隧道。UDP 的 ICMP 被阻止时会标记为未知，而不是
错误地标记为故障；`diagnose` 检查路由、DNS、服务、接口、握手以及通过 VPN
访问互联网的能力。

`test` 和 `test-all` 会保存原连接，实测新隧道，并在成功、失败、超时或收到
信号后自动恢复。只有显式指定 `--keep` 才会保留通过测试的连接。如果 OpenVPN
服务器返回 `Too many connections`，Mazzy VPN 会准确报告服务器侧限制并立即
恢复原连接。

## 安全

请勿将真实私钥、PSK、密码或个人 VPN 配置提交到 Git。公开仓库不包含运行配置；
本地配置保存在 `/etc/vpnctl/profiles`，权限为 `600`。发布前会运行回归测试、
ShellCheck、公开内容审计和 Gitleaks。

## 作者与许可证

Copyright © 2026 [Nik m (@mazurovn)](https://github.com/mazurovn)。
项目采用 [GNU AGPL v3.0 或更高版本](LICENSE)。允许修改与分发，但必须遵守
AGPL 并保留原作者声明。
