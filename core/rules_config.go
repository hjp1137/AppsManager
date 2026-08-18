package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type RulesConfigManager struct {
	filePath string
	mu       sync.RWMutex
	rules    []*DetectionRule
}

func NewRulesConfigManager() *RulesConfigManager {
	appData := os.Getenv("LOCALAPPDATA")
	if appData == "" {
		appData = "."
	}
	dir := filepath.Join(appData, "AppsManager")
	_ = os.MkdirAll(dir, 0755)
	filePath := filepath.Join(dir, "detection_rules.json")

	mgr := &RulesConfigManager{filePath: filePath}
	mgr.Load()
	return mgr
}

func (m *RulesConfigManager) GetDefaultRules() []*DetectionRule {
	return []*DetectionRule{
		{Id: "rule-uniapp-h5", Name: "Uni-App (dev:h5)", Category: "Node/前端", MatchFile: "package.json", MatchContent: `"dev:h5":`, Command: "npm run dev:h5", DefaultPort: 8080, Enabled: true, IsBuiltin: true},
		{Id: "rule-vue-serve", Name: "Vue CLI (serve)", Category: "Node/前端", MatchFile: "package.json", MatchContent: `"serve":`, Command: "npm run serve", DefaultPort: 8080, Enabled: true, IsBuiltin: true},
		{Id: "rule-node-dev", Name: "前端开发服务 (dev)", Category: "Node/前端", MatchFile: "package.json", MatchContent: `"dev":`, Command: "pnpm dev", DefaultPort: 5173, Enabled: true, IsBuiltin: true},
		{Id: "rule-node-start", Name: "Node 服务 (start)", Category: "Node/前端", MatchFile: "package.json", MatchContent: `"start":`, Command: "npm start", DefaultPort: 3000, Enabled: true, IsBuiltin: true},
		{Id: "rule-dotnet-web", Name: "ASP.NET Core Web", Category: ".NET", MatchFile: "*.csproj", MatchContent: "Microsoft.NET.Sdk.Web", Command: "dotnet run", DefaultPort: 5000, PortExtractRegex: `https?://localhost:(\d+)`, Enabled: true, IsBuiltin: true},
		{Id: "rule-dotnet-worker", Name: ".NET 后台 Worker", Category: ".NET", MatchFile: "*.csproj", MatchContent: "Microsoft.NET.Sdk.Worker", Command: "dotnet run", DefaultPort: 0, Enabled: true, IsBuiltin: true},
		{Id: "rule-dotnet-exe", Name: ".NET 控制台/可执行程序", Category: ".NET", MatchFile: "*.csproj", MatchContent: "<OutputType>Exe</OutputType>", Command: "dotnet run", DefaultPort: 0, Enabled: true, IsBuiltin: true},
		{Id: "rule-py-django", Name: "Django Web 服务", Category: "Python", MatchFile: "manage.py", MatchContent: "", Command: "python manage.py runserver", DefaultPort: 8000, Enabled: true, IsBuiltin: true},
		{Id: "rule-py-fastapi", Name: "FastAPI / Uvicorn", Category: "Python", MatchFile: "main.py", MatchContent: "FastAPI", Command: "uvicorn main:app --reload", DefaultPort: 8000, Enabled: true, IsBuiltin: true},
		{Id: "rule-py-flask", Name: "Flask 服务", Category: "Python", MatchFile: "app.py", MatchContent: "Flask", Command: "python app.py", DefaultPort: 5000, Enabled: true, IsBuiltin: true},
		{Id: "rule-py-streamlit", Name: "Streamlit 数据看板", Category: "Python", MatchFile: "app.py", MatchContent: "streamlit", Command: "streamlit run app.py", DefaultPort: 8501, Enabled: true, IsBuiltin: true},
		{Id: "rule-py-script", Name: "Python 通用脚本", Category: "Python", MatchFile: "main.py", MatchContent: "", Command: "python main.py", DefaultPort: 0, Enabled: true, IsBuiltin: true},
		{Id: "rule-go-app", Name: "Go 应用程序", Category: "Go", MatchFile: "main.go", MatchContent: "", Command: "go run .", DefaultPort: 8080, Enabled: true, IsBuiltin: true},
		{Id: "rule-java-maven", Name: "Spring Boot (Maven)", Category: "Java", MatchFile: "pom.xml", MatchContent: "spring-boot", Command: "mvn spring-boot:run", DefaultPort: 8080, Enabled: true, IsBuiltin: true},
		{Id: "rule-java-gradle", Name: "Spring Boot (Gradle)", Category: "Java", MatchFile: "build.gradle", MatchContent: "spring-boot", Command: "gradle bootRun", DefaultPort: 8080, Enabled: true, IsBuiltin: true},
		{Id: "rule-rust-app", Name: "Rust 应用程序", Category: "Rust", MatchFile: "Cargo.toml", MatchContent: "", Command: "cargo run", DefaultPort: 8080, Enabled: true, IsBuiltin: true},
	}
}


func (m *RulesConfigManager) Load() []*DetectionRule {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil || len(data) == 0 {
		m.rules = m.GetDefaultRules()
		_ = m.saveInternal(m.rules)
		return m.rules
	}

	var rules []*DetectionRule
	if err := json.Unmarshal(data, &rules); err != nil || len(rules) == 0 {
		m.rules = m.GetDefaultRules()
		_ = m.saveInternal(m.rules)
		return m.rules
	}

	m.rules = rules
	return m.rules
}

func (m *RulesConfigManager) GetRules() []*DetectionRule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rules
}

func (m *RulesConfigManager) Save(rules []*DetectionRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules = rules
	return m.saveInternal(rules)
}

func (m *RulesConfigManager) ResetToDefault() []*DetectionRule {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules = m.GetDefaultRules()
	_ = m.saveInternal(m.rules)
	return m.rules
}

func (m *RulesConfigManager) saveInternal(rules []*DetectionRule) error {
	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.filePath, data, 0644)
}
