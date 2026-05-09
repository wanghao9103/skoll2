package plugin

type Status string

const (
	StatusNotInstalled Status = "NOT_INSTALLED"
	StatusInstalled    Status = "INSTALLED"
	StatusEnabled      Status = "ENABLED"
	StatusDisabled     Status = "DISABLED"
	StatusUpgradable   Status = "UPGRADABLE"
)

type Item struct {
	Name          string   `json:"name"`
	Key           string   `json:"key"`
	Type          string   `json:"type"`
	RuntimeSupported bool   `json:"runtimeSupported"`
	RuntimeReason    string `json:"runtimeReason,omitempty"`
	Version       string   `json:"version"`
	Description   string   `json:"description"`
	Icon          string   `json:"icon"`
	Status        Status   `json:"status"`
	APIPrefix     string   `json:"apiPrefix"`
	FrontendEntry string   `json:"frontendEntry"`
	Menus         []Menu   `json:"menus"`
	Permissions   []string `json:"permissions"`
}

type Menu struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	Component    string `json:"component"`
	Icon         string `json:"icon"`
	RemoteModule string `json:"remoteModule,omitempty"`
}
