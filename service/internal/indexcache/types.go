package indexcache

type Index struct {
	Entries map[string][]Entry `json:"entries"`
}

type Entry struct {
	Urls    []string `json:"urls"`
	Version string   `json:"version"`
}
