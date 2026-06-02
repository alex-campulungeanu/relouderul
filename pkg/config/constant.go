package config

const configFileName = "config.json"
const configFileDir = "relouderul"

type ServiceInfo struct {
	Path       string   `json:"path" validate:"required"`
	Name       string   `json:"name" validate:"required"`
	Command    []string `json:"command" validate:"required"`
	WatchPath  string   `json:"watch_path" validate:"required"`
	Extensions []string `json:"extensions" validate:"required"`
}

type ConfigStructure map[string]ServiceInfo

var configTemplate = ConfigStructure{
	"first": {
		Path:       "full service path 1",
		Name:       "service 1 name",
		Command:    []string{"first", "second"},
		WatchPath:  "full service 1 path",
		Extensions: []string{".py", ".yaml"},
	},
	"second": {
		Path:       "full service path 2",
		Name:       "service 2 name",
		Command:    []string{"first", "second"},
		WatchPath:  "full service 2 path",
		Extensions: []string{".py", ".yaml"},
	},
}
