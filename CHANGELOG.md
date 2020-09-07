# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project's packages adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.0] - 2020-09-07

### Added

- Add validation resource that checks if references to other resources exist in
app CRs. A message is added to the app CR status for the user.

### Fixed

- Update the status when failing to merge configMaps or secrets on the initial reconciliation.

## [1.0.9] - 2020-07-23

- Disable app, appcatalog CRDs creation.

## [v1.0.8] 2020-06-23

- Fix problems with openapi valdidation rules for app and appcatalog CRDs.
- Make optional fields nullable for app and appcatalog CRDs.
- Only emit metrics for app CRs reconciled by this instance of the operator.
- Expose App's `.spec.catalog` field as a collected metric

## [v1.0.7] 2020-06-11

- Cancel tiller resource if the chart-operator is already deployed in clusters.

## [v1.0.6] 2020-06-10

- Update status accordingly when parsing configMaps/secrets failed.

## [v1.0.5] 2020-06-05

- Using selector in controller instead of handleFunc in resourceset.

## [v1.0.4] 2020-06-01

- Reconcile different app CR versions for tenant and control planes.

## [v1.0.3] 2020-05-18

### Changed

- Cancel resources when app CRs are cordoned.

## [v1.0.2] 2020-05-08

### Added

- Add team label to app info metrics for routing alerts.

## [v1.0.1] 2020-04-23

### Added

- Flattening operator release structure.

[Unreleased]: https://github.com/giantswarm/app-operator/compare/v1.0.9...HEAD
[1.0.9]: https://github.com/giantswarm/app-operator/compare/v1.0.8...v1.0.9
[v1.0.8]: https://github.com/giantswarm/app-operator/compare/v1.0.7...v1.0.8
[v1.0.7]: https://github.com/giantswarm/app-operator/compare/v1.0.6...v1.0.7
[v1.0.6]: https://github.com/giantswarm/app-operator/compare/v1.0.5...v1.0.6
[v1.0.5]: https://github.com/giantswarm/app-operator/compare/v1.0.4...v1.0.5
[v1.0.4]: https://github.com/giantswarm/app-operator/compare/v1.0.3...v1.0.4
[v1.0.3]: https://github.com/giantswarm/app-operator/compare/v1.0.2...v1.0.3
[v1.0.2]: https://github.com/giantswarm/app-operator/compare/v1.0.1...v1.0.2
[v1.0.1]: https://github.com/giantswarm/app-operator/releases/tag/v1.0.1
