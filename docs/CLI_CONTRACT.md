# NetCrate CLI 交互契约

## 概览

本文档定义了 NetCrate 所有命令的精确交互行为、输入输出格式、错误处理和用户体验标准。所有实现必须严格遵循此契约。

**命令族架构**:
- `netcrate quick` - 向导式交互模式
- `netcrate ops` - 原子操作模式  
- `netcrate templates` - 模板管理模式
- `netcrate config` - 配置管理模式
- `netcrate output` - 结果管理模式

**全局选项**:
```bash
--json / --ndjson          # 机器可读输出
--out <path>               # 导出运行结果
--dry-run                  # 显示计划，不执行
--yes                      # 跳过所有确认
--quiet                    # 极简输出
--dangerous                # 允许高风险操作
--profile <safe|fast|custom> # 预设速率配置
--config <file>            # 指定配置文件
--workspace <dir>          # 运行产物目录
```

---

## Quick 模式详细契约

### 命令格式
```bash
netcrate quick [--dry-run] [--yes] [--out <file>] [--profile <profile>]
```

### 交互流程标准

#### 步骤 1: 网卡选择
**触发条件**: 检测到多个活跃网络接口

**界面契约**:
```
[网卡选择]
→ 检测到活动接口：
  ▸ en0  192.168.0.23/24  网关:192.168.0.1
    en5  10.10.0.5/24     网关:10.10.0.1
    eth0 172.16.1.100/16  网关:172.16.0.1
↑↓ 选择，Enter 确认，/ 搜索   (默认: en0)
>
```

**交互规则**:
- `↑↓` 箭头键导航
- `Enter` 确认选择
- `/` 进入搜索模式
- `q` 退出程序
- **自动选择**: 仅一个接口时直接使用
- **默认值**: 第一个私网接口，或主网关接口

**搜索模式**:
```
[网卡选择 - 搜索模式]
过滤: en_
  ▸ en0  192.168.0.23/24  网关:192.168.0.1
    en5  10.10.0.5/24     网关:10.10.0.1
Backspace 清除，Esc 退出搜索
```

#### 步骤 2: 网段确认
**界面契约**:
```
[网段确认]
→ 推导网段: 192.168.0.0/24 (基于 en0)
输入以修改（CIDR/逗号分隔目标），直接回车确认：
> 
```

**输入处理**:
- **空输入**: 使用推导的网段
- **CIDR格式**: `192.168.1.0/24`, `10.0.0.0/8`
- **IP范围**: `192.168.1.1-192.168.1.100`  
- **多目标**: `192.168.0.0/24,10.0.0.1,example.com`
- **文件引用**: `file:targets.txt`

**验证错误**:
```
[网段确认]
→ 推导网段: 192.168.0.0/24 (基于 en0)
输入以修改（CIDR/逗号分隔目标），直接回车确认：
> 8.8.8.8
❌ 错误: 公网地址需要 --dangerous 标志
   支持的格式: CIDR (192.168.0.0/24)、IP范围 (192.168.0.1-10)、文件 (file:targets.txt)
> 
```

#### 步骤 3: 合规确认
**触发条件**: 首次使用或检测到风险操作

**界面契约**:
```
[合规确认]
仅在授权范围内使用。本工具默认限制在私网段。
目标: 192.168.0.0/24 (256 个地址)
预计探活时间: ~30秒 (100 pps)

继续？ (y/N)
```

**公网目标提醒**:
```
[⚠️  公网目标警告]
检测到公网地址: 203.0.113.1
- 确保拥有目标授权
- 遵守当地法律法规  
- 建议使用较低速率

需要额外确认 (输入 'YES' 继续): 
```

#### 步骤 4: 主机发现 (Discover)
**界面契约**:
```
[Discover 进行中]  192.168.0.0/24   速率:98/100 pps   已发现: 12   错误: 0
████████████████░░░░  75%   剩余时间: 00:00:45   当前: 192.168.0.192
```

**实时控制**:
- `v` 切换详情/简报视图
- `s` 暂停/恢复
- `+/-` 调整速率 (±10%)
- `[/]` 调整并发 (±20%)  
- `c` 取消操作
- `q` 退出程序

**详情视图**:
```
[Discover 详情]  192.168.0.0/24
发送: 195/256  接收: 12  错误: 3  超时: 8
最近响应:
  192.168.0.1    1.2ms  [Gateway]
  192.168.0.23   2.4ms  
  192.168.0.55   15.6ms [Slow]

按 v 返回简报视图
```

**完成界面**:
```
[Discover 完成]  192.168.0.0/24   耗时: 01:23   已发现: 18 台主机
成功率: 94.2%   平均延迟: 3.8ms   错误: 8   超时: 12

按 Enter 选择目标，r 重新扫描，q 退出
```

#### 步骤 5: 目标主机选择
**界面契约**:
```
[存活主机 18 台]
过滤：/ 输入IP或前缀；Space 多选；A 全选；Enter 确认
  ▸ 192.168.0.1    RTT 1.8ms  (网关)
    192.168.0.8    RTT 2.4ms
  ✓ 192.168.0.16   RTT 3.1ms  
    192.168.0.23   RTT 4.2ms  (本机)
    192.168.0.55   RTT 15.6ms (慢速)
    ...

已选择: 1 台   按 Enter 继续下一步
```

**选择交互**:
- `↑↓` 导航
- `Space` 切换选中状态  
- `A` 全选/取消全选
- `Enter` 确认选择
- `/` 进入过滤模式
- `Esc` 清除过滤/选择

#### 步骤 6: 端口扫描设置
**界面契约**:
```
[端口集合]
选择端口集合：
  ▸ top100 (默认) - 最常用的100个端口
    top1000 - 常用1000端口，适合详细扫描  
    web - Web服务端口 (80,443,8080,8000,8443)
    database - 数据库端口 (3306,5432,1433,27017)
    custom - 自定义端口范围
↑↓ 选择，Enter 确认
```

**自定义端口输入**:
```
[自定义端口]
输入端口表达式：
> 22,80,443,8000-8100,9000
支持格式: 单个端口 (80)、范围 (8000-8100)、列表 (22,80,443)
```

#### 步骤 7: 端口扫描执行
**界面契约**:
```
[Port Scan 进行中] 目标: 5 主机  端口: 100  并发:156/200  速率:98/100 pps
████████████░░░░  85%  ETA: 00:02:15  开放: 11  错误: 2

192.168.0.1   ████████████████████ [20/20]  开放: 4   
192.168.0.8   ████████████░░░░░░░░ [12/20]  开放: 2
192.168.0.16  ██████░░░░░░░░░░░░░░ [6/20]   开放: 1
...

实时控制: v详情 s暂停 +/-速率 [/]并发 c取消
```

**扫描完成**:
```
[Port Scan 完成] 目标: 5 主机  耗时: 03:24  开放端口: 15 个
成功率: 97.8%   平均端口响应: 45ms   超时: 34

按 Enter 选择端口进行探测，s 保存结果，q 退出
```

#### 步骤 8: 开放端口选择  
**界面契约**:
```
[开放端口 15 个]
选择要探测的服务：Space 多选，Enter 确认，/ 过滤
  ▸ 192.168.0.1:22    SSH    (响应快)
  ✓ 192.168.0.1:80    HTTP   
  ✓ 192.168.0.1:443   HTTPS  
    192.168.0.8:22    SSH
    192.168.0.8:3306  MySQL  (可能)
    192.168.0.16:80   HTTP
    ...

已选择: 2 个端口   按 Enter 继续探测
```

#### 步骤 9: 服务探测模板选择
**界面契约**:
```
[服务探测模板]
选择探测方式：
  ▸ auto - 根据端口自动选择最佳探测模板
    tcp_connect - 简单 TCP 连接测试
    http_get - HTTP GET 请求 (/)  
    tls_hello - TLS 握手信息
    service_banner - 尝试获取服务横幅
    custom - 自定义探测方式

↑↓ 选择，Enter 确认   (推荐: auto)
```

#### 步骤 10: 服务探测执行
**界面契约**:
```
[Service Probe 进行中] 目标: 2 个端口  模板: auto
进度: ████████████████████ 100%  完成: 2/2   耗时: 00:03

192.168.0.1:80   → HTTP/1.1 200 OK (nginx/1.18.0)
192.168.0.1:443  → TLS 1.3, CN=example.com (Let's Encrypt)

实时控制: v 查看详情，Enter 继续
```

#### 步骤 11: 结果汇总  
**界面契约**:
```
[探测完成] 🎯
发现: 18 台主机，15 个开放端口，2 个活跃服务
总耗时: 05:47   数据已保存至: ~/.netcrate/runs/2025-08-29-1430/

📊 汇总统计:
主机发现   18/256 (7.0%)   01:23   成功率 94.2%
端口扫描   15/500 (3.0%)   03:24   成功率 97.8%  
服务探测   2/2   (100%)    00:03   成功率 100%

📁 结果操作:
o - 导出结果 (JSON/CSV)    r - 重新运行完整扫描
n - 基于结果继续探测      s - 保存为模板
q - 退出

选择操作: 
```

### 静默模式行为

**完全静默**:
```bash
netcrate quick --yes --quiet --out results.json 192.168.0.0/24
# 输出: 仅关键信息和进度
Discover: 18/256 hosts [████████████████████] 100%
Scan: 15/500 ports [████████████████████] 100%  
Probe: 2/2 services [████████████████████] 100%
Results: ~/.netcrate/runs/2025-08-29-1430/summary.json
```

**Dry-Run 模式**:
```bash
netcrate quick --dry-run 192.168.0.0/24
# 输出: 完整执行计划
[Dry Run] NetCrate Quick 模式执行计划

1. 网络接口检测
   → 将选择接口: en0 (192.168.0.23/24)
   
2. 目标确认
   → 目标范围: 192.168.0.0/24 (256 地址)
   → 预计私网: 是
   
3. 主机发现  
   → 方法: ICMP Echo + TCP SYN (80,443,22)
   → 速率: 100 pps (Safe profile)
   → 预计时间: ~30 秒
   
4. 端口扫描
   → 端口集: top100 (100 端口)
   → 并发: 200
   → 预计时间: ~45 秒 (基于发现的主机数)
   
5. 服务探测
   → 模板: auto (根据端口选择)
   → 预计时间: ~5 秒
   
6. 结果输出
   → 格式: 交互表格 + JSON
   → 保存位置: ~/.netcrate/runs/2025-08-29-1430/

总预计时间: 01:20
使用配置: Safe profile (合规模式)

✅ 所有检查通过，使用 --yes 跳过确认直接执行
```

---

## Ops 模式详细契约

### discover 原子操作

**命令格式**:
```bash
netcrate ops discover [targets|auto] [options]
```

**参数说明**:
- `targets`: CIDR、IP范围、文件或 `auto` 自动推导
- `--iface`: 指定网络接口 
- `--rate`: 发送速率 (pps)
- `--timeout`: 超时时间  
- `--concurrency`: 并发数
- `--methods`: 探测方法 (icmp,tcp,udp)

**输入处理**:
```bash
# 自动推导 (默认行为)
netcrate ops discover auto
→ 检测接口，推导 CIDR，进入交互确认

# 明确目标
netcrate ops discover 192.168.0.0/24
→ 直接开始扫描

# 多目标  
netcrate ops discover 192.168.0.0/24,10.0.0.1-10.0.0.100,example.com
→ 解析后按顺序扫描

# 从文件
netcrate ops discover file:targets.txt  
→ 读取文件，每行一个目标
```

**输出契约**:

**交互模式 (默认)**:
```
[Discover] 192.168.0.0/24  方法: ICMP+TCP  速率: 100 pps
████████████████████ 100%   发现: 18/256   耗时: 01:23

结果保存: ~/.netcrate/runs/2025-08-29-1430/discover.json
按 Enter 查看结果，o 导出，q 退出
```

**JSON 输出**:
```bash
netcrate ops discover --json 192.168.0.0/24
```
```json
{
  "run_id": "2025-08-29-1430-discover",
  "start_time": "2025-08-29T14:30:15Z",
  "end_time": "2025-08-29T14:31:38Z", 
  "duration": 83.2,
  "targets_input": "192.168.0.0/24",
  "targets_resolved": 256,
  "hosts_discovered": 18,
  "success_rate": 0.942,
  "method": "icmp+tcp",
  "rate": 100,
  "results": [
    {
      "host": "192.168.0.1",
      "status": "up",
      "rtt": 1.8,
      "method": "icmp",
      "timestamp": "2025-08-29T14:30:16Z"
    }
  ],
  "stats": {
    "sent": 256,
    "received": 18, 
    "errors": 8,
    "timeouts": 12
  }
}
```

### scan ports 原子操作

**命令格式**:
```bash
netcrate ops scan ports --targets <input> [options]
```

**目标输入规则**:
```bash
# 使用上次 discover 结果
netcrate ops scan ports --targets last
→ 自动读取最近一次 discover 的存活主机

# 明确指定主机
netcrate ops scan ports --targets 192.168.0.1,192.168.0.8
→ 扫描指定主机

# 从文件读取
netcrate ops scan ports --targets file:hosts.txt
→ 文件格式: 每行一个 IP

# 组合 discover 结果
netcrate ops discover 192.168.0.0/24 --quiet | netcrate ops scan ports --targets -
→ 管道操作
```

**端口参数**:
```bash  
--ports top100              # 预设端口集
--ports 80,443,22            # 明确端口列表
--ports 8000-9000           # 端口范围  
--ports file:ports.txt       # 从文件读取
--ports custom              # 进入交互选择
```

**输出契约**:

**交互模式**:
```
[Port Scan] 5 hosts × top100 ports  并发: 200  速率: 100 pps
████████████████████ 100%  发现开放端口: 15   耗时: 03:24

结果预览:
192.168.0.1:22     SSH     2.1ms
192.168.0.1:80     HTTP    1.8ms  
192.168.0.1:443    HTTPS   3.2ms

完整结果: ~/.netcrate/runs/2025-08-29-1430/scan.json
按 Enter 选择端口，o 导出，n 下一步操作，q 退出
```

### packet send 原子操作

**命令格式**:
```bash  
netcrate ops packet send [options]
```

**目标指定**:
```bash
# 使用上一步结果
netcrate ops packet send --template syn
→ 对上次扫描的开放端口发送 SYN

# 明确目标
netcrate ops packet send --to 192.168.0.1:80 --template http
→ 发送 HTTP 请求

# 多目标
netcrate ops packet send --to 192.168.0.1:80,192.168.0.8:22 --template tcp_connect
```

**模板参数**:
```bash
--template syn                    # TCP SYN 探测
--template http --param path=/admin --param method=GET
--template dns --param domain=example.com --param type=A
--template tls --param sni=example.com
--template icmp --param type=echo
```

**输出契约**:
```bash  
netcrate ops packet send --template http --to 192.168.0.1:80 --json
```
```json
{
  "run_id": "2025-08-29-1430-packet",
  "template": "http",
  "targets": ["192.168.0.1:80"],
  "count": 1,
  "results": [
    {
      "target": "192.168.0.1:80", 
      "rtt": 2.3,
      "status": "success",
      "response": {
        "status_code": 200,
        "headers": {"server": "nginx/1.18.0"},
        "body_preview": "<!DOCTYPE html>..."
      },
      "timestamp": "2025-08-29T14:30:45Z"
    }
  ]
}
```

---

## Templates 模式详细契约

### 模板运行流程

**命令格式**:
```bash
netcrate templates run <name> [--param key=value] [options]
```

**参数补齐交互**:
```bash
netcrate templates run basic_scan
```
```
[模板参数补齐] basic_scan v1
描述: Basic network discovery and port scanning

缺少必需参数，请补充:

target_range (目标网络范围):
描述: Target network range (CIDR notation)  
类型: string, 必填: 是, 默认: auto
> auto

ports (端口范围):  
描述: Port range to scan
类型: string, 必填: 否, 默认: top100
> (直接 Enter 使用默认值)

✅ 参数验证通过，开始执行模板...
```

**执行进度界面**:
```
[模板运行] basic_scan v1  步骤: 2/3   总耗时: 02:15

✅ discover        完成   01:23   发现18台主机
🔄 scan_ports     进行中  00:52   已扫描3/5台主机  
⏳ summary        等待中

当前步骤详情:
[Port Scan] 192.168.0.8  端口: 67/100   开放: 2   ETA: 00:15

控制: v 详情  s 暂停  c 取消步骤  q 终止模板
```

**步骤失败处理**:
```
[模板运行] basic_scan v1  步骤: 2/3   

✅ discover        完成   01:23   发现18台主机
❌ scan_ports     失败   00:32   错误: 网络超时
⏸️  summary        跳过   依赖失败

错误详情: 目标主机 192.168.0.55 连续超时，建议检查网络连接

选择操作:
r - 重试失败步骤 (调整参数)
s - 跳过失败步骤继续  
c - 取消整个模板
> r

重试参数调整:
超时时间 (当前: 800ms): > 2000ms
并发数 (当前: 200): > 100
```

### 模板列表和查看

**命令格式**:
```bash
netcrate templates ls [--tag <tag>] [--author <author>]
netcrate templates view <name>
```

**列表输出**:
```
[模板库] 5 个可用模板

内置模板:
  basic_scan        v1   NetCrate Team      basic,discovery
  web_enum          v1   NetCrate Team      web,enumeration  
  db_discover       v1   NetCrate Team      database,discovery
  
用户模板 (~/.netcrate/templates/):
  my_custom_scan    v1   User               custom
  pentest_flow      v2   User               pentest,comprehensive

使用: netcrate templates view <name> 查看详情
     netcrate templates run <name> 运行模板
```

**模板详情**:
```bash
netcrate templates view basic_scan
```
```
[模板详情] basic_scan v1

基本信息:
  名称: basic_scan
  版本: v1  
  作者: NetCrate Team
  描述: Basic network discovery and port scanning
  标签: basic, discovery, scanning
  危险操作: 否

必需参数:
  target_range (string, required)
    描述: Target network range (CIDR notation)
    默认值: auto
    验证: CIDR格式

可选参数:  
  ports (string, optional)
    描述: Port range to scan  
    默认值: top100
    验证: 端口范围格式

执行步骤:
  1. discover → 主机发现
     输入: {{ .target_range }}
     配置: 速率100pps, 超时1000ms
     
  2. scan_ports → 端口扫描  
     输入: {{ .discover.hosts }} (上一步结果)
     配置: 并发200, 超时800ms
     依赖: discover
     
  3. summary → 结果汇总
     输入: {{ .scan_ports.results }}
     格式: table
     依赖: scan_ports

预计执行时间: 2-5 分钟 (取决于目标数量)
```

---

## Config 模式详细契约

### 配置交互式编辑

**命令格式**:
```bash
netcrate config edit [--category <category>]
```

**主配置菜单**:
```
[NetCrate 配置编辑]

配置类别:
  1. 合规和安全设置    当前: 私网模式，Safe配置档
  2. 性能和速率限制    当前: Safe档 (100pps, 200并发)  
  3. 输出和存储设置    当前: ~/.netcrate/runs, 30天保留
  4. 界面和交互选项    当前: 彩色输出，交互确认开启
  5. 网络和接口设置    当前: 自动选择，优先私网接口

选择类别 (1-5): > 1
```

**安全设置详情**:
```
[合规和安全设置]

当前配置:
✅ 默认仅允许私网目标
✅ 公网操作需要 --dangerous 
✅ 交互确认开启
⚠️  首次合规声明: 已确认

修改选项:
1. 允许的目标范围       当前: RFC1918 私网
2. 明确禁止的范围       当前: 保留地址段
3. 危险操作确认级别     当前: 双重确认
4. 默认速率配置档       当前: Safe
5. 重置首次声明状态     

选择修改项 (1-5，Enter 返回): > 1

[允许的目标范围]
当前允许的网络范围:
  10.0.0.0/8          Class A 私网
  172.16.0.0/12       Class B 私网  
  192.168.0.0/16      Class C 私网
  127.0.0.0/8         回环地址

操作:
+ 添加新范围    - 移除范围    r 恢复默认    Enter 完成
> + 203.0.113.0/24

⚠️  警告: 添加公网范围 203.0.113.0/24
这将允许对公网目标进行测试，请确保:
- 拥有目标测试授权  
- 遵守当地法律法规
- 理解潜在的法律风险

确认添加? (输入 'YES' 确认): > YES

✅ 已添加 203.0.113.0/24 到允许范围
修改后配置将在下次运行时生效
```

### 配置显示

**命令格式**:
```bash
netcrate config show [--category <category>]
```

**完整配置输出**:
```
[NetCrate 当前配置]

📁 配置文件位置:
  用户配置: ~/.netcrate/config.yaml
  系统配置: /etc/netcrate/config.yaml  
  当前使用: ~/.netcrate/config.yaml

🛡️  合规和安全:
  允许公网目标: 否
  需要交互确认: 是
  允许的范围: 4 个私网段
  禁止的范围: 3 个保留段
  首次声明: 已确认 (2025-08-29)

⚡ 性能配置:  
  当前配置档: Safe
  最大速率: 100 pps  
  最大并发: 200
  默认超时: 1000ms
  重试次数: 2

💾 输出和存储:
  工作目录: ~/.netcrate/runs
  保留策略: 30天 或 1000次运行
  默认输出格式: 交互表格
  自动导出: JSON (后台)

🎨 界面设置:
  彩色输出: 启用
  进度条: 启用  
  交互确认: 启用
  详情显示: 自动切换

🌐 网络设置:
  接口选择: 自动 (优先私网)
  DNS服务器: 系统默认
  连接超时: 5秒
  最大重定向: 3次

使用 'netcrate config edit' 修改配置
```

---

## Output 模式详细契约

### 结果管理

**命令格式**:
```bash
netcrate output show [--last | --run <id>] [--format <format>]
netcrate output list [--limit <n>] [--filter <filter>]  
netcrate output export --run <id> --out <path> [--format <format>]
```

### 结果显示

**最近结果**:
```bash
netcrate output show --last
```
```
[运行结果] 2025-08-29-1430 (最近)

📊 执行摘要:
  命令: netcrate quick 192.168.0.0/24
  开始时间: 2025-08-29 14:30:15  
  总耗时: 05:47
  状态: ✅ 完成

🎯 发现统计:  
  目标范围: 192.168.0.0/24 (256 地址)
  存活主机: 18 台 (7.0%)  
  开放端口: 15 个
  识别服务: 12 个

📁 数据位置:
  运行目录: ~/.netcrate/runs/2025-08-29-1430/
  汇总文件: summary.json (2.3KB)
  详细数据: results.ndjson (15.7KB)
  
操作选项:
v - 查看详细结果    e - 导出数据    r - 重新运行
d - 删除此次结果    n - 基于结果继续   q - 退出

选择操作: > v
```

**详细结果视图**:
```
[详细结果] 2025-08-29-1430

🖥️  发现的主机 (18):
┌─────────────────┬──────┬────────┬─────────────┐
│ IP地址          │ RTT  │ 状态   │ 备注        │
├─────────────────┼──────┼────────┼─────────────┤
│ 192.168.0.1     │ 1.8ms│ UP     │ 网关        │
│ 192.168.0.8     │ 2.4ms│ UP     │             │ 
│ 192.168.0.16    │ 3.1ms│ UP     │             │
│ 192.168.0.23    │ 1.2ms│ UP     │ 本机        │
│ ...             │      │        │             │
└─────────────────┴──────┴────────┴─────────────┘

🔌 开放端口 (15):  
┌─────────────────┬──────┬─────────┬────────────────┐
│ 目标            │ 端口 │ 状态    │ 服务           │
├─────────────────┼──────┼─────────┼────────────────┤
│ 192.168.0.1     │ 22   │ OPEN    │ SSH            │
│ 192.168.0.1     │ 80   │ OPEN    │ HTTP           │
│ 192.168.0.1     │ 443  │ OPEN    │ HTTPS          │
│ ...             │      │         │                │
└─────────────────┴──────┴─────────┴────────────────┘

翻页: ↑↓ 导航  Space 下一页  q 退出详情
```

### 历史运行列表

**命令**:
```bash  
netcrate output list --limit 10
```
```
[历史运行记录] 最近10次

┌──────────────────┬─────────────────────┬──────────┬─────────┬───────┬────────┐
│ 运行ID           │ 开始时间            │ 命令     │ 目标    │ 状态  │ 大小   │
├──────────────────┼─────────────────────┼──────────┼─────────┼───────┼────────┤
│ 2025-08-29-1430  │ 2025-08-29 14:30:15 │ quick    │ 256主机 │ ✅完成│ 18.1KB │
│ 2025-08-29-1215  │ 2025-08-29 12:15:33 │ discover │ 512主机 │ ✅完成│ 12.4KB │  
│ 2025-08-29-1156  │ 2025-08-29 11:56:21 │ scan     │ 5主机   │ ❌错误│ 3.2KB  │
│ ...              │                     │          │         │       │        │
└──────────────────┴─────────────────────┴──────────┴─────────┴───────┴────────┘

使用 'netcrate output show --run <id>' 查看详情
使用 'netcrate output export --run <id> --out file.json' 导出

筛选选项: --filter completed (仅完成), --filter error (仅错误)
```

### 结果导出

**交互式导出**:
```bash
netcrate output export --run 2025-08-29-1430
```
```
[结果导出] 2025-08-29-1430

选择导出格式:
  1. JSON - 结构化数据，便于程序处理
  2. CSV - 表格格式，Excel兼容  
  3. HTML - 网页报告，包含图表
  4. TXT - 纯文本，人类可读
  5. 完整包 - 所有文件打包 (tar.gz)

选择格式 (1-5): > 1

导出选项:
✅ 包含主机发现结果
✅ 包含端口扫描结果  
✅ 包含服务探测结果
✅ 包含运行统计信息
☐ 包含原始日志文件
☐ 包含错误详情

输出文件路径: > ~/Desktop/netcrate-results-2025-08-29.json

✅ 导出完成: ~/Desktop/netcrate-results-2025-08-29.json (24.3KB)
数据包含: 18个主机, 15个端口, 12个服务
```

---

## 错误处理和恢复契约

### 网络错误处理

**权限不足**:
```
❌ 错误: 权限不足
Raw socket 创建失败，某些功能受限。

可用选项:
1. 使用 sudo netcrate ... (推荐)
2. 继续使用受限功能 (TCP connect 扫描)
3. 退出并检查权限

选择 (1-3): > 2

⚠️  切换到受限模式:
- ICMP 探测 → 系统 ping 命令
- Raw SYN 扫描 → TCP connect 扫描
- 自定义数据包 → 不可用

继续? (y/N): > y
```

**网络超时**:
```
⚠️  网络异常检测到
目标 192.168.0.55 连续超时 (5次)

建议操作:
1. 增加超时时间 (当前: 1000ms → 3000ms)
2. 降低并发数 (当前: 200 → 100)  
3. 跳过此目标继续
4. 暂停并检查网络

选择 (1-4): > 1

✅ 超时调整为 3000ms，继续扫描...
```

### 中断和恢复

**用户中断** (Ctrl+C):
```
🛑 接收到中断信号

当前状态: Port Scan 进行中 (65% 完成)
已扫描: 325/500 端口组合
发现: 8个开放端口

选择操作:  
1. 保存当前进度并退出
2. 完成当前步骤后退出
3. 立即强制退出 (不保存)
4. 继续运行

选择 (1-4): > 1

💾 保存进度中...
✅ 进度已保存: ~/.netcrate/runs/2025-08-29-1430/checkpoint.json

恢复方式: 
netcrate output resume --run 2025-08-29-1430
```

**任务恢复**:
```bash
netcrate output resume --run 2025-08-29-1430
```
```
[任务恢复] 2025-08-29-1430

上次运行状态:
✅ 主机发现: 已完成 (18台主机)
🔄 端口扫描: 进行中 (65% 完成)
⏸️  服务探测: 待开始

恢复选项:
1. 从中断点继续 (推荐)
2. 重新开始端口扫描  
3. 跳到下一步骤
4. 查看已完成结果

选择 (1-4): > 1

🔄 从端口扫描 65% 处恢复...
剩余目标: 175/500 端口组合
```

---

## 性能和兼容性契约

### 响应时间标准

**交互响应**:
- 按键响应: < 100ms  
- 界面切换: < 200ms
- 命令启动: < 500ms
- 大型结果显示: < 1s

**操作执行**:
- 网卡检测: < 2s
- CIDR 推导: < 100ms  
- 配置验证: < 500ms
- 结果导出: < 5s (中等数据量)

### 资源使用限制

**内存使用**:
- 基础运行: < 50MB
- 大规模扫描: < 200MB  
- 结果缓存: < 100MB

**磁盘空间**:
- 安装大小: < 20MB
- 单次运行: < 10MB
- 历史数据: 用户可配置清理

### 平台兼容性

**macOS** (主要支持):
- macOS 10.15+ (Catalina)
- 原生 arm64 + x86_64
- 系统权限集成

**Linux** (计划支持):
- 主流发行版 (Ubuntu, CentOS, Debian)
- Raw socket 权限处理

**Windows** (未来支持):
- Windows 10+ 
- PowerShell 集成

---

**契约版本**: v1.0  
**最后更新**: 2025-08-28  
**实现状态**: 设计完成，待开发验证