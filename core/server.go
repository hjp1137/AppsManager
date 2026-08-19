package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Server struct {
	ConfigMgr  *ConfigManager
	RulesMgr   *RulesConfigManager
	PortMgr    *PortManager
	ProcMgr    *ProcessManager
	Detector   *Detector
	wsClients  map[*websocket.Conn]bool
	wsMu       sync.Mutex
}

func NewServer() *Server {
	rulesMgr := NewRulesConfigManager()
	detector := NewDetector(rulesMgr)
	s := &Server{
		ConfigMgr: NewConfigManager(),
		RulesMgr:  rulesMgr,
		PortMgr:   NewPortManager(),
		Detector:  detector,
		wsClients: make(map[*websocket.Conn]bool),
	}


	s.ProcMgr = NewProcessManager(
		s.PortMgr,
		s.Detector,
		s.broadcastLog,
		s.broadcastStatus,
		s.broadcastPortCorrection,
	)


	// 启动后台毫秒级非阻塞端口轮询协程 (2秒轮询一次)
	go s.startPortPolling()

	return s
}

func (s *Server) broadcastLog(serviceId string, batch []string) {
	s.broadcast(WSMessage{
		Type:      "log_batch",
		ServiceId: serviceId,
		Data:      batch,
	})
}

func (s *Server) broadcastStatus(serviceId string, status ProjectStatus) {
	s.broadcast(WSMessage{
		Type:      "status",
		ServiceId: serviceId,
		Status:    status,
	})
}

func (s *Server) broadcastPortCorrection(serviceId string, detectedPort int) {
	s.ConfigMgr.mu.Lock()
	for _, p := range s.ConfigMgr.Projects {
		for _, svc := range p.SubServices {
			if svc.Id == serviceId && svc.Port <= 0 {
				svc.Port = detectedPort
			}
		}
	}
	s.ConfigMgr.mu.Unlock()
	_ = s.ConfigMgr.Save(s.ConfigMgr.Projects)

	s.broadcast(WSMessage{
		Type:      "port_corrected",
		ServiceId: serviceId,
		Data:      detectedPort,
	})
}


func (s *Server) broadcast(msg WSMessage) {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	for client := range s.wsClients {
		_ = client.WriteMessage(websocket.TextMessage, data)
	}
}

// startPortPolling 毫秒级非阻塞并发探测各子服务端口状态并无缝接管真实运行态
func (s *Server) startPortPolling() {
	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		projects := s.ConfigMgr.GetProjects()
		for _, p := range projects {
			for _, svc := range p.SubServices {
				if svc.ProjectName == "" {
					svc.ProjectName = p.Name
				}
				if svc.Port > 0 {
					inUse := s.PortMgr.IsPortInUse(svc.Port)
					if inUse {
						if svc.Status != StatusRunning {
							svc.Status = StatusRunning
							s.broadcastStatus(svc.Id, StatusRunning)
							s.ProcMgr.EnsureTakeoverLog(svc)
						}
					} else if !inUse && !s.ProcMgr.IsRunning(svc.Id) && svc.Status == StatusRunning {
						svc.Status = StatusStopped
						s.ProcMgr.ResetTakeover(svc.Id)
						s.broadcastStatus(svc.Id, StatusStopped)
						projName := svc.ProjectName
						if projName == "" {
							projName = svc.Name
						}
						s.ProcMgr.PushLog(svc.Id, []string{
							"\x1b[33m------------------------------------------------------------\x1b[0m",
							fmt.Sprintf("\x1b[1;31m⏹  [%s] 监听端口 :%d 已释放，服务已停止\x1b[0m", projName, svc.Port),
							"\x1b[33m------------------------------------------------------------\x1b[0m",
						})
					}
				}
			}
		}
	}
}




func (s *Server) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/api/projects", s.handleProjects)
	mux.HandleFunc("/api/projects/detect", s.handleDetect)
	mux.HandleFunc("/api/rules", s.handleRules)
	mux.HandleFunc("/api/rules/reset", s.handleResetRules)
	mux.HandleFunc("/api/service/start", s.handleStartService)
	mux.HandleFunc("/api/service/stop", s.handleStopService)
	mux.HandleFunc("/api/service/restart", s.handleRestartService)
	mux.HandleFunc("/api/service/kill_port", s.handleKillPort)
	mux.HandleFunc("/api/service/stop_all", s.handleStopAll)
	mux.HandleFunc("/api/service/logs", s.handleGetLogs)
	mux.HandleFunc("/api/system/open_folder", s.handleOpenFolder)
	mux.HandleFunc("/api/system/open_code", s.handleOpenCode)
	mux.HandleFunc("/api/system/open_hbuilderx", s.handleOpenHBuilderX)
	mux.HandleFunc("/api/system/exit", s.handleSystemExit)
	mux.HandleFunc("/api/system/select_folder", s.handleSelectFolder)
	mux.HandleFunc("/api/system/info", s.handleSystemInfo)
}


func (s *Server) handleRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		_ = json.NewEncoder(w).Encode(s.RulesMgr.GetRules())
		return
	}
	if r.Method == http.MethodPost {
		var rules []*DetectionRule
		if err := json.NewDecoder(r.Body).Decode(&rules); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = s.RulesMgr.Save(rules)
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}
}

func (s *Server) handleResetRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rules := s.RulesMgr.ResetToDefault()
	_ = json.NewEncoder(w).Encode(rules)
}


func (s *Server) handleSystemExit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	go func() {
		time.Sleep(200 * time.Millisecond)
		var allServices []*SubService
		for _, p := range s.ConfigMgr.GetProjects() {
			allServices = append(allServices, p.SubServices...)
		}
		s.ProcMgr.StopAll(allServices)
		os.Exit(0)
	}()
}

func (s *Server) handleSelectFolder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	script := `Add-Type -AssemblyName System.Windows.Forms; $f = New-Object System.Windows.Forms.FolderBrowserDialog; $f.Description = '请选择项目所在的本地物理文件夹'; $f.ShowNewFolderButton = $true; if ($f.ShowDialog() -eq 'OK') { Write-Output $f.SelectedPath }`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	var selectedPath string
	if err == nil {
		selectedPath = strings.TrimSpace(string(out))
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"path": selectedPath})
}



func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	s.wsMu.Lock()
	s.wsClients[conn] = true
	s.wsMu.Unlock()

	defer func() {
		s.wsMu.Lock()
		delete(s.wsClients, conn)
		s.wsMu.Unlock()
		_ = conn.Close()
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		projects := s.ConfigMgr.GetProjects()
		for _, p := range projects {
			for _, svc := range p.SubServices {
				if svc.ProjectName == "" {
					svc.ProjectName = p.Name
				}
				if svc.Port > 0 && s.PortMgr.IsPortInUse(svc.Port) {
					svc.Status = StatusRunning
					s.ProcMgr.EnsureTakeoverLog(svc)
				} else if !s.ProcMgr.IsRunning(svc.Id) {
					svc.Status = StatusStopped
				}
			}
		}
		_ = json.NewEncoder(w).Encode(projects)
		return
	}



	if r.Method == http.MethodPost {
		var projects []*ProjectItem
		if err := json.NewDecoder(r.Body).Decode(&projects); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// 确保所有子服务都继承项目物理路径
		for _, p := range projects {
			for _, svc := range p.SubServices {
				if svc.Path == "" {
					svc.Path = p.Path
				}
			}
		}
		_ = s.ConfigMgr.Save(projects)
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
		s.broadcast(WSMessage{Type: "sync", Data: projects})
	}
}

func (s *Server) handleDetect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	detected := s.Detector.DetectSubServices(req.Path)
	_ = json.NewEncoder(w).Encode(detected)
}

func (s *Server) findSubService(serviceId string) *SubService {
	for _, p := range s.ConfigMgr.GetProjects() {
		for _, svc := range p.SubServices {
			if svc.Id == serviceId {
				if svc.Path == "" {
					svc.Path = p.Path
				}
				if svc.ProjectName == "" {
					svc.ProjectName = p.Name
				}
				return svc
			}
		}
	}
	return nil
}


func (s *Server) handleStartService(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ServiceId string `json:"serviceId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	svc := s.findSubService(req.ServiceId)
	if svc == nil {
		http.Error(w, "service not found", http.StatusNotFound)
		return
	}
	s.ProcMgr.StartSubService(svc)
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *Server) handleStopService(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ServiceId string `json:"serviceId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	svc := s.findSubService(req.ServiceId)
	if svc == nil {
		http.Error(w, "service not found", http.StatusNotFound)
		return
	}
	s.ProcMgr.StopSubService(svc)
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *Server) handleRestartService(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ServiceId string `json:"serviceId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	svc := s.findSubService(req.ServiceId)
	if svc == nil {
		http.Error(w, "service not found", http.StatusNotFound)
		return
	}
	go func() {
		s.ProcMgr.StopSubService(svc)
		time.Sleep(300 * time.Millisecond)
		s.ProcMgr.StartSubService(svc)
	}()
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *Server) handleKillPort(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Port      int    `json:"port"`
		ServiceId string `json:"serviceId,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	var killedCount int
	if req.Port > 0 {
		killedPids := s.PortMgr.KillProcessOnPort(req.Port)
		killedCount = len(killedPids)

		if req.ServiceId != "" {
			svc := s.findSubService(req.ServiceId)
			projName := "应用服务"
			if svc != nil && svc.ProjectName != "" {
				projName = svc.ProjectName
			}
			if killedCount > 0 {
				s.broadcastLog(req.ServiceId, []string{
					fmt.Sprintf("\x1b[33m[%s] ⚡ 成功终止占用端口 %d 的进程 (PID: %v)，端口已释放！\x1b[0m", projName, req.Port, killedPids),
				})
			} else {
				s.broadcastLog(req.ServiceId, []string{
					fmt.Sprintf("\x1b[32m[%s] ⚡ 端口 %d 检查完成，当前未被占用。\x1b[0m", projName, req.Port),
				})
			}
			s.broadcastStatus(req.ServiceId, StatusStopped)
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"killed":  killedCount,
	})
}


func (s *Server) handleStopAll(w http.ResponseWriter, r *http.Request) {
	var allServices []*SubService
	for _, p := range s.ConfigMgr.GetProjects() {
		allServices = append(allServices, p.SubServices...)
	}
	s.ProcMgr.StopAll(allServices)
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *Server) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	serviceId := r.URL.Query().Get("id")
	logs := s.ProcMgr.GetLogs(serviceId)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(logs)
}


func (s *Server) handleOpenFolder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Path != "" {
		cmd := exec.Command("explorer.exe", req.Path)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = cmd.Start()
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *Server) handleOpenCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Path != "" {
		cmd := exec.Command("cmd.exe", "/c", "code", req.Path)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = cmd.Start()
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *Server) handleOpenHBuilderX(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Path != "" {
		hxPaths := []string{
			`D:\Program Files\HBuilderX\cli.exe`,
			`C:\Program Files\HBuilderX\cli.exe`,
			`C:\Program Files (x86)\HBuilderX\cli.exe`,
			`D:\HBuilderX\cli.exe`,
			`C:\HBuilderX\cli.exe`,
		}
		var hxCli string
		for _, p := range hxPaths {
			if _, err := os.Stat(p); err == nil {
				hxCli = p
				break
			}
		}
		if hxCli != "" {
			cmd := exec.Command(hxCli, "open", "--path", req.Path)
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			_ = cmd.Start()
		} else {
			cmd := exec.Command("cmd.exe", "/c", "start", "HBuilderX", req.Path)
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			_ = cmd.Start()
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *Server) handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	localIP := s.PortMgr.GetLocalIP()
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"localIp": localIP,
	})
}






