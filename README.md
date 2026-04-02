# NetSentry

中文 | [English](#english)

NetSentry 是一个基于 Go + Wails 的 Windows 网络监控工具，适合长时间持续检测网络质量。
它会持续探测目标主机，记录丢包、超时、恢复等事件，并自动更新简洁的日报，方便排查宽带、路由或链路抖动问题。

## 中文

### 功能特点

- 使用 Wails 构建单进程桌面 GUI，不再额外弹 PowerShell 窗口
- 监控目标和时序参数保存在 `netsentry.json` 中，可持续修改
- 使用 Windows ICMP API，而不是反复启动 `ping.exe`
- 同时记录 `rtt` 和应用视角的 `total latency`
- 按天写入异常与恢复事件日志
- 自动生成日报，汇总平均 RTT、最大 RTT、丢包率、最大连续超时次数
- 日报采用“定时 + 关键事件”刷新策略，减少磁盘写入

### 为什么做这个项目

排查网络问题时，宽带支持人员通常会让用户执行 `ping www.baidu.com -t`，然后观察是否出现超时或丢包。
NetSentry 把这个手工流程变成了一个可以长期后台运行的小工具，适合你在打游戏、语音通话或长时间排查期间持续记录网络质量。

### 项目结构

- `main.go`：Wails 桌面入口
- `app.go`：Wails 后端应用，负责配置、监控生命周期、事件推送
- `frontend/`：Wails 前端界面
- `wails.json`：Wails 项目配置
- `internal/config`：配置文件加载、保存和校验
- `internal/monitor`：监控循环、统计与事件钩子
- `internal/icmp`：Windows ICMP 探测实现
- `internal/logging`：异常与恢复事件日志
- `internal/report`：日报生成与写入
- `internal/model`：共享类型与统计结构

### 指标说明

- `rtt`：Windows ICMP API 返回的网络往返时间
- `total latency`：应用完成一次探测的总耗时
- `loss rate`：失败探测占比
- `max consecutive timeouts`：当天出现的最长连续超时次数

### GUI 使用方式

启动桌面程序后，你可以在界面里修改：

- 目标主机
- 探测间隔
- 超时时间
- 日志目录
- 日报目录
- 汇总输出间隔
- 日报刷新间隔

点击“保存配置”会写入项目根目录下的 `netsentry.json`。
点击“启动监控”后，界面会实时显示状态、RTT、丢包率和事件日志。
点击“停止监控”会停止当前监控会话，不需要杀进程。

### 命令行模式

如果你想保留终端方式，也可以显式使用 `-cli`：

```powershell
go run . -cli
```

也可以临时覆盖配置文件中的值：

```powershell
go run . -cli -host 1.1.1.1 -interval 1s -timeout 2s
```

### 配置文件示例

```json
{
  "host": "www.baidu.com",
  "interval": "1s",
  "timeout": "2s",
  "log_dir": "logs",
  "report_dir": "reports",
  "summary_every": "30s",
  "report_every": "30s"
}
```

### 开发

安装前端依赖：

```powershell
cd frontend
npm.cmd install
```

运行测试：

```powershell
go test ./...
```

本地开发模式：

```powershell
wails dev
```

生产构建：

```powershell
wails build
```

构建产物默认位于：

```text
build/bin/NetSentry.exe
```

### 资源占用

默认配置下，NetSentry 每秒发送一个很小的 ICMP 探测包，对带宽影响可以忽略。
由于它直接使用 Windows ICMP API，所以比反复启动外部 `ping` 进程更轻量。

日报不会在每次探测后都重写。
默认只会在定时刷新、异常发生、异常恢复和程序退出时更新。

### 开源维护说明

- 当前桌面界面采用 Wails，前后端职责更清晰
- Windows 相关逻辑集中在 `internal/icmp`
- 当前代码以 Windows 和 IPv4 为主
- `logs`、`reports`、`netsentry.json` 和 `frontend/node_modules` 属于运行或构建产物

## English

NetSentry is a Windows network monitoring tool built with Go and Wails.
It continuously probes a target host, records packet loss and recovery events, and updates a compact daily report for troubleshooting broadband issues.

### Highlights

- Uses a single-process Wails desktop UI instead of spawning extra PowerShell windows
- Keeps target host and timing values in `netsentry.json`
- Uses the Windows ICMP API instead of repeatedly spawning `ping.exe`
- Records both `rtt` and application-side `total latency`
- Writes issue and recovery events to daily log files
- Generates a daily summary with average RTT, maximum RTT, loss rate, and max consecutive timeouts
- Keeps disk writes low by refreshing the report on a timer plus important events

### Project Structure

- `main.go`: Wails desktop entrypoint
- `app.go`: Wails backend application and monitor lifecycle
- `frontend/`: Wails frontend UI
- `wails.json`: Wails project configuration
- `internal/config`: config file loading, saving, and validation
- `internal/monitor`: monitoring loop, counters, and hooks
- `internal/icmp`: Windows ICMP probing implementation
- `internal/logging`: issue and recovery event logs
- `internal/report`: daily report generation
- `internal/model`: shared types and statistics

### Development

Install frontend dependencies:

```powershell
cd frontend
npm.cmd install
```

Run tests:

```powershell
go test ./...
```

Run desktop dev mode:

```powershell
wails dev
```

Build production app:

```powershell
wails build
```

The built application is placed at:

```text
build/bin/NetSentry.exe
```

## License

MIT. See [LICENSE](LICENSE).
