# ⚡ AppsManager v1.0

> 极轻量、高性能、零依赖的多项目与微服务统一管理调试桌面工作台。

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Vue 3](https://img.shields.io/badge/Vue-3.5+-4FC08D?style=flat&logo=vuedotjs)](https://vuejs.org)
[![Vite](https://img.shields.io/badge/Vite-8.0+-646CFF?style=flat&logo=vite)](https://vitejs.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## 🌟 核心特性

- 🚀 **极轻量零依赖**：基于 Go 原生静态编译，单文件仅 **7.6 MB**，内存占用仅 **15~25 MB**，无任何外部环境依赖。
- 🖥️ **硬件加速彩色终端**：集成 **xterm.js Canvas** 渲染与 **Ring Buffer 环形缓冲**，彻底告别海量日志刷屏卡顿。
- 🔍 **智能启动项穷举嗅探**：内置现代化多框架规则引擎（Vue/React/Vite/FastAPI/Django/Spring/Go/Rust/DotNet），自动识别子服务。
- 🎯 **真实端口接管与诊断**：支持毫秒级真实端口健康探测，自动接管后台运行服务并回显 PID、进程名与访问地址。
- 📦 **项目分组与拖拽归类**：支持卡片一键拖拽移入分组，自定义分组支持自由拖拽上下排序并即时持久化。
- 🔗 **终端 URL 原生点击跳转**：终端日志中的网络地址支持鼠标悬停下划线高亮，点击直达默认浏览器打开。
- 🛑 **一键安全关停与退出**：右上角一键安全停止所有托管子服务并彻底退出进程，零端口残留。

---

## 🚀 快速开始

### 方式 1：直接运行预编译单文件
从 [Releases](https://github.com/hjp1137/AppsManager/releases) 下载 `AppsManager.exe`，直接双击运行即可，无需安装任何环境！

### 方式 2：从源码编译运行

#### 准备环境
- [Go 1.22+](https://golang.org/dl/)
- [Node.js 18+](https://nodejs.org/) & [pnpm](https://pnpm.io/) (仅在修改前端源码时需要)

#### 1. 克隆代码
```bash
git clone https://github.com/hjp1137/AppsManager.git
cd AppsManager
```

#### 2. 直接运行
```bash
go run main.go
```

#### 3. 生产打包单文件 EXE
```bash
# 1. 编译前端（如已存在 dist 可跳过）
cd frontend && pnpm install && pnpm build && cd ..

# 2. 编译 Go 原生无黑框单文件
go build -ldflags "-s -w -H windowsgui" -o AppsManager.exe .
```

---

## 🏗️ 架构设计

AppsManager 采用前后端彻底解耦但单文件静态内嵌打包的现代化桌面架构：
- **后端引擎 (Go)**：负责子进程 Actor 生命周期管理、非阻塞端口探测、微批日志流（50ms Throttle）以及 WebSocket 双向通信。
- **前端界面 (Vue 3 + Vite + xterm.js)**：负责暗黑主题卡片流、侧边栏拖拽归类与排序、硬件加速终端渲染。
- **数据持久化**：用户配置保存于系统 `%LOCALAPPDATA%\AppsManager\` 与浏览器 LocalStorage 中，源码仓库零私人数据残留。

---

## 📄 开源协议

本项目采用 [MIT License](LICENSE) 协议。
