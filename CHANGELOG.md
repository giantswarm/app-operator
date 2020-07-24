# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project's packages adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.9] - 2020-07-24

### Changed

- Graduate application group CRDs to v1.
- Upgrade to operatorkit 1.2.0.

### Fixed

- Fix API group for PSPs

## [v1.1.8] 2020-07-01

- Extend to 20 minutes for waiting helm 3 migration completed.
- RBAC added to deletion migration resources.

## [v1.1.7] 2020-06-30

### Changed

- Delete migration app after checking the release.

## [v1.1.6] 2020-06-29

### Changed

- Delete helm-2to3-migration job after migration is finished.
- Sending metrics with app CR's version in the spec.
- Only emit metrics for app CRs reconciled by this instance of the operator.
- Expose App's `.spec.catalog` field as a collected metric

## [v1.1.5] 2020-06-16

### Changed

- Cancel the reconciliation when failed to merge configMaps/secrets.
- Fix problems with openapi valdidation rules for app and appcatalog CRDs.
- Make optional fields nullable for app and appcatalog CRDs.

## [v1.1.4] 2020-06-04

### Changed

- Check chart-operator deployment status before initiating helm 3 migration.

## [v1.1.3] 2020-05-26

### Changed

- Log app name in collector when cordon-until annotation cannot be parsed.
- Update to helmclient v1.0.1 for security patch.

## [v1.1.2] 2020-05-21

### Changed

- Fix problem setting image registry for migration job.
- Update dependencies including error handling for unavailable tenant clusters.

## [v1.1.1] 2020-05-21

### Changed

- Set HTTP client timeout for helmclient when pulling charts in China.

## [v1.1.0] 2020-05-18

### Changed

- Updated to use Helm 3 and add releasemigration resource for migrating releases
from Helm 2 to Helm 3.

## [v1.0.3] 2020-05-18

### Changed

- Cancel resources when app CRs are cordoned.

## [v1.0.2] 2020-05-08

### Added

- Add team label to app info metrics for routing alerts.

## [v1.0.1] 2020-04-23

### Added

- Flattening operator release structure.

[Unreleased]: https://github.com/giantswarm/app-operator/compare/v1.1.9...HEAD
[1.1.9]: https://github.com/giantswarm/app-operator/compare/v1.1.8...v1.1.9
[v1.1.8]: https://github.com/giantswarm/app-operator/compare/v1.1.7...v1.1.8
[v1.1.7]: https://github.com/giantswarm/app-operator/compare/v1.1.6...v1.1.7
[v1.1.6]: https://github.com/giantswarm/app-operator/compare/v1.1.5...v1.1.6
[v1.1.5]: https://github.com/giantswarm/app-operator/compare/v1.1.4...v1.1.5
[v1.1.4]: https://github.com/giantswarm/app-operator/compare/v1.1.3...v1.1.4
[v1.1.3]: https://github.com/giantswarm/app-operator/compare/v1.1.2...v1.1.3
[v1.1.2]: https://github.com/giantswarm/app-operator/compare/v1.1.1...v1.1.2
[v1.1.1]: https://github.com/giantswarm/app-operator/compare/v1.1.0...v1.1.1
[v1.1.0]: https://github.com/giantswarm/app-operator/compare/v1.0.3...v1.1.0
[v1.0.3]: https://github.com/giantswarm/app-operator/compare/v1.0.2...v1.0.3
[v1.0.2]: https://github.com/giantswarm/app-operator/compare/v1.0.1...v1.0.2
[v1.0.1]: https://github.com/giantswarm/app-operator/releases/tag/v1.0.1
