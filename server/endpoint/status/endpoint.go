package status

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	kitendpoint "github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Method is the HTTP method this endpoint is register for.
	Method = "PATCH"
	// Name identifies the endpoint. It is aligned to the package path.
	Name = "status/updater"
	// Path is the HTTP request path this endpoint is registered for.
	Path = "/status/{app_namespace}/{app_name}/"
)

type Config struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
}

type Endpoint struct {
	k8sClient k8sclient.Interface
	logger    micrologger.Logger
}

func New(config Config) (*Endpoint, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	e := &Endpoint{
		k8sClient: config.K8sClient,
		logger:    config.Logger,
	}

	return e, nil
}

func (e Endpoint) Decoder() kithttp.DecodeRequestFunc {
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		var request Request

		defer r.Body.Close()
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			return nil, microerror.Maskf(decodeFailedError, "%v", err.Error())
		}

		namespace, _ := ctx.Value("app_namespace").(string)
		name, _ := ctx.Value("app_name").(string)
		request.AppNamespace = namespace
		request.AppName = name

		return request, nil
	}
}

func (e Endpoint) Encoder() kithttp.EncodeResponseFunc {
	return func(ctx context.Context, w http.ResponseWriter, response interface{}) error {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		return json.NewEncoder(w).Encode(response)
	}
}

func (e Endpoint) Endpoint() kitendpoint.Endpoint {
	return func(ctx context.Context, r interface{}) (interface{}, error) {
		var err error

		request := r.(Request)

		desiredStatus := v1alpha1.AppStatus{
			AppVersion: request.AppVersion,
			Release: v1alpha1.AppStatusRelease{
				LastDeployed: request.LastDeployed,
				Reason:       request.Reason,
				Status:       request.Status,
			},
			Version: request.Version,
		}

		// Get app CR again to ensure the resource version is correct.
		currentCR, err := e.k8sClient.G8sClient().ApplicationV1alpha1().Apps(request.AppNamespace).Get(ctx, request.AppName, metav1.GetOptions{})
		if err != nil {
			return nil, microerror.Mask(err)
		}

		if equals(currentCR.Status, desiredStatus) {
			// no-op
			return nil, nil
		}

		e.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("setting status for app %#q in namespace %#q", request.AppName, request.AppNamespace))

		currentCR.Status = desiredStatus

		_, err = e.k8sClient.G8sClient().ApplicationV1alpha1().Apps(request.AppNamespace).UpdateStatus(ctx, currentCR, metav1.UpdateOptions{})
		if err != nil {
			return nil, microerror.Mask(err)
		}

		e.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("status set for app %#q in namespace %#q", request.AppName, request.AppNamespace))

		return nil, nil
	}
}

func (e Endpoint) Method() string {
	return Method
}

func (e Endpoint) Middlewares() []kitendpoint.Middleware {
	return []kitendpoint.Middleware{}
}

func (e Endpoint) Name() string {
	return Name
}

func (e Endpoint) Path() string {
	return Path
}

// equals asseses the equality of AppStatuses with regards to distinguishing
// fields.
func equals(a, b v1alpha1.AppStatus) bool {
	if a.AppVersion != b.AppVersion {
		return false
	}
	if a.Release.LastDeployed != b.Release.LastDeployed {
		return false
	}
	if a.Release.Reason != b.Release.Reason {
		return false
	}
	if a.Release.Status != b.Release.Status {
		return false
	}
	if a.Version != b.Version {
		return false
	}

	return true
}
