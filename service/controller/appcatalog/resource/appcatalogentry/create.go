package appcatalogentry

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/app/v4/pkg/key"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/to"
	"github.com/google/go-cmp/cmp"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/app-operator/v3/pkg/annotation"
	pkglabel "github.com/giantswarm/app-operator/v3/pkg/label"
	"github.com/giantswarm/app-operator/v3/pkg/project"
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

	// Skip creating appcatalogentry CRs if this is a community catalog.
	if key.AppCatalogType(cr) == communityCatalogType {
		r.logger.Debugf(ctx, "not creating CRs for catalog %#q with type %#q", cr.Name, communityCatalogType)
		r.logger.Debugf(ctx, "canceling resource")
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

	var created, updated, deleted int

	r.logger.Debugf(ctx, "finding out changes to appcatalogentries for catalog %#q", cr.Name)

	for name, desiredEntryCR := range desiredEntryCRs {
		currentEntryCR, ok := currentEntryCRs[name]
		if ok {
			// Copy current appCatalogEntry CR so we keep only the values we need
			// for comparing them.
			currentEntryCR = copyAppCatalogEntry(currentEntryCR)

			// Using reflect.DeepEqual doesn't work for the 2 date fields due to time
			// zones. Instead we compare the unix epoch and clear the date fields.
			timeComparer := cmp.Comparer(func(current, desired *metav1.Time) bool {
				if current != nil && desired != nil {
					return current.Unix() == desired.Unix()
				} else if current == nil && desired == nil {
					return true
				}

				return false
			})

			if cmp.Equal(currentEntryCR, desiredEntryCR, timeComparer) {
				// no-op
				continue
			}

			diff := cmp.Diff(currentEntryCR, desiredEntryCR, timeComparer)
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("appCatalogEntry %#q has to be updated", currentEntryCR.Name), "diff", fmt.Sprintf("(-current +desired):\n%s", diff))
			err := r.updateAppCatalogEntry(ctx, desiredEntryCR)
			if err != nil {
				return microerror.Mask(err)
			}

			updated++
		} else {
			err := r.createAppCatalogEntry(ctx, desiredEntryCR)
			if err != nil {
				return microerror.Mask(err)
			}

			created++
		}
	}

	// To keep the number of appCatalogEntry CR below a certain level,
	// we delete any appCatalogEntries older than the max entries.
	for name, currentEntryCR := range currentEntryCRs {
		_, ok := desiredEntryCRs[name]
		if !ok {
			err := r.deleteAppCatalogEntry(ctx, currentEntryCR)
			if err != nil {
				return microerror.Mask(err)
			}

			deleted++
		}
	}

	r.logger.Debugf(ctx, "created %d updated %d deleted %d appcatalogentries for catalog %#q", created, updated, deleted, cr.Name)

	return nil
}

func (r *Resource) createAppCatalogEntry(ctx context.Context, entryCR *v1alpha1.AppCatalogEntry) error {
	r.logger.Debugf(ctx, "creating appcatalogentry CR %#q in namespace %#q", entryCR.Name, entryCR.Namespace)

	_, err := r.k8sClient.G8sClient().ApplicationV1alpha1().AppCatalogEntries(entryCR.Namespace).Create(ctx, entryCR, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		r.logger.Debugf(ctx, "already created appcatalogentry CR %#q in namespace %#q", entryCR.Name, entryCR.Namespace)
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "created appcatalogentry CR %#q in namespace %#q", entryCR.Name, entryCR.Namespace)

	return nil
}

func (r *Resource) deleteAppCatalogEntry(ctx context.Context, entryCR *v1alpha1.AppCatalogEntry) error {
	r.logger.Debugf(ctx, "deleting appcatalogentry CR %#q in namespace %#q", entryCR.Name, entryCR.Namespace)

	err := r.k8sClient.G8sClient().ApplicationV1alpha1().AppCatalogEntries(entryCR.Namespace).Delete(ctx, entryCR.Name, metav1.DeleteOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "deleted appcatalogentry CR %#q in namespace %#q", entryCR.Name, entryCR.Namespace)

	return nil
}

func (r *Resource) updateAppCatalogEntry(ctx context.Context, entryCR *v1alpha1.AppCatalogEntry) error {
	r.logger.Debugf(ctx, "updating appcatalogentry CR %#q in namespace %#q", entryCR.Name, entryCR.Namespace)

	currentCR, err := r.k8sClient.G8sClient().ApplicationV1alpha1().AppCatalogEntries(entryCR.Namespace).Get(ctx, entryCR.Name, metav1.GetOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	entryCR.ResourceVersion = currentCR.ResourceVersion
	_, err = r.k8sClient.G8sClient().ApplicationV1alpha1().AppCatalogEntries(entryCR.Namespace).Update(ctx, entryCR, metav1.UpdateOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "updated appcatalogentry CR %#q in namespace %#q", entryCR.Name, entryCR.Namespace)

	return nil
}

func (r *Resource) newAppCatalogEntries(ctx context.Context, cr v1alpha1.AppCatalog, index index) (map[string]*v1alpha1.AppCatalogEntry, error) {
	var err error
	entryCRs := map[string]*v1alpha1.AppCatalogEntry{}

	for _, entries := range index.Entries {
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Created.After(entries[j].Created.Time)
		})

		latestVersion := entries[0].Version

		maxEntries := r.maxEntriesPerApp
		if len(entries) < maxEntries {
			maxEntries = len(entries)
		}

		for i := 0; i < maxEntries; i++ {
			e := entries[i]
			name := fmt.Sprintf("%s-%s-%s", cr.Name, e.Name, e.Version)

			var rawMetadata []byte
			{
				if url, ok := e.Annotations[annotation.Metadata]; ok {
					rawMetadata, err = r.getMetadata(ctx, url)
					if err != nil {
						r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to get appMetadata for entry %#q in catalog %#q", e.Name, cr.Name), "stack", fmt.Sprintf("%#v", err))
						continue
					}
				}
			}

			// Until we add support for appMetadata files the updated time will be
			// the same as the created time.
			updatedTime := e.Created.DeepCopy()

			var m *appMetadata
			{
				if rawMetadata != nil {
					m, err = parseMetadata(rawMetadata)
					if err != nil {
						return nil, microerror.Mask(err)
					}
				}
			}

			var isLatest bool

			// We set the latest label to true for easier filtering.
			if e.Version == latestVersion {
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
						label.AppKubernetesName:    e.Name,
						label.AppKubernetesVersion: e.Version,
						label.CatalogName:          cr.Name,
						label.CatalogType:          key.AppCatalogType(cr),
						pkglabel.Latest:            strconv.FormatBool(isLatest),
						label.ManagedBy:            key.AppCatalogEntryManagedBy(project.Name()),
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
					AppName:    e.Name,
					AppVersion: e.AppVersion,
					Catalog: v1alpha1.AppCatalogEntrySpecCatalog{
						Name: cr.Name,
						// Namespace will be empty until appcatalog CRs become namespace scoped.
						Namespace: "",
					},
					Chart: v1alpha1.AppCatalogEntrySpecChart{
						Home: e.Home,
						Icon: e.Icon,
					},
					Version: e.Version,
				},
			}

			if m != nil {
				entryCR.Annotations = m.Annotations
				entryCR.Spec.Chart.APIVersion = m.ChartAPIVersion
				entryCR.Spec.Restrictions = m.Restrictions
				entryCR.Spec.DateCreated = m.DataCreated
				entryCR.Spec.DateUpdated = m.DataCreated
			}

			if entryCR.Spec.Chart.APIVersion == "" {
				// chartAPIVersion default is `v1`.
				entryCR.Spec.Chart.APIVersion = "v1"
			}

			if entryCR.Spec.DateCreated == nil {
				// If meta.yaml does not have dateCreated, use the timestamp from app.
				entryCR.Spec.DateCreated = &e.Created
				entryCR.Spec.DateUpdated = updatedTime
			}

			entryCRs[name] = entryCR
		}
	}

	return entryCRs, nil
}

func (r *Resource) getLatestVersion(ctx context.Context, entries []entry) (string, error) {
	var latestVersion semver.Version
	var latestCreated *metav1.Time

	for i := 0; i < len(entries); i++ {
		v, err := semver.NewVersion(entries[i].Version)
		if errors.As(err, &semver.ErrInvalidSemVer) {
			r.logger.Debugf(ctx, "invalid semver from converting app entries %s, version is %s", entries[i].Name, entries[i].Version)
			continue
		} else if err != nil {
			return "", microerror.Mask(err)
		}

		// Removing Prerelease from version since they are mostly SHA strings which we could not compare the size.
		nextVersion, err := v.SetPrerelease("")
		if err != nil {
			return "", microerror.Mask(err)
		}

		if nextVersion.GreaterThan(&latestVersion) {
			latestVersion = nextVersion
			latestCreated = entries[i].Created.DeepCopy()
			continue
		}

		if nextVersion.Equal(&latestVersion) {
			if latestCreated.After(entries[i].Created.Time) {
				latestVersion = nextVersion
				latestCreated = entries[i].Created.DeepCopy()
			}
		}
	}

	return latestVersion.String(), nil
}
