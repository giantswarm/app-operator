package service

import (
	"github.com/giantswarm/versionbundle"

	"github.com/giantswarm/app-operator/service/controller/app/v1"
)

func NewVersionBundles() []versionbundle.Bundle {
	var versionBundles []versionbundle.Bundle

	versionBundles = append(versionBundles, v1.VersionBundle())

	return versionBundles
}
