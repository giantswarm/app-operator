package appcatalogentry

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v2/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/to"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkglabel "github.com/giantswarm/app-operator/v2/pkg/label"
	"github.com/giantswarm/app-operator/v2/pkg/project"
	appkey "github.com/giantswarm/app-operator/v2/service/controller/app/key"
	"github.com/giantswarm/app-operator/v2/service/controller/appcatalog/key"
)

// EnsureCreated ensures appcatalogentry CRs are created or updated for this
// appcatalog CR.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	if !r.uniqueApp {
		// Return early. Only unique instance manages appcatalogentry CRs.
		return nil
	}

	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if key.CatalogVisibility(cr) != publicVisibilityType {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("not creating CRs for catalog %#q with visibility %#q", cr.Name, key.CatalogVisibility(cr)))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("getting index.yaml for %#q appcatalog from %#q", cr.Name, appkey.AppCatalogStorageURL(cr)))

	index, err := getIndex(appkey.AppCatalogStorageURL(cr))
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("got index.yaml for %#q appcatalog", cr.Name))

	desiredEntryCRs, err := newAppCatalogEntries(ctx, cr, index)
	if err != nil {
		return microerror.Mask(err)
	}

	currentEntryCRs, err := r.getCurrentEntryCRs(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	for name, entry := range desiredEntryCRs {
		_, ok := currentEntryCRs[name]
		if ok {
			continue
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating appcatalogentry CR %#q in namespace %#q", name, entry.Namespace))

		_, err = r.k8sClient.G8sClient().ApplicationV1alpha1().AppCatalogEntries(entry.Namespace).Create(ctx, entry, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("already created appcatalogentry CR %#q in namespace %#q", name, entry.Namespace))
			continue
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created appcatalogentry CR %#q in namespace %#q", name, entry.Namespace))
	}

	return nil
}

func newAppCatalogEntries(ctx context.Context, cr v1alpha1.AppCatalog, index index) (map[string]*v1alpha1.AppCatalogEntry, error) {
	entryCRs := map[string]*v1alpha1.AppCatalogEntry{}

	for _, entries := range index.Entries {
		for _, entry := range entries {
			name := fmt.Sprintf("%s-%s-%s", cr.Name, entry.Name, entry.Version)

			createdTime, err := parseTime(entry.Created)
			if err != nil {
				return nil, microerror.Mask(err)
			}

			// Until we add support for metadata files the updated time will be
			// the same as the created time.
			updatedTime := createdTime

			entryCR := &v1alpha1.AppCatalogEntry{
				TypeMeta: metav1.TypeMeta{
					APIVersion: apiVersion,
					Kind:       kindAppCatalogEntry,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: metav1.NamespaceDefault,
					Labels: map[string]string{
						pkglabel.AppKubernetesName: entry.Name,
						pkglabel.CatalogName:       cr.Name,
						pkglabel.CatalogType:       key.CatalogType(cr),
						label.ManagedBy:            project.Name(),
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         apiVersion,
							BlockOwnerDeletion: to.BoolP(true),
							Kind:               kindAppCatalog,
							Name:               cr.Name,
							UID:                cr.UID,
						},
					},
				},
				Spec: v1alpha1.AppCatalogEntrySpec{
					AppName:    entry.Name,
					AppVersion: entry.AppVersion,
					Catalog: v1alpha1.AppCatalogEntrySpecCatalog{
						Name: cr.Name,
						// Namespace will be empty until appcatalog CR becomes namespace scoped.
						Namespace: "",
					},
					Chart: v1alpha1.AppCatalogEntrySpecChart{
						Home: entry.Home,
						Icon: entry.Icon,
					},
					DateCreated: createdTime,
					DateUpdated: updatedTime,
					Version:     entry.Version,
				},
			}

			entryCRs[name] = entryCR
		}
	}

	return entryCRs, nil
}
