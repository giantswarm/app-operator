# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project's packages adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## Unreleased

### Changed

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

[Unreleased]: https://github.com/giantswarm/app-operator/compare/v1.1.1..HEAD

[1.1.1]: https://github.com/giantswarm/app-operator/compare/v1.1.0..v1.1.1
[1.1.0]: https://github.com/giantswarm/app-operator/compare/v1.0.3..v1.1.0
[1.0.3]: https://github.com/giantswarm/app-operator/compare/v1.0.2..v1.0.3
[1.0.2]: https://github.com/giantswarm/app-operator/compare/v1.0.1..v1.0.2
[1.0.1]: https://github.com/giantswarm/app-operator/releases/tag/v1.0.1
