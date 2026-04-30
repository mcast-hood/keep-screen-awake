package transport

// Command identifies the IPC operation requested by the CLI.
type Command string

const (
	CmdStatus Command = "status"
	CmdOn     Command = "on"
	CmdOff    Command = "off"
	CmdMode   Command = "mode"
	CmdLogs   Command = "logs"
)

// Request is the JSON envelope sent from CLI to daemon.
type Request struct {
	Command Command `json:"command"`
	Mode    string  `json:"mode,omitempty"`
	Lines   int     `json:"lines,omitempty"`
}

// Response is the JSON envelope returned by the daemon.
type Response struct {
	OK    bool   `json:"ok"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

// StatusData is the payload for a status response.
type StatusData struct {
	Mode        string   `json:"mode"`
	AwakeActive bool     `json:"awake_active"`
	DisplayOnly bool     `json:"display_only"`
	Schedule    []Window `json:"schedule,omitempty"`
}

// Window mirrors config.Window for transport use.
type Window struct {
	Start string   `json:"start"`
	End   string   `json:"end"`
	Days  []string `json:"days"`
}

// LogsData is the payload for a logs response.
type LogsData struct {
	Lines []string `json:"lines"`
}
