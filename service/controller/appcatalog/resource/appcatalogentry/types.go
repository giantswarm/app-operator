package appcatalogentry

type index struct {
	Entries   map[string][]entry `json:"entries"`
	Generated string             `json:"generated"`
}

type entry struct {
	AppVersion string   `json:"appVersion"`
	Created    string   `json:"created"`
	Home       string   `json:"home"`
	Icon       string   `json:"icon"`
	Name       string   `json:"name"`
	Urls       []string `json:"urls"`
	Version    string   `json:"version"`
}
