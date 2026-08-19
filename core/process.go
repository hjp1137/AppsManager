package core

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type LogCallback func(serviceId string, batchLogs []string)
type StatusCallback func(serviceId string, status ProjectStatus)
type PortCorrectionCallback func(serviceId string, detectedPort int)

type ProcessManager struct {
	mu             sync.RWMutex
	runningCmds    map[string]*exec.Cmd
	cancelFuncs    map[string]context.CancelFunc
	ringBuffers    map[string]*RingBuffer
	takeoverActive map[string]bool
	portManager    *PortManager
	detector       *Detector
	onLog          LogCallback
	onStatus       StatusCallback
	onPortUpdate   PortCorrectionCallback
}

func NewProcessManager(
	pm *PortManager,
	det *Detector,
	onLog LogCallback,
	onStatus StatusCallback,
	onPortUpdate PortCorrectionCallback,
) *ProcessManager {
	return &ProcessManager{
		runningCmds:    make(map[string]*exec.Cmd),
		cancelFuncs:    make(map[string]context.CancelFunc),
		ringBuffers:    make(map[string]*RingBuffer),
		takeoverActive: make(map[string]bool),
		portManager:    pm,
		detector:       det,
		onLog:          onLog,
		onStatus:       onStatus,
		onPortUpdate:   onPortUpdate,
	}

}


// RingBuffer 定长环形缓冲队列，防止内存暴涨
type RingBuffer struct {
	mu       sync.RWMutex
	capacity int
	lines    []string
}

func NewRingBuffer(cap int) *RingBuffer {
	return &RingBuffer{
		capacity: cap,
		lines:    make([]string, 0, cap),
	}
}

func (rb *RingBuffer) Push(line string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if len(rb.lines) >= rb.capacity {
		// 淘汰最老的 20% 数据
		dropCount := rb.capacity / 5
		if dropCount < 1 {
			dropCount = 1
		}
		rb.lines = rb.lines[dropCount:]
	}
	rb.lines = append(rb.lines, line)
}

func (rb *RingBuffer) GetAll() []string {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	res := make([]string, len(rb.lines))
	copy(res, rb.lines)
	return res
}

func (rb *RingBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.lines = rb.lines[:0]
}

func getProjectDisplayName(svc *SubService) string {
	if svc.ProjectName != "" {
		return svc.ProjectName
	}
	if svc.Name != "" {
		return svc.Name
	}
	return "应用服务"
}

func (p *ProcessManager) PushLog(serviceId string, lines []string) {
	p.mu.Lock()
	rb, exists := p.ringBuffers[serviceId]
	if !exists {
		rb = NewRingBuffer(3000)
		p.ringBuffers[serviceId] = rb
	}
	p.mu.Unlock()

	for _, l := range lines {
		rb.Push(l)
	}
	p.onLog(serviceId, lines)
}

func (p *ProcessManager) GetLogs(serviceId string) []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if rb, ok := p.ringBuffers[serviceId]; ok {
		return rb.GetAll()
	}
	return []string{}
}

func (p *ProcessManager) IsRunning(serviceId string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, running := p.runningCmds[serviceId]
	return running
}



// StartSubService 独立协程物理隔离启动单个服务
func (p *ProcessManager) StartSubService(svc *SubService) {
	p.mu.Lock()
	if _, running := p.runningCmds[svc.Id]; running {
		p.mu.Unlock()
		return
	}

	p.takeoverActive[svc.Id] = true

	rb, exists := p.ringBuffers[svc.Id]
	if !exists {
		rb = NewRingBuffer(3000)
		p.ringBuffers[svc.Id] = rb
	}
	rb.Clear()

	ctx, cancel := context.WithCancel(context.Background())
	p.cancelFuncs[svc.Id] = cancel

	p.mu.Unlock()

	p.onStatus(svc.Id, StatusStarting)

	// 独立 Goroutine 启动进程，故障完全物理隔离
	go func() {
		defer func() {
			p.mu.Lock()
			delete(p.runningCmds, svc.Id)
			delete(p.cancelFuncs, svc.Id)
			p.takeoverActive[svc.Id] = false
			svc.ProcessId = nil
			svc.Status = StatusStopped
			p.mu.Unlock()
			p.onStatus(svc.Id, StatusStopped)
		}()


		// 启动前若端口已被残留进程占用，自动先清理端口，防止 EADDRINUSE 报错
		if svc.Port > 0 && p.portManager.IsPortInUse(svc.Port) {
			p.portManager.KillProcessOnPort(svc.Port)
			time.Sleep(100 * time.Millisecond)
		}

		// 解析复合路径
		workDir, cmdLine := p.detector.ParseCompositeCommand(svc.StartCommand, svc.Path)


		cmd := exec.CommandContext(ctx, "cmd.exe", "/c", cmdLine)
		cmd.Dir = workDir
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow: true,
			CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		}

		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			rb.Push(fmt.Sprintf("创建标准输出管道失败: %v", err))
			return
		}
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			rb.Push(fmt.Sprintf("创建标准错误管道失败: %v", err))
			return
		}

		if err := cmd.Start(); err != nil {
			rb.Push(fmt.Sprintf("启动命令失败: %v", err))
			return
		}

		p.mu.Lock()
		p.runningCmds[svc.Id] = cmd
		p.mu.Unlock()

		pid := cmd.Process.Pid
		svc.ProcessId = &pid
		projName := getProjectDisplayName(svc)
		nowStr := time.Now().Format("2006-01-02 15:04:05")

		// 打印服务启动信息与访问地址
		localIP := p.portManager.GetLocalIP()
		rb.Push("\x1b[36m============================================================\x1b[0m")
		rb.Push(fmt.Sprintf("\x1b[1;32m➜  [%s] 正在启动服务: %s\x1b[0m", projName, svc.Name))
		rb.Push(fmt.Sprintf("\x1b[33m➜  启动时间:     \x1b[0m\x1b[33m%s\x1b[0m", nowStr))
		rb.Push(fmt.Sprintf("\x1b[33m➜  工作目录:     \x1b[0m\x1b[33m%s\x1b[0m", workDir))
		rb.Push(fmt.Sprintf("\x1b[33m➜  启动命令:     \x1b[0m\x1b[33m%s\x1b[0m", cmdLine))
		if svc.Port > 0 {
			rb.Push(fmt.Sprintf("\x1b[33m➜  本地 IP 访问: \x1b[0m\x1b[36mhttp://127.0.0.1:%d/\x1b[0m", svc.Port))
			rb.Push(fmt.Sprintf("\x1b[33m➜  本地域名访问: \x1b[0m\x1b[36mhttp://localhost:%d/\x1b[0m", svc.Port))
			if localIP != "" && localIP != "127.0.0.1" {
				rb.Push(fmt.Sprintf("\x1b[33m➜  局域网访问:   \x1b[0m\x1b[36mhttp://%s:%d/\x1b[0m", localIP, svc.Port))
			}
		}
		rb.Push("\x1b[36m============================================================\x1b[0m")

		// 启动异步日志微批处理合并推送
		p.streamLogs(ctx, svc, stdoutPipe, stderrPipe, rb)

		_ = cmd.Wait()
		exitTimeStr := time.Now().Format("2006-01-02 15:04:05")
		exitMsg := []string{
			"\x1b[33m------------------------------------------------------------\x1b[0m",
			fmt.Sprintf("\x1b[1;31m⏹  [%s] 进程已退出，服务已停止 [%s] (工作目录: %s)\x1b[0m", projName, exitTimeStr, workDir),
			"\x1b[33m------------------------------------------------------------\x1b[0m",
		}
		for _, m := range exitMsg {
			rb.Push(m)
		}
		p.onLog(svc.Id, exitMsg)
	}()
}


// ResetTakeover 当服务停止或端口释放时重置接管状态
func (p *ProcessManager) ResetTakeover(serviceId string) {
	p.mu.Lock()
	p.takeoverActive[serviceId] = false
	p.mu.Unlock()
}

// EnsureTakeoverLog 当外部已有进程在端口运行且尚未接管时，原子注入接管与健康诊断日志
func (p *ProcessManager) EnsureTakeoverLog(svc *SubService) {
	if svc.Port <= 0 {
		return
	}
	p.mu.Lock()
	if p.takeoverActive[svc.Id] {
		p.mu.Unlock()
		return
	}
	if _, isRunning := p.runningCmds[svc.Id]; isRunning {
		p.mu.Unlock()
		return
	}
	p.takeoverActive[svc.Id] = true

	rb, exists := p.ringBuffers[svc.Id]
	if !exists {
		rb = NewRingBuffer(3000)
		p.ringBuffers[svc.Id] = rb
	}
	p.mu.Unlock()


	pids := p.portManager.GetPidsByPort(svc.Port)
	pidStr := "未知"
	procName := "未知程序"
	if len(pids) > 0 {
		pidStr = fmt.Sprintf("%d", pids[0])
		procName = p.portManager.GetProcessNameByPid(pids[0])
	}
	health := p.portManager.CheckHttpHealth(svc.Port)
	localIP := p.portManager.GetLocalIP()
	projName := getProjectDisplayName(svc)
	nowStr := time.Now().Format("2006-01-02 15:04:05")

	header := []string{
		"\x1b[36m============================================================\x1b[0m",
		fmt.Sprintf("\x1b[1;32m➜  [%s 自动接管] 检测到服务正在后台正常运行中\x1b[0m", projName),
		fmt.Sprintf("\x1b[33m➜  检测时间:     \x1b[0m\x1b[33m%s\x1b[0m", nowStr),
		fmt.Sprintf("\x1b[33m➜  本地 IP 访问: \x1b[0m\x1b[36mhttp://127.0.0.1:%d/\x1b[0m", svc.Port),
		fmt.Sprintf("\x1b[33m➜  本地域名访问: \x1b[0m\x1b[36mhttp://localhost:%d/\x1b[0m", svc.Port),
	}
	if localIP != "" && localIP != "127.0.0.1" {
		header = append(header, fmt.Sprintf("\x1b[33m➜  局域网访问:   \x1b[0m\x1b[36mhttp://%s:%d/\x1b[0m", localIP, svc.Port))
	}

	header = append(header,
		fmt.Sprintf("\x1b[33m➜  监听端口状态: \x1b[0m\x1b[33m:%d (TCP 监听正常)\x1b[0m", svc.Port),
		fmt.Sprintf("\x1b[33m➜  系统进程信息: \x1b[0m\x1b[33mPID: %s (%s)\x1b[0m", pidStr, procName),
		fmt.Sprintf("\x1b[33m➜  服务健康探测: \x1b[0m\x1b[33m%s\x1b[0m", health),
		"\x1b[36m============================================================\x1b[0m",
	)

	for _, l := range header {
		rb.Push(l)
	}
	p.onLog(svc.Id, header)
}






// streamLogs 50ms微批聚合流，防止日志风暴卡死前端
func (p *ProcessManager) streamLogs(
	ctx context.Context,
	svc *SubService,
	stdout, stderr io.Reader,
	rb *RingBuffer,
) {
	logChan := make(chan string, 1000)
	var wg sync.WaitGroup

	readPipe := func(r io.Reader) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			rb.Push(line)
			select {
			case logChan <- line:
			default:
			}

			// 智能嗅探端口：仅当服务未显式配置端口 (svc.Port <= 0) 时才自动填充，用户手动配置端口优先级最高
			if svc.Port <= 0 {
				if detectedPort := p.detector.ExtractPortFromLog(line); detectedPort > 0 {
					svc.Port = detectedPort
					p.onPortUpdate(svc.Id, detectedPort)
				}
			}

		}
	}

	wg.Add(2)
	go readPipe(stdout)
	go readPipe(stderr)

	go func() {
		wg.Wait()
		close(logChan)
	}()

	// 50ms 聚合分发器
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		batch := make([]string, 0, 50)
		for {
			select {
			case <-ctx.Done():
				return
			case line, ok := <-logChan:
				if !ok {
					if len(batch) > 0 {
						p.onLog(svc.Id, batch)
					}
					return
				}
				batch = append(batch, line)
				if len(batch) >= 100 {
					p.onLog(svc.Id, batch)
					batch = make([]string, 0, 50)
				}
			case <-ticker.C:
				if len(batch) > 0 {
					p.onLog(svc.Id, batch)
					batch = make([]string, 0, 50)
				}
			}
		}
	}()
}

// StopSubService 强杀进程树并释放端口
func (p *ProcessManager) StopSubService(svc *SubService) {
	p.mu.Lock()
	cancel, hasCancel := p.cancelFuncs[svc.Id]
	cmd, hasCmd := p.runningCmds[svc.Id]
	p.takeoverActive[svc.Id] = false
	rb, hasRb := p.ringBuffers[svc.Id]

	if !hasRb {
		rb = NewRingBuffer(3000)
		p.ringBuffers[svc.Id] = rb
	}
	p.mu.Unlock()


	if hasCancel {
		cancel()
	}

	if hasCmd && cmd.Process != nil {
		// 使用 taskkill /F /T 递归杀掉子进程树
		killCmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", cmd.Process.Pid))
		killCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = killCmd.Run()
	}

	if svc.Port > 0 {
		p.portManager.KillProcessOnPort(svc.Port)
	}

	projName := getProjectDisplayName(svc)
	nowStr := time.Now().Format("2006-01-02 15:04:05")
	stopMsg := []string{
		"\x1b[33m------------------------------------------------------------\x1b[0m",
		fmt.Sprintf("\x1b[1;31m⏹  [%s] 服务已正式停止 [%s] (进程已退出，端口 :%d 已完全释放)\x1b[0m", projName, nowStr, svc.Port),
		"\x1b[33m------------------------------------------------------------\x1b[0m",
	}
	for _, m := range stopMsg {
		rb.Push(m)
	}

	p.onLog(svc.Id, stopMsg)

	svc.Status = StatusStopped
	svc.ProcessId = nil
	p.onStatus(svc.Id, StatusStopped)
}


func (p *ProcessManager) StopAll(services []*SubService) {
	for _, s := range services {
		p.StopSubService(s)
	}
}



