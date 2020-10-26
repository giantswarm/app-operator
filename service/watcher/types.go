package configmap

type resourceType string

const (
	configMapType resourceType = "configmap"
	secretType    resourceType = "secret"
)

type appIndex struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type resourceIndex struct {
	ResourceType resourceType
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
}

type patch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}
