package model

import "time"

const (
	NodeStatusOnline  = "online"
	NodeStatusOffline = "offline"

	TaskStatusPending = "pending"
	TaskStatusRunning = "running"
	TaskStatusSuccess = "success"
	TaskStatusFailed  = "failed"
)

type State struct {
	Nodes          []Node          `json:"nodes"`
	Groups         []Group         `json:"groups"`
	Pools          []Pool          `json:"pools"`
	RegisterTokens []RegisterToken `json:"registerTokens"`
	Tasks          []Task          `json:"tasks"`
	Settings       Settings        `json:"settings"`
}

type Settings struct {
	ProxyUsername string `json:"proxyUsername"`
	ProxyPassword string `json:"proxyPassword"`
}

type Node struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	PublicIP           string    `json:"publicIp"`
	Hostname           string    `json:"hostname"`
	OS                 string    `json:"os"`
	Arch               string    `json:"arch"`
	Status             string    `json:"status"`
	LastSeenAt         time.Time `json:"lastSeenAt"`
	AgentToken         string    `json:"agentToken"`
	AgentVersion       string    `json:"agentVersion"`
	GostVersion        string    `json:"gostVersion"`
	GostStatus         string    `json:"gostStatus"`
	ConfigVersion      int       `json:"configVersion"`
	GroupIDs           []string  `json:"groupIds"`
	HTTPPort           int       `json:"httpPort"`
	SocksPort          int       `json:"socksPort"`
	TodayUploadBytes   int64     `json:"todayUploadBytes"`
	TodayDownloadBytes int64     `json:"todayDownloadBytes"`
	TotalUploadBytes   int64     `json:"totalUploadBytes"`
	TotalDownloadBytes int64     `json:"totalDownloadBytes"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type Group struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Remark    string    `json:"remark"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Pool struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	GroupIDs  []string  `json:"groupIds"`
	HTTPPort  int       `json:"httpPort"`
	SocksPort int       `json:"socksPort"`
	Strategy  string    `json:"strategy"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type RegisterToken struct {
	Token     string    `json:"token"`
	Name      string    `json:"name"`
	ExpiresAt time.Time `json:"expiresAt"`
	Used      bool      `json:"used"`
	CreatedAt time.Time `json:"createdAt"`
}

type Task struct {
	ID         string    `json:"id"`
	NodeID     string    `json:"nodeId"`
	Type       string    `json:"type"`
	Status     string    `json:"status"`
	Payload    string    `json:"payload"`
	Result     string    `json:"result"`
	Error      string    `json:"error"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`
}
