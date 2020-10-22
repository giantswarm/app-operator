package appcatalogentry

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/Masterminds/semver"
	"github.com/giantswarm/apiextensions/v2/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v2/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/to"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkglabel "github.com/giantswarm/app-operator/v2/pkg/label"
	"github.com/giantswarm/app-operator/v2/service/controller/key"
)

// EnsureCreated ensures appcatalogentry CRs are created or updated for this
// appcatalog CR.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	if !r.uniqueApp {
		// Return early. Only unique instance manages appcatalogentry CRs.
		return nil
	}

	cr, err := key.ToAppCatalog(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	// Skip creating appcatalogentry CRs if the catalog is not public.
	if key.CatalogVisibility(cr) != publicVisibilityType {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("not creating CRs for catalog %#q with visibility %#q", cr.Name, key.CatalogVisibility(cr)))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}
	// Skip creating appcatalogentry CRs if this is a community catalog.
	if key.CatalogType(cr) == communityCatalogType {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("not creating CRs for catalog %#q with type %#q", cr.Name, communityCatalogType))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	currentEntryCRs, err := r.getCurrentEntryCRs(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	index, err := r.getIndex(ctx, key.AppCatalogStorageURL(cr))
	if err != nil {
		return microerror.Mask(err)
	}

	desiredEntryCRs, err := r.newAppCatalogEntries(ctx, cr, index)
	if err != nil {
		return microerror.Mask(err)
	}

	var created, updated int

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("finding out changes to appcatalogentries for catalog %#q", cr.Name))

	for name, desiredEntryCR := range desiredEntryCRs {
		currentEntryCR, ok := currentEntryCRs[name]
		if ok {
			if !equals(currentEntryCR, desiredEntryCR) {
				err := r.updateAppCatalogEntry(ctx, desiredEntryCR)
				if err != nil {
					return microerror.Mask(err)
				}

				updated++
			}
		} else {
			err := r.createAppCatalogEntry(ctx, desiredEntryCR)
			if err != nil {
				return microerror.Mask(err)
			}

			created++
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created %d updated %d appcatalogentries for catalog %#q", created, updated, cr.Name))

	return nil
}

func (r *Resource) createAppCatalogEntry(ctx context.Context, entryCR *v1alpha1.AppCatalogEntry) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating appcatalogentry CR %#q in namespace %#q", entryCR.Name, entryCR.Namespace))

	_, err := r.k8sClient.G8sClient().ApplicationV1alpha1().AppCatalogEntries(entryCR.Namespace).Create(ctx, entryCR, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("already created appcatalogentry CR %#q in namespace %#q", entryCR.Name, entryCR.Namespace))
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created appcatalogentry CR %#q in namespace %#q", entryCR.Name, entryCR.Namespace))

	return nil
}

func (r *Resource) updateAppCatalogEntry(ctx context.Context, entryCR *v1alpha1.AppCatalogEntry) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating appcatalogentry CR %#q in namespace %#q", entryCR.Name, entryCR.Namespace))

	currentCR, err := r.k8sClient.G8sClient().ApplicationV1alpha1().AppCatalogEntries(entryCR.Namespace).Get(ctx, entryCR.Name, metav1.GetOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	entryCR.ResourceVersion = currentCR.ResourceVersion
	_, err = r.k8sClient.G8sClient().ApplicationV1alpha1().AppCatalogEntries(entryCR.Namespace).Update(ctx, entryCR, metav1.UpdateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated appcatalogentry CR %#q in namespace %#q", entryCR.Name, entryCR.Namespace))

	return nil
}

func (r *Resource) newAppCatalogEntries(ctx context.Context, cr v1alpha1.AppCatalog, index index) (map[string]*v1alpha1.AppCatalogEntry, error) {
	entryCRs := map[string]*v1alpha1.AppCatalogEntry{}

	for name, entries := range index.Entries {
		latestVersion, err := parseLatestVersion(entries)
		if err != nil {
			// Log error but continue generating CRs.
			r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to parse latest version for %#q in catalog %#q", name, cr.Name), "stack", fmt.Sprintf("%#v", err))
		}

		for _, entry := range entries {
			name := fmt.Sprintf("%s-%s-%s", cr.Name, entry.Name, entry.Version)

			createdTime, err := parseTime(entry.Created)
			if err != nil {
				return nil, microerror.Mask(err)
			}

			// Until we add support for metadata files the updated time will be
			// the same as the created time.
			updatedTime := createdTime

			var isLatest bool

			// We set the latest label to true for easier filtering.
			if entry.Version == latestVersion {
				isLatest = true
			}

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
						pkglabel.Latest:            strconv.FormatBool(isLatest),
						label.ManagedBy:            key.AppCatalogEntryManagedBy(),
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
						// Namespace will be empty until appcatalog CRs become namespace scoped.
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

func equals(current, desired *v1alpha1.AppCatalogEntry) bool {
	if current.Name != desired.Name {
		return false
	}
	if !reflect.DeepEqual(current.Labels, desired.Labels) {
		return false
	}

	return reflect.DeepEqual(current.Spec, desired.Spec)
}

func parseLatestVersion(entries []entry) (string, error) {
	if len(entries) == 0 {
		return "", nil
	}

	versions := make([]*semver.Version, len(entries))

	for i, entry := range entries {
		v, err := semver.NewVersion(entry.Version)
		if err != nil {
			return "", microerror.Mask(err)
		}

		versions[i] = v
	}

	// Sort the versions semantically and return the latest.
	sort.Sort(semver.Collection(versions))
	latest := versions[len(versions)-1]

	return latest.String(), nil
}

func parseTime(created string) (*metav1.Time, error) {
	rawTime, err := time.Parse(time.RFC3339, created)
	if err != nil {
		return nil, microerror.Maskf(executionFailedError, "wrong timestamp format %#q", created)
	}
	timeVal := metav1.NewTime(rawTime)

	return &timeVal, nil
}
