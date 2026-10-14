package types

import "time"

type Sandbox struct {
	ID                string            `json:"id,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	Entrypoint        []string          `json:"entrypoint,omitempty"`
	Cmd               []string          `json:"cmd,omitempty"`
	Environment       map[string]string `json:"environment,omitempty"`
	SourcesRef        *SandboxSource    `json:"sourcesRef,omitempty"`
	Resources         *SandboxResources `json:"resources,omitempty"`
	CreatedAt         time.Time         `json:"createdAt,omitzero"`
	State             string            `json:"state,omitempty"`
	StateDetails      *SandboxStateInfo `json:"stateDetails,omitempty"`
	Ports             []SandboxPort     `json:"ports,omitempty"`
	ContainerStatuses []ContainerStatus `json:"containerStatuses,omitempty"`
}

type PagedSandbox struct {
	Value    []Sandbox `json:"value"`
	NextLink string    `json:"nextLink,omitempty"`
}

const (
	SandboxStateRunning    = "Running"
	SandboxStateStopped    = "Stopped"
	SandboxStateIdle       = "Idle"
	SandboxStateCreating   = "Creating"
	SandboxStateStopping   = "Stopping"
	SandboxStateStopFailed = "StopFailed"
)

type SandboxStateInfo struct {
	StoppedReason string    `json:"stoppedReason"`
	StoppedAt     time.Time `json:"stoppedAt"`
}

type SandboxPort struct {
	Name string `json:"name,omitempty"`
	Port int    `json:"port"`
	URL  string `json:"url,omitempty"`
}
