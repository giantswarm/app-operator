# app-operator Helm Chart
Helm Chart for the app-operator

## Configuration

| Parameter          | Description                                 | Default                                                           |
|--------------------|---------------------------------------------|-------------------------------------------------------------------|
| `name`             | The name of the operator                    | `app-operator`                                                  |
| `namespace`        | The namespaces of the operator              | `giantswarm`                                                      |
| `port`             | The port of the operator container          | `8000`                                                            |
| `image.registry`   | The operator container image registry       | `quay.io`                                                         |
| `image.repository` | The operator container image repository     | `giantswarm/app-operator`                                       |
| `image.tag`        | The operator container image tag            | `[latest commit SHA]`                                             |
| `resources`        | The operator pod resource requests & limits | `request: cpu:100m, memory:75Mi;  limits: cpu:250m, memory:250Mi` |
