package core

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)


type PortManager struct{}

func NewPortManager() *PortManager {
	return &PortManager{}
}

// IsPortInUse 毫秒级非阻塞检测端口是否处于监听状态
func (pm *PortManager) IsPortInUse(port int) bool {
	if port <= 0 || port > 65535 {
		return false
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 40*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		return true
	}
	return false
}

// GetPidsByPort 快速精准获取占用指定端口的所有 PID
func (pm *PortManager) GetPidsByPort(port int) []int {
	pids := make([]int, 0)
	if port <= 0 || port > 65535 {
		return pids
	}

	// 1. 使用 netstat -ano 并在 Go 内部做正则精准匹配
	cmd := exec.Command("netstat", "-ano", "-p", "tcp")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		// 匹配形如 :5001 后面跟着空白的本地地址
		portPattern := regexp.MustCompile(fmt.Sprintf(`[:\.]%d\s+`, port))
		for _, line := range lines {
			if portPattern.MatchString(line) {
				fields := strings.Fields(line)
				if len(fields) >= 5 {
					pidStr := fields[len(fields)-1]
					if pid, err := strconv.Atoi(pidStr); err == nil && pid > 0 {
						addPidIfUnique(&pids, pid)
					}
				}
			}
		}
	}

	// 2. PowerShell Get-NetTCPConnection 补充探测
	psScript := fmt.Sprintf(`Get-NetTCPConnection -LocalPort %d -ErrorAction SilentlyContinue | Select-Object -ExpandProperty OwningProcess`, port)
	psCmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	psCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if psOut, err := psCmd.Output(); err == nil {
		lines := strings.Split(string(psOut), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if pid, err := strconv.Atoi(trimmed); err == nil && pid > 0 {
				addPidIfUnique(&pids, pid)
			}
		}
	}

	return pids
}

func addPidIfUnique(pids *[]int, pid int) {
	for _, p := range *pids {
		if p == pid {
			return
		}
	}
	*pids = append(*pids, pid)
}

// KillProcessOnPort 强杀占用端口的所有进程树
func (pm *PortManager) KillProcessOnPort(port int) []int {
	pids := pm.GetPidsByPort(port)
	currentPid := os.Getpid()
	killed := make([]int, 0)

	for _, pid := range pids {
		if pid == 0 || pid == 4 || pid == currentPid {
			continue
		}
		// 1. taskkill /F /T /PID 递归强杀整棵进程树
		kCmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
		kCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = kCmd.Run()

		// 2. PowerShell 强行终止保护
		psKill := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", fmt.Sprintf("Stop-Process -Id %d -Force -ErrorAction SilentlyContinue", pid))
		psKill.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = psKill.Run()

		killed = append(killed, pid)
	}

	return killed
}

// GetProcessNameByPid 获取指定 PID 的进程名称
func (pm *PortManager) GetProcessNameByPid(pid int) string {
	if pid <= 0 {
		return "unknown"
	}
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if out, err := cmd.Output(); err == nil {
		str := strings.TrimSpace(string(out))
		parts := strings.Split(str, ",")
		if len(parts) > 0 {
			return strings.Trim(parts[0], "\" \t\r\n")
		}
	}
	return "process"
}

// CheckHttpHealth 向端口发起健康探测
func (pm *PortManager) CheckHttpHealth(port int) string {
	if port <= 0 {
		return "无有效端口"
	}
	client := &http.Client{Timeout: 300 * time.Millisecond}
	start := time.Now()
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d", port))
	duration := time.Since(start).Milliseconds()
	if err == nil {
		defer resp.Body.Close()
		return fmt.Sprintf("HTTP %s (响应耗时: %dms)", resp.Status, duration)
	}
	if pm.IsPortInUse(port) {
		return fmt.Sprintf("TCP 连接正常 (响应耗时: %dms)", duration)
	}
	return "未连通"
}

// GetLocalIP 获取本机局域网 IP
func (pm *PortManager) GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

