# Crypto Price Alert - 操作手册

## 1. 项目简介

本服务定时轮询币安（Binance）K 线数据，监控指定交易对在配置的时间窗口内的价格变化。当涨跌幅达到阈值时，通过飞书群机器人 Webhook 发送交互卡片告警。

**核心特性：**
- 每个交易对独立配置监控时间窗口和涨跌幅阈值
- 飞书交互卡片通知，涨绿跌红，附带 Binance 交易链接
- 同一交易对告警冷却去重，避免重复打扰
- 支持 HTTP 代理访问币安 API

---

## 2. 环境要求

| 依赖 | 版本要求 | 说明 |
|------|---------|------|
| Go   | >= 1.22 | 编译项目（运行编译好的 exe 不需要） |
| 网络 | 能访问 api.binance.com | 国内需要配置 HTTP 代理 |
| 飞书 | 群机器人 Webhook URL | 在飞书群设置中添加自定义机器人获取 |

---

## 3. 安装与编译

### 3.1 安装 Go（如果尚未安装）

**方式一：Chocolatey（管理员终端）**
```powershell
choco install golang -y
```

**方式二：手动安装**
从 https://go.dev/dl/ 下载 Windows amd64 安装包，运行安装即可。

安装完成后验证：
```bash
go version
# 输出示例：go version go1.26.2 windows/amd64
```

### 3.2 编译项目

```bash
cd D:/github/crypto-price-alert

# 下载依赖（国内使用 goproxy.cn 镜像）
GOPROXY=https://goproxy.cn,direct go mod tidy

# 编译
go build -o crypto-price-alert.exe .
```

编译成功后会在项目目录生成 `crypto-price-alert.exe`。

---

## 4. 配置说明

复制示例配置文件后编辑：

```bash
cp config.example.yaml config.yaml
```

### 4.1 完整配置文件示例

```yaml
# ==================== 交易对配置 ====================
# 每个交易对独立设置：交易对名称、时间窗口（分钟）、涨跌幅阈值（%）
symbols:
  - symbol: BTCUSDT       # 交易对名称（币安格式）
    window_minutes: 15     # 监控 15 分钟内的价格变化
    threshold_pct: 2.0     # 涨跌幅 >= 2% 触发告警

  - symbol: ETHUSDT
    window_minutes: 5
    threshold_pct: 3.0

  - symbol: SOLUSDT
    window_minutes: 5
    threshold_pct: 5.0

  - symbol: BNBUSDT
    window_minutes: 10
    threshold_pct: 3.0

# ==================== 轮询设置 ====================
monitor:
  poll_interval_seconds: 30   # 每 30 秒查询一次（建议 15~60）

# ==================== 告警设置 ====================
alert:
  cooldown_minutes: 10        # 同一交易对触发告警后，10 分钟内不再重复告警

# ==================== 飞书 Webhook ====================
feishu:
  webhook_url: "https://open.feishu.cn/open-apis/bot/v2/hook/你的webhook-id"
  secret: ""                  # 如果机器人开启了签名校验，填写签名密钥

# ==================== 币安 API ====================
binance:
  base_url: "https://api.binance.com"
  timeout_seconds: 10         # 单次请求超时时间

# ==================== HTTP 代理 ====================
# 国内网络无法直接访问币安 API，需要配置代理
proxy:
  http: "http://127.0.0.1:7890"   # 留空则不使用代理
```

### 4.2 各字段说明

#### symbols（交易对列表）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `symbol` | string | 是 | 币安交易对名称，如 `BTCUSDT`、`ETHUSDT` |
| `window_minutes` | int | 否 | 监控时间窗口（分钟），默认 5 |
| `threshold_pct` | float | 否 | 涨跌幅阈值（%），默认 3.0 |

**配置示例理解：**
```yaml
- symbol: BTCUSDT
  window_minutes: 15
  threshold_pct: 2.0
```
含义：监控 BTC/USDT 交易对，当 **15 分钟** 内涨跌幅 **>= 2%** 时发送飞书告警。

#### 常用交易对参考

| 交易对 | 说明 |
|--------|------|
| BTCUSDT | 比特币 |
| ETHUSDT | 以太坊 |
| BNBUSDT | 币安币 |
| SOLUSDT | Solana |
| DOGEUSDT | 狗狗币 |
| XRPUSDT | 瑞波币 |
| ADAUSDT | 卡尔达诺 |
| AVAXUSDT | 雪崩协议 |
| DOTUSDT | 波卡 |
| MATICUSDT | Polygon |

### 4.3 获取飞书 Webhook URL

1. 打开飞书，进入目标群聊
2. 点击群名称 → **设置** → **群机器人**
3. 点击 **添加机器人** → 选择 **自定义机器人**
4. 填写机器人名称（如 "币价监控"），点击 **添加**
5. 复制生成的 **Webhook 地址**，填入 `config.yaml` 的 `feishu.webhook_url` 字段
6. 如果开启了 **签名校验**，将密钥填入 `feishu.secret` 字段

---

## 5. 启动与运行

### 5.1 前台运行（调试模式）

```bash
cd D:/github/crypto-price-alert
./crypto-price-alert.exe -config config.yaml -debug
```

`-debug` 会输出每次轮询的详细价格数据，适合首次验证配置是否正确。

**正常启动日志示例：**
```
level=INFO msg="config loaded" symbols=4 poll_interval=30
level=INFO msg=monitoring symbol=BTCUSDT window=15 threshold=2
level=INFO msg=monitoring symbol=ETHUSDT window=5 threshold=3
level=INFO msg=monitoring symbol=SOLUSDT window=5 threshold=5
level=INFO msg=monitoring symbol=BNBUSDT window=10 threshold=3
level=INFO msg="monitor started" symbols=4 poll_interval=30s cooldown=10m0s
level=DEBUG msg="price check" symbol=BTCUSDT window=15 open=71712.5 close=71749.59 change_pct=0.05
```

### 5.2 前台运行（生产模式）

```bash
./crypto-price-alert.exe -config config.yaml
```

不加 `-debug`，只输出 INFO 级别日志（启动信息和告警触发记录）。

### 5.3 后台运行（Windows）

**方式一：nohup 方式（Git Bash / MSYS2）**
```bash
nohup ./crypto-price-alert.exe -config config.yaml > alert.log 2>&1 &
echo $!    # 记录 PID，后续可用 kill 停止
```

**方式二：PowerShell 后台任务**
```powershell
Start-Process -FilePath "D:\github\crypto-price-alert\crypto-price-alert.exe" `
  -ArgumentList "-config", "D:\github\crypto-price-alert\config.yaml" `
  -WindowStyle Hidden `
  -RedirectStandardOutput "D:\github\crypto-price-alert\alert.log" `
  -RedirectStandardError "D:\github\crypto-price-alert\error.log"
```

**方式三：注册为 Windows 服务（推荐长期运行）**
```powershell
# 使用 NSSM (Non-Sucking Service Manager) 注册
choco install nssm -y

nssm install CryptoPriceAlert "D:\github\crypto-price-alert\crypto-price-alert.exe"
nssm set CryptoPriceAlert AppParameters "-config D:\github\crypto-price-alert\config.yaml"
nssm set CryptoPriceAlert AppDirectory "D:\github\crypto-price-alert"
nssm set CryptoPriceAlert AppStdout "D:\github\crypto-price-alert\alert.log"
nssm set CryptoPriceAlert AppStderr "D:\github\crypto-price-alert\error.log"

# 启动服务
nssm start CryptoPriceAlert

# 查看状态
nssm status CryptoPriceAlert

# 停止服务
nssm stop CryptoPriceAlert

# 卸载服务
nssm remove CryptoPriceAlert confirm
```

### 5.4 停止服务

- **前台运行：** 按 `Ctrl+C`，程序会优雅退出
- **后台 nohup：** `kill <PID>`
- **Windows 服务：** `nssm stop CryptoPriceAlert`

---

## 6. 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-config` | `config.yaml` | 配置文件路径 |
| `-debug` | `false` | 开启 DEBUG 级别日志 |

```bash
# 指定配置文件路径
./crypto-price-alert.exe -config /path/to/my-config.yaml

# 开启调试日志
./crypto-price-alert.exe -config config.yaml -debug
```

---

## 7. 飞书告警卡片说明

当涨跌幅触发阈值时，飞书群会收到如下格式的交互卡片：

```
┌──────────────────────────────────────┐
│  Price Alert: BTCUSDT +2.35% (15min) │  <- 绿色标题头（涨）
├──────────────────────────────────────┤
│  交易对:     BTCUSDT                  │
│  涨跌幅:     +2.35%（15 分钟）        │
│  当前价:     $73,397.50               │
│  窗口起始价: $71,712.50               │
│  时间:       2026-04-08 16:30:00 CST  │
│  阈值:       >=2.0%                   │
├──────────────────────────────────────┤
│  [ 查看 Binance ]                    │  <- 点击跳转币安交易页
└──────────────────────────────────────┘
```

- 价格上涨：卡片标题为 **绿色**
- 价格下跌：卡片标题为 **红色**

---

## 8. 验证与测试

### 8.1 验证币安 API 连通性

启动后观察日志中是否出现 `price check` 记录。如果看到：
```
level=ERROR msg="fetch klines failed" ... error="context deadline exceeded"
```
说明网络不通，请检查代理配置。

### 8.2 测试飞书通知

临时将某个交易对的阈值改为极��值（如 `0.01`），重启服务：

```yaml
symbols:
  - symbol: BTCUSDT
    window_minutes: 15
    threshold_pct: 0.01    # 临时改小，几乎一定会触发
```

```bash
./crypto-price-alert.exe -config config.yaml -debug
```

观察日志出现 `feishu alert sent` 即为成功，同时检查飞书群是否收到卡片消息。

测试完毕后 **务必将阈值改回正常值**。

### 8.3 常见问题排查

| 现象 | 可能原因 | 解决办法 |
|------|---------|---------|
| `fetch klines failed: context deadline exceeded` | 无法访问币安 API | 检查代理配置，确认代理服务正在运行 |
| `feishu API error 400` | Webhook URL 错误 | 确认 URL 完整且未过期 |
| `feishu error code 19021` | 签名校验失败 | 检查 `secret` 是否正确，或关闭机器人签名校验 |
| 程序启动后无任何输出 | 配置文件路径错误 | 检查 `-config` 参数指向的文件是否存在 |
| 触发告警但飞书没收到 | 冷却时间内 | 检查日志中是否有 `alert suppressed by cooldown` |

---

## 9. 项目文件结构

```
D:\github\crypto-price-alert\
├── crypto-price-alert.exe       # 编译产物
├── config.yaml                  # 运行配置（含敏感信息，不提交 git）
├── config.example.yaml          # 配置模板
├── .gitignore
├── go.mod / go.sum              # Go 依赖管理
├── main.go                      # 程序入口
└── internal/
    ├── config/config.go         # 配置解��与校验
    ├── binance/client.go        # 币安 K 线 API 客户端
    ├── price/tracker.go         # 价格滑动窗口与变化检测
    ├── alert/
    │   ├── feishu.go            # 飞书卡片构建与发送
    │   └── dedup.go             # 告警去重（冷却计时）
    └── monitor/scheduler.go     # 主轮询调度循环
```

---

## 10. 快速启动清单

```
1. [ ] 安装 Go >= 1.22
2. [ ] cd D:/github/crypto-price-alert
3. [ ] cp config.example.yaml config.yaml
4. [ ] 编辑 config.yaml：
       - 填写飞书 Webhook URL
       - 配置需要监控的交易对和阈值
       - 配置 HTTP 代理（国内网络必须）
5. [ ] GOPROXY=https://goproxy.cn,direct go mod tidy
6. [ ] go build -o crypto-price-alert.exe .
7. [ ] ./crypto-price-alert.exe -config config.yaml -debug
8. [ ] 确认日志中有 price check 输出
9. [ ] 测试飞书通知（临时改小阈值）
10.[ ] 改回正式阈值，后台运行
```
