package app

import (
	"github.com/giantswarm/versionbundle"
)

func VersionBundle() versionbundle.Bundle {
	return versionbundle.Bundle{
		Changelogs: []versionbundle.Changelog{
			{
				Component:   "app-operator",
				Description: "Added initial version.",
				Kind:        versionbundle.KindAdded,
			},
		},
		Components: []versionbundle.Component{},
		Name:       "app-operator",
		Version:    "1.0.0",
	}
}
