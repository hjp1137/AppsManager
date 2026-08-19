package core

type ProjectStatus int

const (
	StatusStopped ProjectStatus = 0
	StatusStarting ProjectStatus = 1
	StatusRunning ProjectStatus = 2
)

type SubService struct {
	Id           string        `json:"Id"`
	Name         string        `json:"Name"`
	ProjectName  string        `json:"ProjectName,omitempty"`
	Path         string        `json:"Path"`
	StartCommand string        `json:"StartCommand"`
	Port         int           `json:"Port"`
	Status       ProjectStatus `json:"Status"`
	ProcessId    *int          `json:"ProcessId,omitempty"`
	LogContent   string        `json:"LogContent,omitempty"`
}


type ProjectItem struct {
	Id          string        `json:"Id"`
	Name        string        `json:"Name"`
	Path        string        `json:"Path"`
	Group       string        `json:"Group"`
	SubServices []*SubService `json:"SubServices"`
}

type WSMessage struct {
	Type    string      `json:"type"`    // "log", "status", "sync", "port_conflict"
	ServiceId string    `json:"serviceId,omitempty"`
	Status  ProjectStatus `json:"status,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type DetectionRule struct {
	Id               string `json:"id"`
	Name             string `json:"name"`
	Category         string `json:"category"`
	MatchFile        string `json:"matchFile"`
	MatchContent     string `json:"matchContent"`
	Command          string `json:"command"`
	DefaultPort      int    `json:"defaultPort"`
	PortExtractRegex string `json:"portExtractRegex"`
	Enabled          bool   `json:"enabled"`
	IsBuiltin        bool   `json:"isBuiltin"`
}

