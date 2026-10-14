package types

import "time"

type DiskImage struct {
	ID       string            `json:"id,omitempty"`
	Name     string            `json:"name,omitempty"`
	Labels   map[string]string `json:"labels,omitempty"`
	Source   *DiskImageSource  `json:"source,omitempty"`
	Image    *DiskImageInfo    `json:"image,omitempty"`
	Status   *DiskImageStatus  `json:"status,omitempty"`
	SizeInMB int64             `json:"sizeInMB,omitempty"`
}

type DiskImageSource struct {
	Kind           string                  `json:"kind"`
	ImageURL       string                  `json:"imageUrl,omitempty"`
	Authentication *RegistryAuthentication `json:"authentication,omitempty"`
}

const DiskImageSourceKindRegistry = "registry"

const (
	DiskImageStateReady     = "Ready"
	DiskImageStateSucceeded = "Succeeded"
	DiskImageStateFailed    = "Failed"
)

type RegistryAuthentication struct {
	RegistryCredentials *RegistryCredentials           `json:"registryCredentials,omitempty"`
	Identity            *ManagedIdentityAuthentication `json:"identity,omitempty"`
}

type RegistryCredentials struct {
	Username string `json:"username"`
	Token    string `json:"token"`
}

type ManagedIdentityAuthentication struct {
	Type               string `json:"type"`
	IdentityResourceID string `json:"identityResourceId,omitempty"`
}

type DiskImageInfo struct {
	Base       string   `json:"base"`
	Entrypoint []string `json:"entrypoint,omitempty"`
	Cmd        []string `json:"cmd,omitempty"`
}

type DiskImageStatus struct {
	State        string    `json:"state"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
	CreatedAt    time.Time `json:"createdAt,omitzero"`
	UpdatedAt    time.Time `json:"updatedAt,omitzero"`
}
