package types

type SandboxSource struct {
	DiskImage *SandboxDiskImage `json:"diskImage,omitempty"`
	Pod       *SandboxPod       `json:"pod,omitempty"`
}

type SandboxDiskImage struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	IsPublic bool   `json:"isPublic,omitempty"`
}

type SandboxPod struct {
	Containers    []ContainerSpec `json:"containers"`
	RestartPolicy string          `json:"restartPolicy,omitempty"`
}
