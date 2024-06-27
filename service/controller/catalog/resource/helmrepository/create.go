package helmrepository

import (
	"context"
	"reflect"
	"time"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v7/pkg/key"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	operatorkey "github.com/giantswarm/app-operator/v6/pkg/key"
)

// EnsureCreated ensures Catalog CRs are translated into HelmRepository CRs.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCatalog(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	// Create Flux representation of desired list of repositories
	// the Catalog CR provides.
	desiredState, err := r.createOrUpdateHelmRepositories(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	// Delete Flux representation of repositories that were known to
	// be on the list, but are now gone.
	err = r.deleteGoneHelmRepositories(ctx, desiredState, cr.Status.HelmRepositoryList)
	if err != nil {
		return microerror.Mask(err)
	}

	// Update Catalog CR status with currently known list of Flux HelmRepository CRs
	// representing repositories of this CR.
	err = r.updateCatalogStatus(ctx, cr, desiredState)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// createOrUpdateHelmRepositories triggers a creation or an update of the HelmRepositories.
// It takes the Catalog CR's .storage and .repositories fields as input, which are decisive
// about the desired state, and it turns them into HelmRepository CRs. It then returns this
// desired state for it to be compared against the current state kept in the status field.
func (r *Resource) createOrUpdateHelmRepositories(ctx context.Context, catalog v1alpha1.Catalog) (map[string]empty, error) {
	desiredState := make(map[string]empty)

	// The template HelmRepository CR. The namespace is common to all CRs created out of
	// this catalog, hence it is configured here, and the name is to be re-adjusted
	// in the downstream methods with the catalog name is used as a baseline.
	helmRepository := sourcev1.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      catalog.Name,
			Namespace: catalog.Namespace,
		},
	}

	r.logger.Debugf(
		ctx,
		"Translating %#q Catalog CR in %#q namespace storage into HelmRepository CR",
		catalog.Name,
		catalog.Namespace,
	)

	// create or update HelmRepository CR from the Catalog CR storage,
	// and on success update desired state with a name of the resource.
	name, err := r.createOrUpdateHelmRepository(ctx, helmRepository, catalog.Spec.Storage)
	if err != nil {
		return map[string]empty{}, microerror.Mask(err)
	}
	desiredState[name] = empty{}

	r.logger.Debugf(
		ctx,
		"Translating %#q Catalog CR in %#q namespace repositories list into HelmRepository CRs",
		catalog.Name,
		catalog.Namespace,
	)
	for _, repository := range catalog.Spec.Repositories {
		// create or update HelmRepository CR from the Catalog CR repositories,
		// and on success update desired state with a name of the resource.
		name, err = r.createOrUpdateHelmRepository(ctx, helmRepository, repository)
		if err != nil {
			return map[string]empty{}, microerror.Mask(err)
		}
		desiredState[name] = empty{}
	}

	return desiredState, nil
}

// createOrUpdateHelmRepository deals with a single repository at a time. It makes sure HelmRepository CR
// gets a unique name, configures it, and then makes sure it is either created or updated accordingly.
func (r *Resource) createOrUpdateHelmRepository(ctx context.Context, helmRepository sourcev1.HelmRepository, repository interface{}) (string, error) {
	hrType, hrURL, err := operatorkey.GetRepositoryConfiguration(repository)
	if err != nil {
		return "", microerror.Mask(err)
	}

	hrName, err := operatorkey.GetHelmRepositoryName(helmRepository.Name, hrType, hrURL)
	if err != nil {
		return "", microerror.Mask(err)
	}

	helmRepository.Name = hrName
	helmRepository.Spec = sourcev1.HelmRepositorySpec{
		Interval: metav1.Duration{Duration: 10 * time.Minute},
		Provider: "generic",
		Timeout:  &metav1.Duration{Duration: 1 * time.Minute},
		URL:      hrURL,
	}

	if hrType == "oci" {
		helmRepository.Spec.Type = hrType
	}

	return hrName, r.createOrUpdate(ctx, &helmRepository)
}

// createOrUpdate deals with the actual creation or updates of a HelmRepository
// custom resource.
func (r *Resource) createOrUpdate(ctx context.Context, helmRepository *sourcev1.HelmRepository) error {
	existingHelmRepository := &sourcev1.HelmRepository{}
	err := r.ctrlClient.Get(
		ctx,
		types.NamespacedName{Name: helmRepository.Name, Namespace: helmRepository.Namespace},
		existingHelmRepository,
	)

	// CR is not found hence it must be created
	if apierrors.IsNotFound(err) {
		r.logger.Debugf(
			ctx,
			"%#q HelmRepository CR in %#q namespace not found, creating it",
			helmRepository.Name,
			helmRepository.Namespace,
		)
		return r.create(ctx, helmRepository)
	}

	if err != nil {
		return microerror.Mask(err)
	}

	// return early when CR exists and has not been changed
	if !needsUpdate(helmRepository, existingHelmRepository) {
		r.logger.Debugf(
			ctx,
			"%#q HelmRepository CR in %#q namespace up to date",
			helmRepository.Name,
			helmRepository.Namespace,
		)
		return nil
	}

	r.logger.Debugf(
		ctx,
		"%#q HelmRepository CR in %#q namespace not in desired state, updating it",
		helmRepository.Name,
		helmRepository.Namespace,
	)

	// otherwise update the CR
	helmRepository.ResourceVersion = existingHelmRepository.ResourceVersion
	return r.update(ctx, helmRepository)
}

// deleteGoneHelmRepositories deals with HelmRepository CR deletion
func (r *Resource) deleteGoneHelmRepositories(ctx context.Context, desired map[string]empty, current *v1alpha1.HelmRepositoryList) error {
	// current list may be empty if we process given
	// object for the first time.
	if current == nil {
		return nil
	}

	for _, chr := range current.Entries {
		// if status entry is still on the desired list then we obviously
		// skip its deletion.
		if _, ok := desired[chr.Name]; ok {
			continue
		}

		r.logger.Debugf(
			ctx,
			"%#q HelmRepository CR in %#q namespace no longer configured in Catalog CR, removing it",
			chr.Name,
			chr.Namespace,
		)

		helmRepository := &sourcev1.HelmRepository{
			ObjectMeta: metav1.ObjectMeta{
				Name:      chr.Name,
				Namespace: chr.Namespace,
			},
		}

		err := r.delete(ctx, helmRepository)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (r *Resource) create(ctx context.Context, desired *sourcev1.HelmRepository) error {
	err := r.ctrlClient.Create(ctx, desired)
	if apierrors.IsAlreadyExists(err) {
		// skip as a fail safe
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) delete(ctx context.Context, desired *sourcev1.HelmRepository) error {
	err := r.ctrlClient.Delete(ctx, desired)
	if apierrors.IsNotFound(err) {
		// skip as a fail safe
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) update(ctx context.Context, desired *sourcev1.HelmRepository) error {
	err := r.ctrlClient.Update(ctx, desired)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// updateCatalogStatus updates Catalog CR status to offer a relation between it and
// HelmRepository CRs created out of it.
func (r *Resource) updateCatalogStatus(ctx context.Context, catalog v1alpha1.Catalog, desired map[string]empty) error {
	entries := make([]v1alpha1.HelmRepositoryRef, 0)

	for name := range desired {
		entries = append(entries, v1alpha1.HelmRepositoryRef{
			Name:      name,
			Namespace: catalog.Namespace,
		})
	}

	newStatus := v1alpha1.CatalogStatus{
		HelmRepositoryList: &v1alpha1.HelmRepositoryList{
			Entries: entries,
		},
	}

	// We may consider getting the object again for its resource version could have
	// changed, but if it has changed then updating the status might be pointless anyway
	// for it may not be the most up to date one.
	catalog.Status = newStatus
	err := r.ctrlClient.Status().Update(ctx, &catalog)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func needsUpdate(desired, current *sourcev1.HelmRepository) bool {
	if !reflect.DeepEqual(desired.Spec, current.Spec) {
		return true
	}
	if !reflect.DeepEqual(desired.Annotations, current.Annotations) {
		return true
	}
	if !reflect.DeepEqual(desired.Labels, current.Labels) {
		return true
	}

	return false
}
