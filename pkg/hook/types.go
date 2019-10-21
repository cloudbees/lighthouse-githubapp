package hook

// WorkspaceAccess returns details of the rocket for connecting to it
type WorkspaceAccess struct {
	Project   string `json:"project,omitempty"`
	Cluster   string `json:"cluster,omitempty"`
	Region    string `json:"region,omitempty"`
	Zone      string `json:"zone,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// LogFields returns fields for logging
func (w *WorkspaceAccess) LogFields() map[string]interface{} {
	return map[string]interface{}{
		"Project": w.Project,
		"Cluster": w.Cluster,
		"Region":  w.Region,
	}
}

type RepositoryInfo struct {
	URL string `json:"url,omitempty"`
}
