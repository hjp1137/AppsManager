package core

import (
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRingBuffer(t *testing.T) {
	rb := NewRingBuffer(10)
	for i := 0; i < 15; i++ {
		rb.Push(fmt.Sprintf("log line %d", i))
	}
	logs := rb.GetAll()
	if len(logs) > 10 {
		t.Fatalf("RingBuffer size exceeded: %d", len(logs))
	}
}

func TestDetectorParseCommand(t *testing.T) {
	rulesMgr := NewRulesConfigManager()
	d := NewDetector(rulesMgr)
	dir, cmd := d.ParseCompositeCommand("cd web && pnpm dev", "D:\\myproject")
	if cmd != "pnpm dev" {
		t.Errorf("expected pnpm dev, got %s", cmd)
	}
	if dir != "D:\\myproject\\web" {
		t.Errorf("expected D:\\myproject\\web, got %s", dir)
	}
}

func TestDetectorExtractPort(t *testing.T) {
	rulesMgr := NewRulesConfigManager()
	d := NewDetector(rulesMgr)
	port := d.ExtractPortFromLog("\x1b[36m  ➜  Local:   http://localhost:5173/\x1b[39m")
	if port != 5173 {
		t.Errorf("expected 5173, got %d", port)
	}


	port2 := d.ExtractPortFromLog("Application listening on port 8080")
	if port2 != 8080 {
		t.Errorf("expected 8080, got %d", port2)
	}
}

func TestPortManager(t *testing.T) {
	pm := NewPortManager()
	
	// 随机起一个临时监听测试
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	time.Sleep(10 * time.Millisecond)

	if !pm.IsPortInUse(port) {
		t.Errorf("port %d should be in use", port)
	}

	if pm.IsPortInUse(59999) {
		t.Errorf("port 59999 should not be in use")
	}
}

func TestDetectorExhaustiveRules(t *testing.T) {
	rulesMgr := NewRulesConfigManager()
	d := NewDetector(rulesMgr)
	tempDir := t.TempDir()

	// 1. 创建子目录前端项目
	webDir := tempDir + "/frontend_app"
	_ = os.MkdirAll(webDir, 0755)
	pkgJSON := `{"name":"test-web","scripts":{"serve":"vue-cli-service serve"}}`
	_ = os.WriteFile(webDir+"/package.json", []byte(pkgJSON), 0644)

	// 2. 创建子目录 Python 项目
	pyDir := tempDir + "/python_api"
	_ = os.MkdirAll(pyDir, 0755)
	_ = os.WriteFile(pyDir+"/main.py", []byte("from fastapi import FastAPI\napp = FastAPI()"), 0644)

	results := d.DetectSubServices(tempDir)
	if len(results) < 2 {
		t.Fatalf("expected at least 2 services detected via exhaustive rules, got %d", len(results))
	}

	foundServe := false
	foundFastAPI := false
	for _, res := range results {
		if res.Port == 8080 && strings.Contains(res.StartCommand, "serve") {
			foundServe = true
		}
		if res.Port == 8000 && strings.Contains(res.Name, "FastAPI") {
			foundFastAPI = true
		}
	}

	if !foundServe {
		t.Errorf("failed to detect npm run serve on port 8080")
	}
	if !foundFastAPI {
		t.Errorf("failed to detect FastAPI on port 8000")
	}
}

func TestRulesConfigManager(t *testing.T) {
	mgr := NewRulesConfigManager()
	rules := mgr.GetRules()
	if len(rules) == 0 {
		t.Fatal("expected default rules, got empty")
	}

	// 测试新增与保存
	newRule := &DetectionRule{
		Id:          "test-custom-rule",
		Name:        "Custom Ruby",
		Category:    "Ruby",
		MatchFile:   "config.ru",
		Command:     "rackup",
		DefaultPort: 9292,
		Enabled:     true,
	}
	rules = append(rules, newRule)
	if err := mgr.Save(rules); err != nil {
		t.Fatalf("failed to save rules: %v", err)
	}

	loaded := mgr.GetRules()
	found := false
	for _, r := range loaded {
		if r.Id == "test-custom-rule" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected test-custom-rule to be saved and retrieved")
	}
}


