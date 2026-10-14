package types

import "time"

type ContainerSpec struct {
	Name      string              `json:"name"`
	DiskImage *SandboxDiskImage   `json:"diskImage,omitempty"`
	Command   []string            `json:"command,omitempty"`
	Args      []string            `json:"args,omitempty"`
	Env       map[string]string   `json:"env,omitempty"`
	Resources *ContainerResources `json:"resources,omitempty"`
}

type ContainerStatus struct {
	Name           string     `json:"name"`
	State          string     `json:"state"`
	Ready          bool       `json:"ready"`
	Started        bool       `json:"started"`
	RestartCount   int        `json:"restartCount"`
	Reason         string     `json:"reason,omitempty"`
	Message        string     `json:"message,omitempty"`
	LastExitCode   *int       `json:"lastExitCode,omitempty"`
	LastStartedAt  *time.Time `json:"lastStartedAt,omitempty"`
	LastFinishedAt *time.Time `json:"lastFinishedAt,omitempty"`
}
