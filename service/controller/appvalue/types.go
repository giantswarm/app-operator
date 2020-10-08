package appvalue

type index struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type patch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}
