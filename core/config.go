package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type ConfigManager struct {
	mu       sync.RWMutex
	filePath string
	Projects []*ProjectItem
}

func NewConfigManager() *ConfigManager {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		localAppData = "."
	}
	folderPath := filepath.Join(localAppData, "AppsManager")
	_ = os.MkdirAll(folderPath, 0755)

	cm := &ConfigManager{
		filePath: filepath.Join(folderPath, "projects.json"),
		Projects: make([]*ProjectItem, 0),
	}
	cm.Load()
	return cm
}

func (cm *ConfigManager) Load() []*ProjectItem {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	data, err := os.ReadFile(cm.filePath)
	if err != nil {
		cm.Projects = make([]*ProjectItem, 0)
		return cm.Projects
	}

	var projects []*ProjectItem
	if err := json.Unmarshal(data, &projects); err != nil {
		cm.Projects = make([]*ProjectItem, 0)
		return cm.Projects
	}

	// 强制重置启动状态为已停止
	for _, p := range projects {
		for _, svc := range p.SubServices {
			svc.Status = StatusStopped
			svc.ProcessId = nil
		}
	}

	cm.Projects = projects
	return cm.Projects
}

func (cm *ConfigManager) Save(projects []*ProjectItem) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 保存时同样清除运行态脏数据
	for _, p := range projects {
		for _, svc := range p.SubServices {
			svc.Status = StatusStopped
			svc.ProcessId = nil
		}
	}

	cm.Projects = projects
	data, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cm.filePath, data, 0644)
}


func (cm *ConfigManager) GetProjects() []*ProjectItem {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.Projects
}
