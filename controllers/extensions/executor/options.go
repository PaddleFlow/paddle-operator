package executor

type SyncJobOptions struct {
	Source string `json:"source,omitempty"`
	Start  string `json:"start,omitempty"`
	End    string `json:"end,omitempty"`
}

type WarmupJobOptions struct {
	Path string `json:"start,omitempty"`
}

type RmrJobOptions struct {
	Path string `json:"start,omitempty"`
}

type ClearJobOptions struct {
	Path string `json:"start,omitempty"`
}
