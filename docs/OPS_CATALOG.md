# NetCrate 原子操作目录 (OPS_CATALOG)

## 概览

本文档定义了 NetCrate 所有基础原子操作的输入输出规范、依赖关系、权限需求和错误处理策略。每个操作都是独立可测试的单元，可以单独执行或组合成复杂工作流。

**设计原则**:
- **原子性**: 每个操作职责单一，不可再分
- **幂等性**: 相同输入产生相同输出
- **可组合**: 操作可以串联组成复杂流程
- **错误透明**: 明确的错误类型和处理策略

---

## 核心操作清单

### 网络探测类
- **discover** - 主机发现和存活检测
- **scan_ports** - 端口扫描和服务检测
- **packet_send** - 自定义数据包发送

### 环境管理类  
- **netenv_detect** - 网络环境检测
- **compliance_check** - 合规性验证
- **rate_limit** - 速率限制控制

### 数据管理类
- **output_save** - 结果数据保存
- **output_load** - 历史数据加载
- **output_export** - 数据格式转换

### 模板系统类
- **template_parse** - 模板解析和验证
- **template_execute** - 模板执行引擎
- **template_compose** - 动态模板组合

---

## 详细操作规范

### discover (主机发现)

**功能描述**: 检测网络范围内的活跃主机

#### 输入规范
```yaml
inputs:
  targets: 
    type: string | []string
    description: 目标范围
    formats:
      - "auto"                    # 自动推导当前网段
      - "192.168.1.0/24"         # CIDR 格式
      - "192.168.1.1-100"        # IP 范围
      - "192.168.1.1,10.0.0.1"   # 逗号分隔列表
      - "file:targets.txt"        # 文件引用
      - "last:discover"           # 上次发现结果
    required: true
    
  methods:
    type: []string
    description: 探测方法
    options: ["icmp", "tcp", "arp", "udp"]
    default: ["icmp", "tcp"]
    
  rate_limit:
    type: int
    description: 发送速率 (packets/second)
    default: 100
    range: [1, 1000]
    
  timeout:
    type: duration
    description: 单个目标超时时间
    default: "1000ms"
    range: ["100ms", "30s"]
    
  concurrency:
    type: int  
    description: 最大并发数
    default: 200
    range: [1, 2000]
    
  tcp_ports:
    type: []int
    description: TCP 探测端口 (method=tcp时)
    default: [80, 443, 22]
    
  interface:
    type: string
    description: 指定网络接口
    default: "auto"
    
  resolve_hostnames:
    type: bool
    description: 是否解析主机名
    default: false
```

#### 输出规范
```yaml
outputs:
  success:
    type: object
    schema:
      run_id: string              # 运行唯一标识
      start_time: timestamp       # 开始时间
      end_time: timestamp         # 结束时间  
      duration: float             # 执行时长(秒)
      targets_input: string       # 原始输入
      targets_resolved: int       # 解析出的目标数
      hosts_discovered: int       # 发现的活跃主机数
      success_rate: float         # 成功率 (0.0-1.0)
      method_used: []string       # 实际使用的方法
      interface_used: string      # 使用的网络接口
      
      results: []object
        - host: string            # IP 地址
          status: enum            # "up", "down", "timeout", "error"
          rtt: float             # 响应时间(ms), null if down
          method: string         # 探测方法
          details: object        # 方法相关的详细信息
            icmp_seq: int        # ICMP 序列号
            tcp_port: int        # TCP 探测端口
            mac_address: string  # MAC 地址 (ARP)
          timestamp: timestamp   # 响应时间戳
          hostname: string       # 主机名 (如果解析)
          
      stats:
        sent: int                # 发送的包数
        received: int            # 收到的响应数
        errors: int              # 错误数
        timeouts: int            # 超时数
        method_breakdown:        # 按方法统计
          icmp: {sent: int, received: int}
          tcp: {sent: int, received: int}
          arp: {sent: int, received: int}
          
  error:
    type: object
    schema:
      error_type: enum          # 错误类型
      error_message: string     # 错误描述
      partial_results: object   # 部分结果 (如果有)
```

#### 错误类型与处理
```yaml
error_handling:
  permission_denied:
    description: "Raw socket 权限不足"
    mitigation: "回退到 TCP connect 方式"
    user_action: "建议使用 sudo 或调整方法"
    
  network_unreachable:
    description: "网络不可达"
    mitigation: "跳过不可达的网段"
    user_action: "检查网络连接和路由"
    
  invalid_target_format:
    description: "目标格式错误"
    mitigation: "停止执行"
    user_action: "修正目标格式"
    
  rate_limit_exceeded:
    description: "速率超出系统限制"
    mitigation: "自动调整到最大允许值"
    user_action: "可以手动调整 --profile"
    
  interface_not_found:
    description: "指定网络接口不存在"
    mitigation: "回退到自动选择"
    user_action: "检查接口名称"
    
  timeout_threshold:
    description: "大量目标超时 (>50%)"
    mitigation: "建议用户确认后继续"
    user_action: "检查网络状况或调整超时时间"
```

#### 权限需求
```yaml
permissions:
  raw_socket:
    required_for: ["icmp", "arp"]
    fallback: "tcp connect"
    platforms:
      darwin: "sudo required"
      linux: "CAP_NET_RAW or root"
      windows: "Administrator"
      
  network_interface:
    required_for: "interface selection"
    fallback: "use default route interface"
    
  dns_resolution:
    required_for: "hostname lookup"
    fallback: "show IP only"
```

#### 性能特征
```yaml
performance:
  memory_usage: "~2MB base + 100B per target"
  cpu_intensive: false
  network_intensive: true
  disk_io: minimal
  
  scaling:
    targets_1000: "~30s (100pps)"
    targets_10000: "~5min (100pps)"
    max_recommended_targets: 65535
    
  rate_limits:
    safe_profile: "100 pps, 200 concurrent"
    fast_profile: "400 pps, 800 concurrent"
    system_max: "1000 pps, 2000 concurrent"
```

---

### scan_ports (端口扫描)

**功能描述**: 扫描指定主机的端口开放状态和服务信息

#### 输入规范
```yaml
inputs:
  targets:
    type: []string
    description: 目标主机列表
    formats:
      - ["192.168.1.1", "192.168.1.2"]  # IP 列表
      - "last:discover"                  # 引用上次发现结果
      - "file:hosts.txt"                 # 文件引用
      - "192.168.1.0/24"                # CIDR (自动发现后扫描)
    required: true
    
  ports:
    type: string | []int
    description: 端口集合
    formats:
      - "top100"                    # 预定义端口集
      - "top1000"
      - "web"                       # [80,443,8080,8000,8443]
      - "database"                  # [3306,5432,1433,27017,6379]
      - "22,80,443"                 # 逗号分隔
      - "8000-9000"                 # 端口范围
      - "22,80,443,8000-9000"       # 混合格式
      - "file:ports.txt"            # 文件引用
    default: "top100"
    
  scan_type:
    type: enum
    options: ["syn", "connect", "udp", "auto"]
    description: 扫描类型
    default: "auto"
    
  service_detection:
    type: bool
    description: 是否进行服务识别
    default: true
    
  rate_limit:
    type: int
    description: 扫描速率 (packets/second)
    default: 100
    
  timeout:
    type: duration
    description: 单个端口超时
    default: "800ms"
    
  concurrency:
    type: int
    description: 最大并发连接数
    default: 200
    
  retry_count:
    type: int
    description: 失败重试次数
    default: 1
    range: [0, 5]
```

#### 输出规范
```yaml
outputs:
  success:
    type: object
    schema:
      run_id: string
      start_time: timestamp
      end_time: timestamp
      duration: float
      
      targets_count: int          # 目标主机数
      ports_per_target: int       # 每个主机扫描的端口数
      total_combinations: int     # 总的 (IP,Port) 组合数
      
      open_ports: int            # 开放端口总数
      closed_ports: int          # 关闭端口数
      filtered_ports: int        # 被过滤端口数
      
      scan_type_used: string     # 实际使用的扫描类型
      
      results: []object
        - host: string           # 目标 IP
          port: int              # 端口号
          status: enum           # "open", "closed", "filtered", "error"
          protocol: enum         # "tcp", "udp"
          rtt: float            # 响应时间(ms)
          service: object       # 服务信息 (如果检测)
            name: string        # 服务名 (http, ssh, mysql)
            version: string     # 版本信息
            banner: string      # 服务横幅
            confidence: float   # 识别置信度 (0.0-1.0)
          timestamp: timestamp
          
      stats:
        hosts_scanned: int
        ports_scanned: int
        success_rate: float
        avg_rtt: float
        scan_rate: float        # 实际扫描速率 (pps)
        
        by_status:
          open: int
          closed: int
          filtered: int
          error: int
          
        by_service:             # 服务统计
          http: int
          https: int
          ssh: int
          unknown: int
```

#### 错误处理
```yaml
error_handling:
  no_targets:
    description: "目标列表为空"
    mitigation: "提示用户先执行 discover"
    user_action: "提供有效的目标列表"
    
  invalid_port_format:
    description: "端口格式错误"
    mitigation: "停止执行"
    user_action: "修正端口格式"
    
  permission_denied:
    description: "SYN 扫描需要 raw socket 权限"
    mitigation: "自动回退到 connect 扫描"
    user_action: "使用 sudo 或接受 connect 模式"
    
  connection_refused:
    description: "大量连接被拒绝"
    mitigation: "标记为 closed，继续扫描"
    user_action: "可能目标有防火墙"
    
  rate_limit_hit:
    description: "触发目标速率限制"
    mitigation: "自动降低扫描速率"
    user_action: "考虑使用更保守的配置"
    
  host_unreachable:
    description: "目标主机不可达"
    mitigation: "跳过该主机，继续其他目标"
    user_action: "检查目标是否在线"
```

#### 权限需求
```yaml
permissions:
  syn_scan:
    required_for: "TCP SYN 扫描"
    privilege: "raw socket"
    fallback: "TCP connect"
    
  connect_scan:
    required_for: "TCP connect 扫描"
    privilege: "normal user"
    fallback: "none"
    
  udp_scan:
    required_for: "UDP 扫描" 
    privilege: "raw socket"
    fallback: "limited UDP connect"
    
  service_detection:
    required_for: "服务横幅获取"
    privilege: "normal user"
    fallback: "port status only"
```

---

### packet_send (数据包发送)

**功能描述**: 发送自定义数据包并收集响应

#### 输入规范
```yaml
inputs:
  targets:
    type: []string
    description: 目标端点 (IP:Port)
    formats:
      - ["192.168.1.1:80", "192.168.1.2:443"]
      - "last:scan"              # 引用扫描结果的开放端口
      - "file:endpoints.txt"     # 文件引用
    required: true
    
  template:
    type: string
    description: 数据包模板
    options:
      - "syn"         # TCP SYN 探测
      - "connect"     # TCP 完整连接
      - "http"        # HTTP 请求
      - "https"       # HTTPS 请求
      - "tls"         # TLS 握手
      - "dns"         # DNS 查询
      - "icmp"        # ICMP 包
      - "udp"         # UDP 探测
      - "custom"      # 自定义模板
    required: true
    
  template_params:
    type: map[string]string
    description: 模板参数
    examples:
      http:
        method: "GET"
        path: "/admin"
        user_agent: "NetCrate/1.0"
        headers: "Authorization: Bearer token"
      https:
        sni: "example.com"
        verify_cert: "false"
      dns:
        domain: "example.com"
        query_type: "A"
        server: "8.8.8.8"
      tls:
        sni: "example.com"
        version: "1.3"
    default: {}
    
  count:
    type: int
    description: 每个目标发送次数
    default: 1
    range: [1, 100]
    
  interval:
    type: duration
    description: 发送间隔
    default: "100ms"
    
  timeout:
    type: duration
    description: 响应超时时间
    default: "5s"
    
  follow_redirects:
    type: bool
    description: 是否跟随 HTTP 重定向
    default: false
    
  max_response_size:
    type: int
    description: 最大响应大小 (bytes)
    default: 1048576  # 1MB
```

#### 输出规范
```yaml
outputs:
  success:
    type: object
    schema:
      run_id: string
      template_used: string
      targets_count: int
      total_packets: int
      successful_responses: int
      
      results: []object
        - target: string          # IP:Port
          sequence: int           # 发送序号
          status: enum            # "success", "timeout", "error"
          rtt: float             # 响应时间(ms)
          
          request: object         # 请求详情
            method: string        # HTTP method / packet type
            headers: map          # 请求头 (HTTP)
            body_size: int        # 请求体大小
            
          response: object        # 响应详情
            status_code: int      # HTTP 状态码
            headers: map          # 响应头
            body_preview: string  # 响应体预览 (前1KB)
            body_size: int        # 完整响应体大小
            tls_version: string   # TLS 版本 (HTTPS)
            cert_info: object     # 证书信息
              subject: string
              issuer: string
              expires: timestamp
            
          error: object          # 错误信息 (status=error时)
            type: string
            message: string
            
          timestamp: timestamp
          
      stats:
        avg_rtt: float
        min_rtt: float
        max_rtt: float
        success_rate: float
        
        by_status_code:          # HTTP 状态码分布
          "200": int
          "404": int
          "500": int
          
        by_template:             # 模板使用统计
          http: {count: int, success: int}
          tls: {count: int, success: int}
```

#### 模板定义
```yaml
templates:
  syn:
    description: "TCP SYN 探测包"
    required_params: []
    optional_params: ["tcp_flags", "window_size"]
    requires_raw_socket: true
    
  http:
    description: "HTTP 请求"
    required_params: []
    optional_params: ["method", "path", "headers", "body", "user_agent"]
    requires_raw_socket: false
    default_params:
      method: "GET"
      path: "/"
      user_agent: "NetCrate/1.0"
      
  https:
    description: "HTTPS 请求"
    required_params: []
    optional_params: ["method", "path", "headers", "sni", "verify_cert"]
    requires_raw_socket: false
    
  tls:
    description: "TLS 握手探测"
    required_params: []
    optional_params: ["sni", "version", "ciphers"]
    requires_raw_socket: false
    
  dns:
    description: "DNS 查询"
    required_params: ["domain"]
    optional_params: ["query_type", "server"]
    requires_raw_socket: false
    default_params:
      query_type: "A"
      server: "8.8.8.8"
      
  icmp:
    description: "ICMP ping"
    required_params: []
    optional_params: ["type", "code", "payload"]
    requires_raw_socket: true
    default_params:
      type: "echo"
```

#### 错误处理
```yaml
error_handling:
  invalid_template:
    description: "不支持的模板类型"
    mitigation: "列出可用模板"
    user_action: "使用有效的模板名"
    
  missing_required_params:
    description: "缺少必需的模板参数"
    mitigation: "提示缺少的参数"
    user_action: "补充必需参数"
    
  connection_failed:
    description: "无法连接到目标"
    mitigation: "标记为错误，继续其他目标"
    user_action: "检查目标可达性"
    
  response_too_large:
    description: "响应超过大小限制"
    mitigation: "截断响应，记录实际大小"
    user_action: "可调整 max_response_size"
    
  ssl_handshake_failed:
    description: "TLS/SSL 握手失败"
    mitigation: "记录错误详情"
    user_action: "检查证书或 SNI 配置"
    
  dns_resolution_failed:
    description: "域名解析失败"
    mitigation: "跳过该目标"
    user_action: "检查域名是否有效"
```

---

### netenv_detect (网络环境检测)

**功能描述**: 检测当前网络环境和接口配置

#### 输入规范
```yaml
inputs:
  interface_filter:
    type: string
    description: 接口名称过滤
    formats:
      - "auto"        # 自动选择最佳接口
      - "en0"         # 指定接口名
      - "eth*"        # 通配符匹配
      - "all"         # 显示所有接口
    default: "auto"
    
  include_loopback:
    type: bool
    description: 是否包含回环接口
    default: false
    
  include_inactive:
    type: bool  
    description: 是否包含非活跃接口
    default: false
    
  resolve_gateways:
    type: bool
    description: 是否解析网关信息
    default: true
    
  ping_test:
    type: bool
    description: 是否测试接口连通性
    default: false
```

#### 输出规范
```yaml
outputs:
  success:
    type: object
    schema:
      interfaces: []object
        - name: string          # 接口名 (en0, eth0)
          display_name: string  # 友好名称
          mac_address: string   # MAC 地址
          mtu: int             # MTU 值
          status: enum         # "up", "down", "unknown"
          type: enum           # "ethernet", "wireless", "loopback", "vpn"
          
          addresses: []object   # IP 地址列表
            - ip: string        # IP 地址
              network: string   # 网络 CIDR
              netmask: string   # 子网掩码
              broadcast: string # 广播地址
              scope: enum       # "global", "link", "host"
              
          gateway: object       # 网关信息
            ip: string
            mac_address: string
            rtt: float         # ping 测试结果 (如果启用)
            
          stats: object         # 接口统计 (如果可用)
            bytes_sent: int
            bytes_received: int
            packets_sent: int
            packets_received: int
            errors: int
            drops: int
            
      recommended: string       # 推荐使用的接口名
      
      system_info:
        platform: string        # darwin, linux, windows
        hostname: string        # 主机名
        dns_servers: []string   # DNS 服务器列表
        default_route: string   # 默认路由接口
        
      capabilities:
        raw_socket: bool        # 是否支持 raw socket
        promiscuous_mode: bool  # 是否支持混杂模式
        packet_capture: bool    # 是否支持包捕获
```

#### 权限需求
```yaml
permissions:
  interface_enumeration:
    required_for: "列出网络接口"
    privilege: "normal user"
    
  interface_stats:
    required_for: "获取接口统计信息"
    privilege: "normal user (部分平台需要 admin)"
    
  gateway_discovery:
    required_for: "发现网关 MAC 地址"
    privilege: "raw socket (ARP)"
    fallback: "仅显示 IP"
    
  connectivity_test:
    required_for: "ping 测试"
    privilege: "ICMP (raw socket)"
    fallback: "TCP connect test"
```

---

### compliance_check (合规检查)

**功能描述**: 验证操作是否符合安全策略和法律要求

#### 输入规范
```yaml
inputs:
  targets:
    type: []string
    description: 要检查的目标列表
    required: true
    
  operation_type:
    type: enum
    options: ["discover", "scan", "packet", "template"]
    description: 操作类型
    required: true
    
  rate_limit:
    type: int
    description: 请求的速率限制
    required: true
    
  concurrency:
    type: int
    description: 请求的并发数
    required: true
    
  dangerous_flag:
    type: bool
    description: 用户是否使用了 --dangerous 标志
    default: false
    
  config_profile:
    type: string
    description: 当前配置档案
    default: "safe"
```

#### 输出规范
```yaml
outputs:
  success:
    type: object
    schema:
      compliant: bool           # 总体是否合规
      warnings: []string        # 警告信息
      errors: []string          # 错误信息
      
      checks: []object          # 检查详情
        - check_name: string    # 检查项名称
          status: enum          # "pass", "warn", "fail"
          message: string       # 详细信息
          recommendation: string # 建议操作
          
      target_analysis:
        total_targets: int
        private_targets: int    # 私网目标数
        public_targets: int     # 公网目标数
        invalid_targets: int    # 无效目标数
        
        private_ranges: []string  # 识别的私网范围
        public_targets_detail: []string  # 公网目标详情
        
      rate_analysis:
        requested_rate: int
        max_allowed_rate: int
        adjusted_rate: int      # 调整后的速率
        rate_compliant: bool
        
      recommendations: []object
        - priority: enum        # "high", "medium", "low"  
          action: string        # 建议的操作
          reason: string        # 建议原因
```

#### 检查规则
```yaml
compliance_rules:
  target_validation:
    private_networks_only:
      enabled: true
      description: "默认仅允许私网目标"
      private_ranges:
        - "10.0.0.0/8"
        - "172.16.0.0/12"  
        - "192.168.0.0/16"
        - "127.0.0.0/8"
      exceptions:
        - "需要 --dangerous 标志"
        - "交互确认"
        
    blocked_ranges:
      enabled: true
      description: "明确禁止的网络范围"
      ranges:
        - "0.0.0.0/8"          # 本网段
        - "169.254.0.0/16"     # 链路本地
        - "224.0.0.0/4"        # 组播
        - "240.0.0.0/4"        # 保留
        
    domain_validation:
      enabled: true
      description: "域名目标验证"
      rules:
        - "解析后的 IP 也要符合规则"
        - "警告动态 DNS 风险"
        
  rate_limits:
    safe_profile:
      max_rate: 100
      max_concurrency: 200
      description: "日常使用安全限制"
      
    fast_profile:  
      max_rate: 400
      max_concurrency: 800
      description: "专业测试限制"
      requires: "交互确认"
      
    custom_profile:
      max_rate: 1000
      max_concurrency: 2000  
      description: "专家自定义"
      requires: ["--dangerous", "交互确认"]
      
  operational_safety:
    first_time_warning:
      enabled: true
      description: "首次使用法律声明"
      action: "显示完整法律声明并要求确认"
      
    public_target_warning:
      enabled: true
      description: "公网目标额外警告"
      action: "显示法律风险并要求双重确认"
      
    resource_limits:
      max_targets_per_run: 65535
      max_ports_per_target: 65535
      max_concurrent_runs: 5
```

---

### output_save (结果保存)

**功能描述**: 保存运行结果到本地存储

#### 输入规范
```yaml
inputs:
  run_context:
    type: object
    description: 运行上下文信息
    schema:
      run_id: string          # 运行唯一标识
      command: string         # 执行的命令
      start_time: timestamp   # 开始时间
      user: string           # 执行用户
      hostname: string       # 执行主机
      version: string        # NetCrate 版本
      config_profile: string # 使用的配置档案
    required: true
    
  results_data:
    type: object
    description: 操作结果数据
    schema:
      discover: object       # discover 操作结果
      scan: object          # scan 操作结果  
      packet: object        # packet 操作结果
      template: object      # template 操作结果
    required: true
    
  workspace_dir:
    type: string
    description: 工作区目录
    default: "~/.netcrate/runs"
    
  formats:
    type: []string
    description: 保存格式
    options: ["json", "ndjson", "csv", "summary"]
    default: ["json", "summary"]
    
  compression:
    type: bool
    description: 是否压缩大文件
    default: true
    
  retention_policy:
    type: object
    description: 保留策略
    schema:
      max_days: int         # 最大保留天数
      max_runs: int         # 最大保留运行数
      auto_cleanup: bool    # 自动清理
    default: {max_days: 30, max_runs: 1000, auto_cleanup: true}
```

#### 输出规范  
```yaml
outputs:
  success:
    type: object
    schema:
      run_directory: string      # 运行结果目录
      files_created: []object    # 创建的文件列表
        - name: string           # 文件名
          path: string           # 完整路径
          format: string         # 文件格式
          size: int              # 文件大小 (bytes)
          compressed: bool       # 是否压缩
          
      summary:
        total_size: int          # 总大小
        file_count: int          # 文件数量
        compression_ratio: float # 压缩比 (如果有)
        
      metadata:
        created_at: timestamp
        expires_at: timestamp    # 过期时间 (基于保留策略)
        tags: []string          # 标签 (方便搜索)
```

#### 文件结构
```yaml
file_structure:
  run_directory: "~/.netcrate/runs/{run_id}/"
  
  files:
    "summary.json":
      description: "运行摘要和统计"
      format: "structured JSON"
      content: "高层统计信息"
      
    "results.ndjson":  
      description: "详细结果数据"
      format: "newline-delimited JSON"
      content: "每行一个结果记录"
      
    "results.csv":
      description: "CSV 格式结果 (如果请求)"
      format: "comma-separated values"
      content: "扁平化的结果数据"
      
    "metadata.json":
      description: "运行元数据"
      format: "JSON"
      content: "命令、参数、环境信息"
      
    "logs.txt":
      description: "执行日志 (如果启用)"
      format: "plain text"  
      content: "详细的执行日志"
```

#### 错误处理
```yaml
error_handling:
  disk_full:
    description: "磁盘空间不足"
    mitigation: "尝试保存到 /tmp，提示用户"
    user_action: "清理磁盘空间或指定其他目录"
    
  permission_denied:
    description: "目录写权限不足"
    mitigation: "尝试保存到用户主目录"
    user_action: "检查目录权限或指定其他目录"
    
  invalid_workspace:
    description: "工作区目录不存在或不可写"
    mitigation: "创建目录或使用默认位置"
    user_action: "指定有效的工作区路径"
    
  serialization_error:
    description: "数据序列化失败"
    mitigation: "尝试简化数据结构"
    user_action: "检查结果数据是否有效"
    
  cleanup_failed:
    description: "自动清理失败"
    mitigation: "记录错误，继续执行"
    user_action: "可手动清理旧文件"
```

---

### output_load (数据加载)

**功能描述**: 加载历史运行结果

#### 输入规范
```yaml
inputs:
  run_id:
    type: string
    description: 运行标识符
    formats:
      - "2025-08-29-1430"      # 完整运行 ID
      - "last"                 # 最近一次运行
      - "last:discover"        # 最近一次特定操作
    required: true
    
  data_types:
    type: []string
    description: 要加载的数据类型
    options: ["summary", "results", "metadata", "logs"]
    default: ["summary", "results"]
    
  filters:
    type: object
    description: 数据过滤条件
    schema:
      status: []string         # 按状态过滤 ["up", "open"]
      hosts: []string          # 按主机过滤
      ports: []int            # 按端口过滤
      time_range: object      # 时间范围
        start: timestamp
        end: timestamp
    optional: true
    
  workspace_dir:
    type: string
    description: 工作区目录
    default: "~/.netcrate/runs"
```

#### 输出规范
```yaml
outputs:
  success:
    type: object
    schema:
      run_info:
        run_id: string
        command: string
        start_time: timestamp
        end_time: timestamp
        status: enum           # "completed", "failed", "interrupted"
        
      data:
        summary: object        # 摘要数据 (如果请求)
        results: []object      # 结果数据 (如果请求)
        metadata: object       # 元数据 (如果请求)  
        logs: []string        # 日志行 (如果请求)
        
      statistics:
        total_records: int     # 总记录数
        filtered_records: int  # 过滤后记录数
        file_sizes: map        # 各文件大小
        load_time: float      # 加载耗时
```

---

### output_export (数据导出)

**功能描述**: 将结果数据导出为指定格式

#### 输入规范
```yaml
inputs:
  run_id:
    type: string
    description: 要导出的运行 ID
    required: true
    
  output_path:
    type: string  
    description: 导出文件路径
    required: true
    
  format:
    type: enum
    options: ["json", "csv", "html", "xml", "txt"]
    description: 导出格式
    required: true
    
  template:
    type: string
    description: 导出模板 (HTML格式时)
    options: ["default", "detailed", "summary", "custom"]
    default: "default"
    
  include_sections:
    type: []string
    description: 包含的数据节
    options: ["summary", "hosts", "ports", "services", "errors"]
    default: ["summary", "hosts", "ports", "services"]
    
  filters:
    type: object
    description: 导出过滤条件
    optional: true
```

#### 输出规范
```yaml
outputs:
  success:
    type: object
    schema:
      output_file: string      # 导出文件路径
      format: string          # 实际使用的格式
      file_size: int          # 文件大小
      records_exported: int   # 导出记录数
      compression_used: bool  # 是否使用压缩
      export_time: float     # 导出耗时
```

---

### template_parse (模板解析)

**功能描述**: 解析和验证 YAML 模板文件

#### 输入规范
```yaml
inputs:
  template_source:
    type: string
    description: 模板来源
    formats:
      - "builtin:basic_scan"    # 内置模板
      - "file:my_template.yaml" # 文件路径
      - "url:https://..."       # URL 引用
    required: true
    
  validation_level:
    type: enum
    options: ["strict", "normal", "permissive"]  
    description: 验证严格程度
    default: "normal"
    
  resolve_includes:
    type: bool
    description: 是否解析包含的子模板
    default: true
```

#### 输出规范
```yaml
outputs:
  success:
    type: object
    schema:
      template: object         # 解析后的模板对象
        name: string
        version: string
        description: string
        author: string
        tags: []string
        
        parameters: []object   # 参数定义
          - name: string
            type: string
            required: bool
            default: any
            description: string
            validation: object
            
        steps: []object        # 执行步骤
          - name: string
            operation: string
            inputs: object
            outputs: object
            depends_on: []string
            error_handling: object
            
      validation_result:
        valid: bool
        warnings: []string
        errors: []string
        
      dependencies: []string   # 依赖的其他模板/操作
```

---

### template_execute (模板执行)

**功能描述**: 执行解析后的模板工作流

#### 输入规范
```yaml
inputs:
  template:
    type: object
    description: 解析后的模板对象
    required: true
    
  parameters:
    type: map[string]any
    description: 模板参数值
    required: true
    
  execution_mode:
    type: enum
    options: ["interactive", "batch", "dry_run"]
    description: 执行模式
    default: "interactive"
    
  step_filters:
    type: []string
    description: 要执行的步骤 (空表示全部)
    default: []
    
  continue_on_error:
    type: bool
    description: 遇到错误时是否继续
    default: false
```

#### 输出规范
```yaml
outputs:
  success:
    type: object
    schema:
      execution_id: string     # 执行唯一标识
      template_name: string
      total_steps: int
      completed_steps: int
      failed_steps: int
      skipped_steps: int
      
      step_results: []object   # 步骤执行结果
        - step_name: string
          status: enum         # "success", "failed", "skipped"
          start_time: timestamp
          end_time: timestamp
          duration: float
          output: object       # 步骤输出
          error: object       # 错误信息 (如果有)
          
      final_output: object     # 最终聚合输出
      execution_summary: object # 执行统计摘要
```

---

## 操作依赖关系

### 依赖图
```yaml
dependencies:
  discover:
    depends_on: []
    provides: ["host_list"]
    
  scan_ports:
    depends_on: ["host_list"]
    provides: ["port_list", "service_info"]
    
  packet_send:
    depends_on: ["port_list"] # 可选
    provides: ["response_data"]
    
  netenv_detect:
    depends_on: []
    provides: ["interface_info"]
    
  compliance_check:
    depends_on: []
    provides: ["compliance_result"]
    
  output_save:
    depends_on: ["任何操作结果"]
    provides: ["saved_run_id"]
    
  output_load:
    depends_on: ["saved_run_id"]
    provides: ["historical_data"]
    
  template_parse:
    depends_on: []
    provides: ["parsed_template"]
    
  template_execute:
    depends_on: ["parsed_template"]
    provides: ["workflow_result"]
```

### 数据流
```yaml
data_flow:
  typical_quick_mode:
    1. netenv_detect → interface_selection
    2. compliance_check(targets) → approved_targets  
    3. discover(approved_targets) → live_hosts
    4. scan_ports(live_hosts) → open_ports
    5. packet_send(open_ports) → response_data
    6. output_save(all_results) → run_id
    
  ops_mode_chaining:
    # 每个操作独立，但可以引用前面的结果
    netcrate ops discover 192.168.1.0/24
    netcrate ops scan ports --targets last:discover
    netcrate ops packet send --targets last:scan --template http
    
  template_mode:
    1. template_parse(template_file) → parsed_template
    2. template_execute(parsed_template, params) → workflow_result
    3. output_save(workflow_result) → run_id
```

---

## 性能和资源规范

### 内存使用估算
```yaml
memory_usage:
  base_overhead: "10-20MB"
  
  per_operation:
    discover: "100B per target"
    scan_ports: "50B per (host,port) combination"  
    packet_send: "1KB per response (with body preview)"
    output_save: "minimal (streaming write)"
    
  maximum_limits:
    max_targets: 65535        # IPv4 地址空间限制
    max_concurrent: 2000      # 系统资源限制
    max_response_size: "1MB"  # 单个响应大小
    max_total_memory: "500MB" # 整个程序内存限制
```

### CPU 和网络特征
```yaml
performance_characteristics:
  cpu_intensive:
    - template_parse (YAML parsing)
    - compliance_check (规则匹配)
    
  network_intensive:
    - discover (大量 ICMP/TCP)
    - scan_ports (大量并发连接)
    - packet_send (自定义协议)
    
  io_intensive:
    - output_save (磁盘写入)
    - output_load (磁盘读取)
    - output_export (格式转换)
    
  memory_intensive:
    - packet_send (响应缓存)
    - template_execute (中间结果)
```

---

## 错误代码规范

### 统一错误代码
```yaml
error_codes:
  # 通用错误 (1000-1999)
  1000: "UNKNOWN_ERROR"
  1001: "INVALID_INPUT"
  1002: "PERMISSION_DENIED" 
  1003: "RESOURCE_EXHAUSTED"
  1004: "TIMEOUT"
  1005: "NETWORK_ERROR"
  
  # 合规错误 (2000-2099)
  2000: "COMPLIANCE_VIOLATION"
  2001: "PUBLIC_TARGET_BLOCKED"
  2002: "RATE_LIMIT_EXCEEDED"
  2003: "DANGEROUS_OPERATION"
  
  # 发现错误 (2100-2199)  
  2100: "DISCOVER_NO_INTERFACE"
  2101: "DISCOVER_NO_TARGETS"
  2102: "DISCOVER_ALL_TIMEOUT"
  
  # 扫描错误 (2200-2299)
  2200: "SCAN_NO_HOSTS"
  2201: "SCAN_INVALID_PORTS"
  2202: "SCAN_CONNECTION_FAILED"
  
  # 包发送错误 (2300-2399)
  2300: "PACKET_INVALID_TEMPLATE"
  2301: "PACKET_MISSING_PARAMS"
  2302: "PACKET_SEND_FAILED"
  
  # 输出错误 (2400-2499)
  2400: "OUTPUT_DISK_FULL"
  2401: "OUTPUT_PERMISSION_DENIED"
  2402: "OUTPUT_INVALID_FORMAT"
  
  # 模板错误 (2500-2599)
  2500: "TEMPLATE_PARSE_ERROR"
  2501: "TEMPLATE_VALIDATION_ERROR"
  2502: "TEMPLATE_EXECUTION_ERROR"
```

---

**文档版本**: 1.0  
**最后更新**: 2024-08-29  
**实现状态**: 设计完成，待开发实现

这份操作目录为每个原子操作提供了精确的接口定义，是实现阶段的重要参考文档。所有操作都遵循统一的输入输出格式、错误处理机制和权限模型。