package helmrepository

import (
	"context"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/app/v7/pkg/key"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorkey "github.com/giantswarm/app-operator/v6/pkg/key"
)

// EnsureDeleted ensures HelmRepository CRs, the Catalog CR has created, are gone.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCatalog(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	// Delete Flux representation of repositories
	err = r.deleteAllHelmRepositories(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) deleteAllHelmRepositories(ctx context.Context, catalog v1alpha1.Catalog) error {
	// Start with a template HelmRepository CR. The name is obviously
	// wrong at this point and needs adjustment, which is going to be done
	// further down the stream.
	helmRepository := sourcev1.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      catalog.Name,
			Namespace: catalog.Namespace,
		},
	}

	r.logger.Debugf(
		ctx,
		"Deleting HelmRepository CR created from %#q Catalog CR storage in %#q namespace",
		catalog.Name,
		catalog.Namespace,
	)

	// Make sure HelmRepository CR configured by catalog storage gets
	// deleted.
	err := r.deleteHelmRepoitory(ctx, helmRepository, catalog.Spec.Storage)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(
		ctx,
		"Deleting HelmRepository CRs created from %#q Catalog CR repositories list in %#q namespace",
		catalog.Name,
		catalog.Namespace,
	)

	// Make sure HelmRepository CRs configured by catalog repository
	// list get deleted.
	for _, repository := range catalog.Spec.Repositories {
		err = r.deleteHelmRepoitory(ctx, helmRepository, repository)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	if catalog.Status.HelmRepositoryList == nil {
		return nil
	}

	r.logger.Debugf(
		ctx,
		"Deleting HelmRepository CRs from %#q Catalog CR status list",
		catalog.Name,
	)

	// If catalog status contains a non-empty list of HelmRepository CRs
	// make sure all referenced resources are gone from cluster.
	for _, entry := range catalog.Status.HelmRepositoryList.Entries {
		helmRepository.Name = entry.Name
		helmRepository.Namespace = entry.Namespace

		err = r.delete(ctx, &helmRepository)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (r *Resource) deleteHelmRepoitory(ctx context.Context, helmRepository sourcev1.HelmRepository, repository interface{}) error {
	hrType, hrURL, err := operatorkey.GetRepositoryConfiguration(repository)
	if err != nil {
		return microerror.Mask(err)
	}

	hrName, err := operatorkey.GetHelmRepositoryName(helmRepository.Name, hrType, hrURL)
	if err != nil {
		return microerror.Mask(err)
	}

	helmRepository.Name = hrName

	err = r.delete(ctx, &helmRepository)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
