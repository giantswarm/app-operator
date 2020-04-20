package service

import (
	"github.com/giantswarm/app-operator/service/controller/app"
	"github.com/giantswarm/app-operator/service/controller/appcatalog"
	"github.com/giantswarm/versionbundle"
)

func NewVersionBundles() []versionbundle.Bundle {
	var versionBundles []versionbundle.Bundle

	versionBundles = append(versionBundles, app.VersionBundle())
	versionBundles = append(versionBundles, appcatalog.VersionBundle())

	return versionBundles
}
