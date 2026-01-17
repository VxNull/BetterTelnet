
# BetterTelnet (btel)

[English](#english) | [中文](#chinese)

**BetterTelnet** (`btel`) is a lightweight, modern replacement for the native Windows Telnet client, designed specifically to work perfectly with **Windows Terminal**.

It solves the annoying issue where the native `telnet.exe` takes over the console buffer, preventing mouse scrolling and history review. With `btel`, output acts as a standard stream, allowing Windows Terminal to handle the scrollback buffer naturally.

---

<a name="chinese"></a>

# BetterTelnet (btel) - Windows Terminal 的最佳 Telnet 伴侣

**BetterTelnet** (`btel`) 是一个轻量级的 Windows Telnet 客户端替代工具，专为完美适配 **Windows Terminal** 而设计。

它解决了 Windows 自带 `telnet.exe` 的核心痛点：原生工具会强制接管控制台缓冲区，导致**无法使用鼠标滚轮查看历史记录**（Scrollback）。`btel` 通过标准流式输出，将滚动与缓冲区的管理权交还给 Windows Terminal，并增加了日志记录功能。

## ✨ 核心特性

*   **🖱️ 完美支持滚动**：不再有“屏幕独占”模式。数据作为标准流输出，你可以使用鼠标滚轮随意查看历史记录。
*   **📜 自动日志记录**：支持将会话内容实时保存到本地文件，方便审计和排错。
*   **🎨 ANSI 颜色透传**：完美保留远程设备（如 Linux Shell, Cisco 路由器）的语法高亮和彩色输出。
*   **🚀 轻量单文件**：基于 Go 语言编写，编译后为单个 `.exe` 文件，无任何运行时依赖。
*   **⌨️ 原始模式体验**：模拟终端原始模式（Raw Mode），支持 Ctrl+C、Tab 补全等快捷键的透传。
*   **🛠️ 兼容原生语法**：参数传递方式与标准 Telnet 保持一致，无需学习新命令。

## 🚀 快速开始

### 下载安装
您可以直接从 [Releases](../../releases) 页面下载最新的 `btel.exe` 并将其放入您的 PATH 路径中（例如 `C:\Windows\System32` 或自定义工具目录）。

### 使用方法

语法与标准 Telnet 完全一致：

```bash
btel [options] <host> [port]
```

#### 示例

1.  **连接到默认端口 (23)**
    ```powershell
    btel 192.168.1.1
    ```

2.  **连接到指定端口**
    ```powershell
    btel 192.168.1.1 8080
    ```

3.  **连接并保存日志到文件**
    ```powershell
    btel -log session.log 192.168.1.1
    ```

4.  **查看帮助**
    ```powershell
    btel -h
    ```

## 🛠️ 编译指南

如果您想自己修改代码或从源码编译，请确保已安装 Go 1.16+ 环境。

1.  **克隆项目**
    ```bash
    git clone https://github.com/VxNull/BetterTelnet.gits
    cd BetterTelnet
    ```

2.  **下载依赖**
    ```bash
    go mod tidy
    ```

3.  **编译**
    ```bash
    go build -o btel.exe main.go
    ```

## 🧩 技术原理

Windows 自带的 Telnet 客户端使用了古老的 Console API 来控制屏幕绘制，这与现代的终端模拟器（如 Windows Terminal, VS Code Terminal）兼容性不佳。

**BetterTelnet 的工作原理：**
1.  **网络层**：建立 TCP 连接，并内置一个轻量级的 Telnet 协议状态机，过滤掉协议握手指令（IAC Commands），只保留纯文本数据。
2.  **终端层**：将本地终端设置为 `Raw Mode`（原始模式），实现按键的字节级透传。
3.  **输出层**：将清洗后的数据直接写入 `Stdout`。这使得 Windows Terminal 可以像处理普通文本流一样处理 Telnet 输出，从而利用其原生的高性能缓冲区和滚动条功能。

## 📄 开源协议

本项目采用 [MIT License](LICENSE) 开源。

---

<a name="english"></a>

# English Description

## ✨ Features

*   **🖱️ Scrollback Support**: Output is streamed to stdout, allowing Windows Terminal to manage the buffer. Mouse wheel works perfectly.
*   **📜 Session Logging**: Easily save your session output to a file with the `-log` flag.
*   **🎨 ANSI Passthrough**: Colors and formatting from remote hosts are preserved.
*   **🚀 Lightweight**: A single static binary with no dependencies.
*   **🛠️ Familiar Syntax**: Usage arguments match the standard `telnet` command.

## 🚀 Usage

```bash
# Standard connection (default port 23)
btel 192.168.1.1

# Specify port
btel 192.168.1.1 8080

# Log output to a file
btel -log output.txt 192.168.1.1
```

## 🛠️ Building from Source

Requirements: Go 1.16+

```bash
git clone https://github.com/VxNull/BetterTelnet.git
cd BetterTelnet
go mod tidy
go build -o btel.exe main.go
```

## License

MIT License.