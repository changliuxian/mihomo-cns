<h1 align="center">
  <img src="Meta.png" alt="Meta Kennel" width="200">
  <br>Meta 内核<br>
</h1>

<h3 align="center">另一个 Mihomo 内核。</h3>

<p align="center">
  <a href="https://goreportcard.com/report/github.com/MetaCubeX/mihomo">
    <img src="https://goreportcard.com/badge/github.com/MetaCubeX/mihomo?style=flat-square">
  </a>
  <img src="https://img.shields.io/github/go-mod/go-version/MetaCubeX/mihomo/Alpha?style=flat-square">
  <a href="https://github.com/MetaCubeX/mihomo/releases">
    <img src="https://img.shields.io/github/release/MetaCubeX/mihomo/all.svg?style=flat-square">
  </a>
  <a href="https://github.com/MetaCubeX/mihomo">
    <img src="https://img.shields.io/badge/release-Meta-00b4f0?style=flat-square">
  </a>
</p>

## Mihomo CNS 核心说明

### 成品

- 文件：`mihomo-linux-arm64-v1.*.*-cns`
- 平台：Linux ARM64（AArch64，ARMv8.0）
- Go：1.24.4
- 构建：`CGO_ENABLED=0`、`with_gvisor`、静态 ELF
- SHA256：`<计算后的 SHA256 值>`

### CNS 节点写法

把下列条目放到 Mihomo/OpenClash 配置的 `proxies:` 中，并将占位值换成服务端配置：

```yaml
- name: CNS-新加坡
  type: cns
  server: 你的服务器地址
  port: 23333
  key: Meng                 # 必须与服务端 Proxy_key 一致；省略时默认为 Meng
  password: 你的加密密码    # 必须与服务端 Encrypt_password 一致；服务端为空时这里也留空
  flag: httpUDP             # 必须与服务端 Udp_flag 一致；省略时默认为 httpUDP
  udp: true
  headers:
    Host: baidu.com         # 伪装 Host，可按需要修改
```

配置中的代理组继续按普通节点名引用 `CNS-新加坡` 即可。

### 手动替换 OpenClash 核心

1. 先在路由器上备份 `/etc/openclash/core/clash_meta`。
2. 停止 OpenClash。
3. 把本目录中的核心上传到 `/etc/openclash/core/clash_meta`。
4. 执行 `chmod 0755 /etc/openclash/core/clash_meta`。
5. 启动 OpenClash，在日志或命令行确认版本显示为 `v1.19.11-cns linux arm64`。

OpenClash 的“更新 Meta 核心”会覆盖自定义核心；确认可用后不要自动更新核心。
OpenClash 运行时可能使用 `/etc/openclash/clash` 链接，但持久核心位置仍应替换 `/etc/openclash/core/clash_meta`。

### 已验证项目

- CNS 节点 YAML 解析及 `type: cns` 注册
- CNS 目标字段使用标准 HTTP 头格式 `Key: Value`
- CNS 目标地址 Base64 + XOR 编码
- TCP 双向独立流式加解密
- UDP-over-TCP 的 IPv4/IPv6 封包与拆包
- HTTP 隧道握手及响应后首批缓存数据
- Mihomo 全量测试（带 `with_gvisor`）
- Linux ARM64 静态交叉编译及 ELF 架构检查
- 使用真实 CNS 节点访问国内百度地址，返回 HTTP 200

正式替换前建议先保留原核心，便于随时恢复。

## 功能特性

- 本地 HTTP/HTTPS/SOCKS 服务器，支持认证
- 支持 VMess、VLESS、Shadowsocks、Trojan、Snell、TUIC、Hysteria 协议
- 内置 DNS 服务器，旨在将 DNS 污染攻击的影响降到最低，支持 DoH/DoT 上游与 Fake IP
- 基于域名、GEOIP、IPCIDR 或进程的规则，将数据包转发到不同节点
- 远程代理组允许用户实现强大的规则，支持基于延迟的自动回退、负载均衡或自动选择节点
- 远程 Provider，允许用户远程获取节点列表，而无需硬编码到配置中
- Netfilter TCP 重定向。通过 `iptables` 将 Mihomo 部署在互联网网关上
- 功能全面的 HTTP RESTful API 控制器

## 面板

已为该项目创建了一个提供一流支持的 Web 面板，可在 [metacubexd](https://github.com/MetaCubeX/metacubexd) 查看。

## 配置示例

配置示例位于 [/docs/config.yaml](https://github.com/MetaCubeX/mihomo/blob/Alpha/docs/config.yaml)。

## 文档

文档可在 [mihomo Docs](https://wiki.metacubex.one/) 中找到。

## 开发

要求：
[Go 1.20 或更高版本](https://go.dev/dl/)

构建 mihomo：

```shell
git clone https://github.com/MetaCubeX/mihomo.git
cd mihomo && go mod download
go build
```

如果无法连接 GitHub，请设置 Go 代理：

```shell
go env -w GOPROXY=https://goproxy.io,direct
```

使用 gvisor tun 栈构建：

```shell
go build -tags with_gvisor
```

### IPTABLES 配置

在支持 `iptables` 的 Linux 系统上工作

```yaml
# 启用 TPROXY 监听器
tproxy-port: 9898

iptables:
  enable: true # 默认为 false
  inbound-interface: eth0 # 检测入站接口，默认为 'lo'
```

## 调试

查看 [wiki](https://wiki.metacubex.one/api/#debug) 获取调试 API 的使用说明。

## 致谢

- [Dreamacro/clash](https://github.com/Dreamacro/clash)
- [SagerNet/sing-box](https://github.com/SagerNet/sing-box)
- [riobard/go-shadowsocks2](https://github.com/riobard/go-shadowsocks2)
- [v2ray/v2ray-core](https://github.com/v2ray/v2ray-core)
- [WireGuard/wireguard-go](https://github.com/WireGuard/wireguard-go)
- [yaling888/clash-plus-pro](https://github.com/yaling888/clash)

## 许可证

本软件基于 GPL-3.0 许可证发布。

**此外，任何与 `MetaCubeX` 无关联的下游项目，其名称中不得包含 `mihomo` 字样。**
