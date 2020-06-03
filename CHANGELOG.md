# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project's packages adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## Unreleased

## [v1.0.5] 2020-06-03

### Changed

- Reconcile app CRs with both versions 1.0.0 and 0.0.0. This will be removed
later.

## [v1.0.4] 2020-06-01

### Changed

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

[Unreleased]: https://github.com/giantswarm/app-operator/compare/v1.0.5..HEAD

[1.0.5]: https://github.com/giantswarm/app-operator/compare/v1.0.4..v1.0.5
[1.0.4]: https://github.com/giantswarm/app-operator/compare/v1.0.3..v1.0.4
[1.0.3]: https://github.com/giantswarm/app-operator/compare/v1.0.2..v1.0.3
[1.0.2]: https://github.com/giantswarm/app-operator/compare/v1.0.1..v1.0.2
[1.0.1]: https://github.com/giantswarm/app-operator/releases/tag/v1.0.1
