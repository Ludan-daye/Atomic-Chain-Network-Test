# NetCrate 速率限制与并发控制

## 默认配置档位

### Safe Profile (默认)
**适用场景**：日常网络诊断、小规模测试

| 操作类型 | 最大速率 | 最大并发 | 超时设置 | 重试次数 | 需要确认 |
|---------|----------|----------|----------|----------|----------|
| ICMP Ping | 50 pps | 100 | 1000ms | 2 | 否 |
| TCP Connect | 100 pps | 200 | 800ms | 1 | 否 |
| UDP Probe | 80 pps | 150 | 1200ms | 1 | 否 |
| HTTP Request | 30 pps | 50 | 3000ms | 2 | 否 |
| DNS Query | 100 pps | 100 | 2000ms | 2 | 否 |

### Fast Profile
**适用场景**：专业渗透测试、大规模资产扫描

| 操作类型 | 最大速率 | 最大并发 | 超时设置 | 重试次数 | 需要确认 |
|---------|----------|----------|----------|----------|----------|
| ICMP Ping | 200 pps | 400 | 800ms | 1 | 是 |
| TCP Connect | 400 pps | 800 | 600ms | 1 | 是 |
| UDP Probe | 300 pps | 600 | 1000ms | 1 | 是 |
| HTTP Request | 100 pps | 200 | 2000ms | 1 | 是 |
| DNS Query | 400 pps | 400 | 1500ms | 1 | 是 |

### Custom Profile
**适用场景**：专家用户自定义配置

- 用户可通过配置文件或命令行参数自定义所有限制
- 超过 Fast Profile 限制时需要 `--dangerous` 标志
- 系统硬限制：≤2000 pps，≤5000 并发（防止系统过载）

## 动态调节机制

### 交互模式下的实时调节

**键盘快捷键：**
- `+` / `-`：调节发送速率（±10%）
- `[` / `]`：调节并发数（±20%）
- `s`：暂停/恢复发送
- `r`：重置为默认配置

**自适应限速：**
- 检测到网络拥堵时自动降低速率
- 错误率 >5% 时触发退避算法
- 成功率恢复后逐步提升速率

### 网络环境感知

| 网络类型 | 速率系数 | 并发系数 | 自动检测依据 |
|---------|----------|----------|--------------|
| 本地回环 | 5.0x | 2.0x | 目标为 127.0.0.1/::1 |
| 局域网 | 1.0x | 1.0x | 私网地址 + RTT <10ms |
| 广域网 | 0.5x | 0.8x | 公网地址 + RTT >50ms |
| 移动网络 | 0.3x | 0.5x | 高延迟 + 抖动 |

## 合规性控制

### 私网 vs 公网

**私网目标（默认允许）：**
```
10.0.0.0/8          # Class A 私网
172.16.0.0/12       # Class B 私网
192.168.0.0/16      # Class C 私网
127.0.0.0/8         # IPv4 回环
fe80::/10           # IPv6 本地链路
::1/128             # IPv6 回环
```

**公网目标（需要授权）：**
- 需要 `--dangerous` 标志
- 强制降低到 Safe Profile 限制
- 显示目标地址归属信息（ASN、地理位置）
- 需要交互式二次确认

### 特殊限制

**教育网段：**
- `.edu` 域名：强制 Safe Profile
- 已知大学 IP 段：额外 0.5x 速率限制

**云服务提供商：**
- AWS/GCP/Azure 已知 IP 段：需要额外确认
- 显示服务商名称和可能的服务类型

**关键基础设施：**
- 政府、金融、医疗已知网段：默认禁止
- 需要 `--allow-critical` 标志

## 配置文件示例

### 默认配置位置
```
~/.netcrate/config.yaml
/etc/netcrate/config.yaml
./netcrate.yaml
```

### 配置文件格式
```yaml
# NetCrate 配置文件
rate_limits:
  profile: "safe"  # safe, fast, custom
  
  # 自定义配置（profile: custom 时生效）
  custom:
    icmp:
      rate: 100      # pps
      concurrency: 200
      timeout: "1s"
      retries: 2
    tcp:
      rate: 200
      concurrency: 400
      timeout: "800ms"
      retries: 1
    udp:
      rate: 150
      concurrency: 300
      timeout: "1.2s"
      retries: 1
    http:
      rate: 50
      concurrency: 100
      timeout: "3s"
      retries: 2
    dns:
      rate: 200
      concurrency: 200
      timeout: "2s"
      retries: 2

# 合规性设置
compliance:
  allow_public: false         # 是否允许公网目标
  require_confirmation: true  # 是否需要交互确认
  
  # 允许的目标范围（CIDR 格式）
  allowed_ranges:
    - "10.0.0.0/8"
    - "172.16.0.0/12"
    - "192.168.0.0/16"
    - "127.0.0.0/8"
  
  # 明确禁止的目标范围
  blocked_ranges:
    - "0.0.0.0/8"          # 保留地址
    - "224.0.0.0/3"        # 多播地址
    - "169.254.0.0/16"     # 链路本地地址

# 系统限制（硬限制，不可超越）
system_limits:
  max_rate: 2000         # 绝对最大 pps
  max_concurrency: 5000  # 绝对最大并发
  max_targets: 100000    # 单次最大目标数
  max_runtime: "24h"     # 单次最大运行时间
```

## 监控与统计

### 实时指标
- 当前发送速率（实时/平均/峰值）
- 并发连接数
- 成功率和错误率
- 网络延迟统计（最小/平均/最大）

### 输出示例
```
[Scanning] 192.168.0.0/24  Rate: 98/100 pps  Concurrent: 156/200  Success: 94.2%
Progress: ████████████░░  85%  ETA: 00:02:15  Errors: 8  Timeouts: 3
```

---

**版本**：v1.0  
**最后更新**：2025-08-28  
**适用版本**：NetCrate >= 0.1.0