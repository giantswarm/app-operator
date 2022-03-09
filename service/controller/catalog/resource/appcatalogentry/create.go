package appcatalogentry

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v6/pkg/key"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/to"
	"github.com/google/go-cmp/cmp"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	pkglabel "github.com/giantswarm/app-operator/v5/pkg/label"
	"github.com/giantswarm/app-operator/v5/pkg/project"
)

// EnsureCreated ensures appcatalogentry CRs are created or updated for this
// catalog CR.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	if !r.uniqueApp {
		// Return early. Only unique instance manages appcatalogentry CRs.
		return nil
	}

	cr, err := key.ToCatalog(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	// Skip creating appcatalogentry CRs if this is a community catalog.
	if key.CatalogType(cr) == communityCatalogType {
		r.logger.Debugf(ctx, "not creating CRs for catalog %#q with type %#q", cr.Name, communityCatalogType)
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	currentEntryCRs, err := r.getCurrentEntryCRs(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	index, err := r.getIndex(ctx, key.CatalogStorageURL(cr))
	if err != nil {
		return microerror.Mask(err)
	}

	desiredEntryCRs, err := r.newAppCatalogEntries(ctx, cr, index)
	if err != nil {
		return microerror.Mask(err)
	}

	var created, updated, deleted, errored int

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
				// Log error but continue processing other CRs.
				r.logger.Errorf(ctx, err, "failed to create appCatalogEntry %#q", currentEntryCR.Name)
				errored++
			}

			updated++
		} else {
			err := r.createAppCatalogEntry(ctx, desiredEntryCR)
			if err != nil {
				// Log error but continue processing other CRs.
				r.logger.Errorf(ctx, err, "failed to update appCatalogEntry %#q", currentEntryCR.Name)
				errored++
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
				// Log error but continue processing other CRs.
				r.logger.Errorf(ctx, err, "failed to update appCatalogEntry %#q", currentEntryCR.Name)
				errored++
			}

			deleted++
		}
	}

	r.logger.Debugf(ctx, "created %d updated %d deleted %d appcatalogentries for catalog %#q", created, updated, deleted, cr.Name)
	if errored > 0 {
		r.logger.Debugf(ctx, "failed to process %d appcatalogentries for catalog %#q", errored, cr.Name)
	}

	return nil
}

func (r *Resource) createAppCatalogEntry(ctx context.Context, entryCR *v1alpha1.AppCatalogEntry) error {
	r.logger.Debugf(ctx, "creating appcatalogentry CR %#q in namespace %#q", entryCR.Name, entryCR.Namespace)

	err := r.k8sClient.CtrlClient().Create(ctx, entryCR)
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

	err := r.k8sClient.CtrlClient().Delete(ctx, entryCR)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "deleted appcatalogentry CR %#q in namespace %#q", entryCR.Name, entryCR.Namespace)

	return nil
}

func (r *Resource) updateAppCatalogEntry(ctx context.Context, entryCR *v1alpha1.AppCatalogEntry) error {
	r.logger.Debugf(ctx, "updating appcatalogentry CR %#q in namespace %#q", entryCR.Name, entryCR.Namespace)

	var currentCR v1alpha1.AppCatalogEntry

	err := r.k8sClient.CtrlClient().Get(
		ctx,
		types.NamespacedName{Name: entryCR.Name, Namespace: entryCR.Namespace},
		&currentCR,
	)
	if err != nil {
		return microerror.Mask(err)
	}

	entryCR.ResourceVersion = currentCR.ResourceVersion
	err = r.k8sClient.CtrlClient().Update(ctx, entryCR)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "updated appcatalogentry CR %#q in namespace %#q", entryCR.Name, entryCR.Namespace)

	return nil
}

func (r *Resource) getDesiredAppCatalogEntryCR(ctx context.Context, cr *v1alpha1.Catalog, e entry, isLatest bool) (*v1alpha1.AppCatalogEntry, error) {
	var err error
	name := key.AppCatalogEntryName(cr.Name, e.Name, e.Version)

	var rawMetadata []byte
	{
		if url, ok := e.Annotations[annotation.AppMetadata]; ok {
			rawMetadata, err = r.getMetadata(ctx, url)
			if err != nil {
				r.logger.LogCtx(ctx, "level", "info", "message", fmt.Sprintf("failed to get appMetadata for entry %#q in catalog %#q", e.Name, cr.Name), "stack", fmt.Sprintf("%#v", err))
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

	entryCR := &v1alpha1.AppCatalogEntry{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       kindAppCatalogEntry,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   cr.GetNamespace(),
			Annotations: e.Annotations,
			Labels: map[string]string{
				label.AppKubernetesName:    e.Name,
				label.AppKubernetesVersion: e.Version,
				label.CatalogName:          cr.Name,
				label.CatalogType:          key.CatalogType(*cr),
				pkglabel.Latest:            strconv.FormatBool(isLatest),
				label.ManagedBy:            key.AppCatalogEntryManagedBy(project.Name()),
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         apiVersion,
					BlockOwnerDeletion: to.BoolP(true),
					Kind:               kindCatalog,
					Name:               cr.Name,
					UID:                cr.UID,
				},
			},
		},
		Spec: v1alpha1.AppCatalogEntrySpec{
			AppName:    e.Name,
			AppVersion: e.AppVersion,
			Catalog: v1alpha1.AppCatalogEntrySpecCatalog{
				Name:      cr.Name,
				Namespace: cr.Namespace,
			},
			Chart: v1alpha1.AppCatalogEntrySpecChart{
				Description: e.Description,
				Home:        e.Home,
				Icon:        e.Icon,
				Keywords:    e.Keywords,
			},
			Version: e.Version,
		},
	}

	if m != nil {
		entryCR.Annotations = m.Annotations
		entryCR.Spec.Chart.APIVersion = m.ChartAPIVersion
		entryCR.Spec.Chart.UpstreamChartVersion = m.UpstreamChartVersion
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

	return entryCR, nil
}

// getLatestEntry returns the entry with the highest version without considering the creation date.
func (r *Resource) getLatestEntry(ctx context.Context, entries []entry) (entry, error) {
	var latestIndex int
	var latestVersion semver.Version
	var latestCreated metav1.Time

	for i := 0; i < len(entries); i++ {
		v, err := semver.NewVersion(entries[i].Version)
		if errors.As(err, &semver.ErrInvalidSemVer) {
			r.logger.Debugf(ctx, "invalid semver from converting app entry %s, version is %s", entries[i].Name, entries[i].Version)
			continue
		} else if err != nil {
			return entry{}, microerror.Mask(err)
		}

		// Removing Prerelease from version since they are mostly SHA strings which we cannot compare.
		nextVersion, err := v.SetPrerelease("")
		if err != nil {
			return entry{}, microerror.Mask(err)
		}

		if nextVersion.GreaterThan(&latestVersion) {
			latestIndex = i
			latestVersion = nextVersion
			latestCreated = entries[i].Created
			continue
		}

		if nextVersion.Equal(&latestVersion) {
			if entries[i].Created.Time.After(latestCreated.Time) {
				latestIndex = i
				latestVersion = nextVersion
				latestCreated = entries[i].Created
			}
		}
	}

	return entries[latestIndex], nil
}

func (r *Resource) newAppCatalogEntries(ctx context.Context, cr v1alpha1.Catalog, index index) (map[string]*v1alpha1.AppCatalogEntry, error) {
	entryCRs := map[string]*v1alpha1.AppCatalogEntry{}

	for _, entries := range index.Entries {
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Created.After(entries[j].Created.Time)
		})

		maxEntries := r.maxEntriesPerApp
		if len(entries) < maxEntries {
			maxEntries = len(entries)
		}

		var latestEntryCR *v1alpha1.AppCatalogEntry
		{
			latestEntry, err := r.getLatestEntry(ctx, entries)
			if err != nil {
				return nil, microerror.Mask(err)
			}

			latestEntryCR, err = r.getDesiredAppCatalogEntryCR(ctx, &cr, latestEntry, true)
			if err != nil {
				return nil, microerror.Mask(err)
			}
		}

		for i := 0; i < maxEntries; i++ {
			e := entries[i]

			entryCR, err := r.getDesiredAppCatalogEntryCR(ctx, &cr, e, latestEntryCR.Spec.Version == e.Version)
			if err != nil {
				return nil, microerror.Mask(err)
			}

			entryCRs[entryCR.Name] = entryCR
		}

		// If the latest entry is not included in the desired CRs, we add it so users can always see the latest CR.
		_, ok := entryCRs[latestEntryCR.Name]
		if !ok {
			entryCRs[latestEntryCR.Name] = latestEntryCR
		}
	}

	return entryCRs, nil
}
