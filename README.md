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
- `max consecutive timeouts`：界面显示本次监控的最长连续超时次数，日报显示当天的最长连续超时次数

界面统计从本次启动监控开始累计；日报仅统计本次会话在对应自然日内的探测。
连续运行跨午夜时会切换日志、日报文件，并重置日报计数，界面的会话计数继续累计。
目前停止后重新启动监控不会续算之前的日报，同一天的日报会被新会话更新；异常日志始终追加保留。

### GUI 使用方式

#### 下载安装

在 GitHub 仓库的 **Releases** 页面下载 `NetSentry-windows-amd64.exe`，放到有写入权限的文件夹后双击即可运行（Windows 10/11 x64）。
不需要安装 Go、Node.js 或另行下载前端文件；界面资源已内置在 exe 中。
程序需要 Microsoft WebView2 Runtime。如果未安装，程序会提示联网下载安装，详见 [Wails 的运行时说明](https://v2.wails.io/docs/guides/windows/)。
配置、日志和日报在运行时生成，无需随 exe 一起下载；不要放到只读目录。
当前 exe 未进行代码签名，Windows 可能显示发布者未知或 SmartScreen 提示，请确认来源是本仓库的 Release。
附件 `SHA256SUMS.txt` 用于校验下载文件，`LICENSE` 为开源许可证。

启动桌面程序后，你可以在界面里修改：

- 目标主机
- 探测间隔
- 超时时间
- 日志目录
- 日报目录
- 汇总输出间隔
- 日报刷新间隔

点击“保存配置”会写入程序当前运行目录下的 `netsentry.json`。
点击“启动监控”后，界面会实时显示状态、RTT、丢包率和事件日志。
点击“停止监控”会停止当前监控会话，不需要杀进程。

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

从项目根目录运行测试（`go:embed` 需要先生成前端资源）：

```powershell
npm.cmd --prefix frontend ci
npm.cmd --prefix frontend run build
go test ./...
go vet ./...
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
默认只会在定时刷新、状态变化、跨日和程序退出时更新，连续相同的超时不会每次重写日报。
如果日志或日报写入失败，界面的 console 会显示错误信息。

### 自动发布 Windows 版本

普通推送到 `main` / `master` 或创建 PR 仍然只运行测试；推送以 `v` 开头的标签会触发 `Windows Release` 工作流。
它会先构建前端、运行 Go 测试和静态检查，再生成 Windows x64 exe，最后创建 GitHub Release 并附上 exe、许可证和 SHA-256 校验文件。
构建使用与 `go.mod` 相同版本的 Wails CLI，不需要手工上传 exe，也不需要额外配置个人访问令牌。

先提交并推送准备发布的代码（必须包含发布工作流），再从该提交创建一个尚未使用的版本标签，例如：

```powershell
git tag -a v1.0.0 -m "NetSentry v1.0.0"
git push origin v1.0.0
```

在 GitHub 的 **Actions** 页面查看进度，成功后到 **Releases** 下载附件。
`v1.0.0` 这样的标签发布正式版；含连字符的标签（例如 `v1.1.0-beta.1`）发布预发布版，不标记为 Latest。
以后发布请使用新的版本号，不要移动已发布标签。工作流不会覆盖已有 Release；测试或构建失败不会进入发布步骤。

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

### Download and Run

Download `NetSentry-windows-amd64.exe` from this repository's **Releases** page and
double-click it in a writable folder on Windows 10/11 x64. Go and Node.js are not
required; frontend assets are embedded. Microsoft WebView2 Runtime is required;
the app offers an online installation if it is missing. The executable is not
code-signed, so Windows may show an unknown-publisher or SmartScreen warning.
Configuration, logs and reports are generated at runtime. Release assets also
include `LICENSE` and `SHA256SUMS.txt`.

### Development

Install frontend dependencies:

```powershell
cd frontend
npm.cmd install
```

From the project root, build the embedded frontend before running tests:

```powershell
npm.cmd --prefix frontend ci
npm.cmd --prefix frontend run build
go test ./...
go vet ./...
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

UI counters cover the current monitoring session. Reports contain that session's
samples for each local calendar day and rotate at midnight. Restarting monitoring
does not restore earlier counters and replaces the same day's report; event logs
are always appended. Reports refresh on a timer, state changes, day changes, and
shutdown. File write errors are shown in the UI console.

### Release Automation

Push a new version tag such as `v1.0.0` to run the **Windows Release** workflow.
It builds the frontend, runs Go tests and static checks, builds the Windows x64
executable with the Wails version from `go.mod`, and publishes the executable,
license and checksums to GitHub Releases. Ordinary branch pushes do not publish.
Tags containing a hyphen (such as `v1.1.0-beta.1`) are marked as prereleases.
Commit and push the workflow before tagging; use a new tag for each release.
Existing releases are not overwritten. No personal access token is required.

## License

MIT. See [LICENSE](LICENSE).
