package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

func isDuplicateService(list []*SubService, item *SubService) bool {
	for _, exist := range list {
		if exist.Path == item.Path && exist.StartCommand == item.StartCommand {
			return true
		}
	}
	return false
}

// matchRuleOnDir 对目录执行特定规则匹配
func matchRuleOnDir(rule *DetectionRule, dirPath, rootPath string) []*SubService {
	results := make([]*SubService, 0)
	displayDir := filepath.Base(dirPath)
	if dirPath == rootPath {
		displayDir = ""
	}

	// 1. .NET 通配匹配 (*.csproj)
	if strings.HasSuffix(rule.MatchFile, ".csproj") {
		entries, err := os.ReadDir(dirPath)
		if err != nil { return results }
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".csproj") {
				contentBytes, _ := os.ReadFile(filepath.Join(dirPath, entry.Name()))
				content := string(contentBytes)
				if rule.MatchContent == "" || strings.Contains(content, rule.MatchContent) {
					projName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
					port := rule.DefaultPort
					// launchSettings.json 端口提取
					launchPath := filepath.Join(dirPath, "Properties", "launchSettings.json")
					if lb, err := os.ReadFile(launchPath); err == nil {
						re := regexp.MustCompile(`https?://localhost:(\d+)`)
						if m := re.FindStringSubmatch(string(lb)); len(m) > 1 {
							var p int
							_ = json.Unmarshal([]byte(m[1]), &p)
							if p > 0 { port = p }
						}
					}
					name := fmt.Sprintf("%s (%s)", rule.Name, projName)
					results = append(results, createSubService(name, dirPath, rule.Command, port))
				}
			}
		}
		return results
	}

	// 2. Node 前端专属匹配 (package.json)
	if rule.MatchFile == "package.json" {
		pkgPath := filepath.Join(dirPath, "package.json")
		data, err := os.ReadFile(pkgPath)
		if err != nil { return results }
		var pkg struct {
			Scripts map[string]string `json:"scripts"`
		}
		if err := json.Unmarshal(data, &pkg); err != nil { return results }

		var scriptKey string
		if re := regexp.MustCompile(`[a-zA-Z0-9_:-]+`); re != nil {
			scriptKey = re.FindString(rule.MatchContent)
		}
		if scriptKey == "" { scriptKey = "dev" }

		if _, ok := pkg.Scripts[scriptKey]; ok {
			pm := "npm"
			if _, err := os.Stat(filepath.Join(dirPath, "pnpm-lock.yaml")); err == nil {
				pm = "pnpm"
			} else if _, err := os.Stat(filepath.Join(rootPath, "pnpm-lock.yaml")); err == nil {
				pm = "pnpm"
			} else if _, err := os.Stat(filepath.Join(dirPath, "yarn.lock")); err == nil {
				pm = "yarn"
			} else if _, err := os.Stat(filepath.Join(dirPath, "bun.lockb")); err == nil {
				pm = "bun"
			}
			cmd := rule.Command
			if strings.HasPrefix(cmd, "npm run ") && pm != "npm" {
				cmd = fmt.Sprintf("%s run %s", pm, scriptKey)
			} else if strings.HasPrefix(cmd, "pnpm ") && pm != "pnpm" {
				cmd = fmt.Sprintf("%s %s", pm, scriptKey)
				if pm == "npm" { cmd = fmt.Sprintf("npm run %s", scriptKey) }
			}
			port := rule.DefaultPort
			customPort := sniffNodeCustomPort(dirPath)
			if customPort > 0 { port = customPort }

			name := rule.Name
			if displayDir != "" { name = fmt.Sprintf("%s (%s)", rule.Name, displayDir) }
			results = append(results, createSubService(name, dirPath, cmd, port))
		}
		return results
	}

	// 3. 通用文件与内容匹配
	targetFile := filepath.Join(dirPath, rule.MatchFile)
	if fi, err := os.Stat(targetFile); err == nil && !fi.IsDir() {
		matched := true
		if rule.MatchContent != "" {
			b, err := os.ReadFile(targetFile)
			if err != nil || !strings.Contains(string(b), rule.MatchContent) {
				matched = false
			}
		}
		if matched {
			name := rule.Name
			if displayDir != "" { name = fmt.Sprintf("%s (%s)", rule.Name, displayDir) }
			results = append(results, createSubService(name, dirPath, rule.Command, rule.DefaultPort))
		}
	}

	return results
}

func sniffNodeCustomPort(dir string) int {
	for _, envName := range []string{".env.development", ".env.local", ".env"} {
		envPath := filepath.Join(dir, envName)
		if b, err := os.ReadFile(envPath); err == nil {
			re := regexp.MustCompile(`(?i)(?:PORT|VITE_PORT)\s*=\s*(\d+)`)
			if m := re.FindStringSubmatch(string(b)); len(m) > 1 {
				var p int
				_ = json.Unmarshal([]byte(m[1]), &p)
				if p > 0 { return p }
			}
		}
	}
	for _, vName := range []string{"vite.config.js", "vite.config.ts", "vue.config.js"} {
		vPath := filepath.Join(dir, vName)
		if b, err := os.ReadFile(vPath); err == nil {
			re := regexp.MustCompile(`(?i)port:\s*(\d+)`)
			if m := re.FindStringSubmatch(string(b)); len(m) > 1 {
				var p int
				_ = json.Unmarshal([]byte(m[1]), &p)
				if p > 0 { return p }
			}
		}
	}
	return 0
}

func createSubService(name, path, cmd string, port int) *SubService {
	return &SubService{
		Id:           uuid.New().String(),
		Name:         name,
		Path:         path,
		StartCommand: cmd,
		Port:         port,
		Status:       StatusStopped,
	}
}
