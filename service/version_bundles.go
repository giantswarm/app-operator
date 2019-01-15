package service

import (
	"github.com/giantswarm/versionbundle"

	appv1 "github.com/giantswarm/app-operator/service/controller/app/v1"
	appcatalogv1 "github.com/giantswarm/app-operator/service/controller/appcatalog/v1"
)

func NewVersionBundles() []versionbundle.Bundle {
	var versionBundles []versionbundle.Bundle

	versionBundles = append(versionBundles, appv1.VersionBundle())
	versionBundles = append(versionBundles, appcatalogv1.VersionBundle())

	return versionBundles
}
