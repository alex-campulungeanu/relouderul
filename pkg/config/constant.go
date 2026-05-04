package config

const configFileName = "config.json"
const configFileDir = "relouderul"

type ServiceInfo struct {
	Path       string   `json:"path"`
	Name       string   `json:"name"`
	Command    []string `json:"command"`
	WatchPath  string   `json:"watch_path"`
	Extensions []string `json:"extensions"`
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
