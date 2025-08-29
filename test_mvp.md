# NetCrate MVP 测试指南

本文档描述如何测试 NetCrate 的四个核心操作，验证完整的 MVP 链路。

## 🏗️ 构建项目

```bash
# 确保在项目根目录
cd "/Users/a1-6/iCloud云盘（归档）/Desktop/文稿 - 杰的MacBook Pro/开源项目/网络攻防测试软件"

# 下载依赖
go mod tidy

# 构建项目
go build -o netcrate ./cmd/netcrate

# 验证构建
./netcrate --version
```

## 🔗 MVP 链路测试

### 1. 网络环境检测 (netenv_detect)

```bash
# 基本网络环境检测
./netcrate ops netenv

# JSON 格式输出
./netcrate ops netenv --json

# 包含网关 ping 测试
./netcrate ops netenv --ping-test

# 过滤特定接口
./netcrate ops netenv --interface en0
```

**期望结果**: 显示网络接口、IP地址、网关信息和系统能力

### 2. 主机发现 (discover)

```bash
# 自动发现当前网段的主机
./netcrate ops discover auto

# 指定网段发现
./netcrate ops discover 192.168.1.0/24

# 使用 TCP 方法发现
./netcrate ops discover auto --methods tcp

# JSON 输出
./netcrate ops discover auto --json

# 调整速率和并发
./netcrate ops discover auto --rate 50 --concurrency 100
```

**期望结果**: 发现网络中的活跃主机，显示 IP、RTT 和发现方法

### 3. 端口扫描 (scan_ports)

```bash
# 扫描发现的主机的常用端口
./netcrate ops scan ports --targets 192.168.1.1 --ports top100

# 扫描多个主机
./netcrate ops scan ports --targets 192.168.1.1,192.168.1.10 --ports web

# 扫描特定端口
./netcrate ops scan ports --targets 192.168.1.1 --ports 22,80,443,3306

# 端口范围扫描
./netcrate ops scan ports --targets 192.168.1.1 --ports 8000-8100

# JSON 输出
./netcrate ops scan ports --targets 192.168.1.1 --ports top100 --json

# 禁用服务检测
./netcrate ops scan ports --targets 192.168.1.1 --ports top100 --service-detection=false
```

**期望结果**: 显示开放端口、服务信息和扫描统计

### 4. 数据包发送 (packet_send)

```bash
# 查看可用模板
./netcrate ops packet templates

# TCP 连接测试
./netcrate ops packet send --targets 192.168.1.1:22 --template connect

# HTTP 请求
./netcrate ops packet send --targets 192.168.1.1:80 --template http

# HTTPS 请求带参数
./netcrate ops packet send --targets example.com:443 --template https --param path=/api

# TLS 握手测试
./netcrate ops packet send --targets example.com:443 --template tls

# 发送多个包
./netcrate ops packet send --targets 192.168.1.1:80 --template http --count 3

# JSON 输出
./netcrate ops packet send --targets 192.168.1.1:80 --template http --json
```

**期望结果**: 显示数据包发送结果、响应信息和统计数据

## 🔄 完整链路测试

演示从网络发现到服务探测的完整流程：

```bash
# 1. 检查网络环境
echo "=== 1. 网络环境检测 ==="
./netcrate ops netenv

# 2. 发现活跃主机
echo "=== 2. 主机发现 ==="
./netcrate ops discover auto

# 3. 扫描发现的主机端口 (手动指定IP)
echo "=== 3. 端口扫描 ==="
./netcrate ops scan ports --targets 192.168.1.1 --ports top100

# 4. 测试开放的服务 (根据扫描结果手动指定)
echo "=== 4. 服务探测 ==="
./netcrate ops packet send --targets 192.168.1.1:80 --template http
./netcrate ops packet send --targets 192.168.1.1:443 --template https
./netcrate ops packet send --targets 192.168.1.1:22 --template connect
```

## 🧪 测试场景

### 场景1: 本地网络扫描
```bash
# 发现本地网段
./netcrate ops discover auto --rate 50

# 扫描发现的主机
./netcrate ops scan ports --targets 192.168.1.1,192.168.1.10 --ports web

# 测试Web服务
./netcrate ops packet send --targets 192.168.1.1:80 --template http --param path=/
```

### 场景2: 单主机详细分析
```bash
# 扫描单主机的所有常用端口
./netcrate ops scan ports --targets 192.168.1.1 --ports top1000

# 对开放端口进行服务探测
./netcrate ops packet send --targets 192.168.1.1:22 --template connect
./netcrate ops packet send --targets 192.168.1.1:80 --template http
./netcrate ops packet send --targets 192.168.1.1:443 --template tls
```

### 场景3: 外部服务测试
```bash
# 测试公网服务 (需要 --dangerous 标志，但当前版本未完全实现)
./netcrate ops packet send --targets google.com:443 --template https
./netcrate ops packet send --targets google.com:443 --template tls
```

## ✅ 验证标准

每个操作应该：

1. **正常执行** - 不崩溃或出现严重错误
2. **提供有用输出** - 显示相关信息和统计数据
3. **支持JSON输出** - `--json` 标志产生格式良好的JSON
4. **处理错误** - 优雅处理网络错误和无效输入
5. **显示进度** - 对长时间运行的操作显示进度信息

## 🐛 常见问题

### 权限问题
某些操作可能需要 root 权限：
```bash
# 如果遇到权限问题，尝试使用 sudo
sudo ./netcrate ops discover auto
```

### 网络访问
确保目标网络可达：
```bash
# 测试基本连通性
ping 192.168.1.1
telnet 192.168.1.1 80
```

### Go 版本
确保使用正确的 Go 版本：
```bash
go version
# 应显示 go version go1.21.x 或更高版本
```

## 📊 性能基准

在标准环境中的预期性能：

- **netenv检测**: < 1秒
- **主机发现** (256个IP): 30-60秒 (100 pps)
- **端口扫描** (100端口×5主机): 30-45秒 (100 pps)
- **数据包发送**: < 5秒 per target

## 🎯 成功标准

MVP测试成功的标志：

1. ✅ 所有四个核心操作都能正常执行
2. ✅ JSON输出格式正确且完整
3. ✅ 错误处理适当（不崩溃）
4. ✅ 性能在合理范围内
5. ✅ 能够完成端到端的网络发现到服务探测流程

完成这些测试后，NetCrate MVP 即可认为功能完整并可进入下一阶段开发！