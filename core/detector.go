package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Detector struct {
	RulesMgr *RulesConfigManager
}

func NewDetector(rulesMgr *RulesConfigManager) *Detector {
	return &Detector{
		RulesMgr: rulesMgr,
	}
}

// ParseCompositeCommand 解析复合命令如 "cd web && pnpm dev"
func (d *Detector) ParseCompositeCommand(rawCmd, basePath string) (string, string) {
	trimmed := strings.TrimSpace(rawCmd)
	if strings.HasPrefix(trimmed, "cd ") {
		parts := strings.SplitN(trimmed, "&&", 2)
		if len(parts) == 2 {
			cdPart := strings.TrimSpace(parts[0])
			cmdPart := strings.TrimSpace(parts[1])
			subDir := strings.TrimPrefix(cdPart, "cd ")
			subDir = strings.TrimSpace(strings.Trim(subDir, "\"'"))
			return filepath.Join(basePath, subDir), cmdPart
		}
	}
	return basePath, trimmed
}

// ExtractPortFromLog 从控制台日志中提取端口
func (d *Detector) ExtractPortFromLog(logLine string) int {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	clean := ansiRegex.ReplaceAllString(logLine, "")
	portRegexes := []*regexp.Regexp{
		regexp.MustCompile(`(?i)localhost:(\d{4,5})`),
		regexp.MustCompile(`(?i)127\.0\.0\.1:(\d{4,5})`),
		regexp.MustCompile(`(?i)0\.0\.0\.0:(\d{4,5})`),
		regexp.MustCompile(`(?i)listening on (?:port )?(\d{4,5})`),
		regexp.MustCompile(`(?i)port:?\s*(\d{4,5})`),
	}
	for _, re := range portRegexes {
		if m := re.FindStringSubmatch(clean); len(m) > 1 {
			var p int
			_ = json.Unmarshal([]byte(m[1]), &p)
			if p > 0 && p <= 65535 {
				return p
			}
		}
	}
	return 0
}

// DetectSubServices 基于动态启用的规则列表穷举识别项目
func (d *Detector) DetectSubServices(rootPath string) []*SubService {
	results := make([]*SubService, 0)
	if _, err := os.Stat(rootPath); err != nil {
		return results
	}

	ignoreDirs := map[string]bool{
		"node_modules": true, ".git": true, "bin": true, "obj": true,
		"dist": true, ".idea": true, ".vscode": true, "vendor": true,
		"target": true, "__pycache__": true, ".venv": true, "env": true,
	}

	// 收集根目录及深度 <= 3 的所有子目录
	dirsToScan := []string{rootPath}
	_ = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		if ignoreDirs[info.Name()] {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(rootPath, path)
		if strings.Count(rel, string(filepath.Separator)) > 3 {
			return filepath.SkipDir
		}
		if path != rootPath {
			dirsToScan = append(dirsToScan, path)
		}
		return nil
	})

	activeRules := d.RulesMgr.GetRules()

	// 对每个目录按动态启用的规则逐一匹配
	for _, dir := range dirsToScan {
		for _, rule := range activeRules {
			if !rule.Enabled {
				continue
			}
			matched := matchRuleOnDir(rule, dir, rootPath)
			for _, m := range matched {
				if !isDuplicateService(results, m) {
					results = append(results, m)
				}
			}
		}
	}

	return results
}
