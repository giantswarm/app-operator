# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project's packages adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v1.1.5] 2020-06-16

### Changed

- Cancel the reconciliation when failed to merge configMaps/secrets. 

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

[Unreleased]: https://github.com/giantswarm/app-operator/compare/v1.1.5..HEAD

[v1.1.5]: https://github.com/giantswarm/app-operator/compare/v1.1.4..v1.1.5
[v1.1.4]: https://github.com/giantswarm/app-operator/compare/v1.1.3..v1.1.4
[v1.1.3]: https://github.com/giantswarm/app-operator/compare/v1.1.2..v1.1.3
[v1.1.2]: https://github.com/giantswarm/app-operator/compare/v1.1.1..v1.1.2
[v1.1.1]: https://github.com/giantswarm/app-operator/compare/v1.1.0..v1.1.1
[v1.1.0]: https://github.com/giantswarm/app-operator/compare/v1.0.3..v1.1.0
[v1.0.3]: https://github.com/giantswarm/app-operator/compare/v1.0.2..v1.0.3
[v1.0.2]: https://github.com/giantswarm/app-operator/compare/v1.0.1..v1.0.2
[v1.0.1]: https://github.com/giantswarm/app-operator/releases/tag/v1.0.1
