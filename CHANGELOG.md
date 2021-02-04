# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project's packages adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Include `apiVersion`, `restrictions.compatibleProviders` in appcatalogentry CRs.
  
### Changed
  
- Limit the number of AppCatalogEntry per app.
- Delete legacy finalizers on app CRs. 
- Reconciling appCatalog CRs only if pod is unique.

## [3.1.0] - 2021-01-13

## [3.0.0] - 2021-01-05

### Changed

- Enable mutating and validating webhooks in app-admission-controller for
tenant app CRs.

### Added

- Make resync period configurable for use in integration tests.
- Pause App CR reconciliation when it has
  `app-operator.giantswarm.io/paused=true` annotation.
- Print difference between the current chart and desired chart.

## [2.8.0] - 2020-12-15

### Changed

- Using values service from the app library.
- Updated Helm to v3.4.2.

### Added

- Add printer columns for Version, Last Deployed and Status to chart CRD in
tenant clusters.
- Use validation logic from the app library.
- Include restrictions data from app metadata files in appcatalogentry CRs.

### Fixed

- Reuse clients in clients resource when app CR uses inCluster.

## [2.7.0] - 2020-11-09

### Added

- Secure the webhook with token value from control plane catalog.

## [2.6.0] - 2020-10-29

### Added

- Adding webhook URL as annotation into chart CRs.
- Added Status update endpoint.

### Changed

- Update apiextensions to v3 and replace CAPI with Giant Swarm fork.

## [2.5.0] - 2020-10-27

### Added

- Watch secrets referenced in app CRs to reduce latency when applying config
changes.

## [2.4.1] - 2020-10-26

### Fixed

- Use resourceVersion of configmap for comparison instead of listing option.

## [2.4.0] - 2020-10-23

### Added

- Create appcatalogentry CRs for public app catalogs.
- Watch configmaps referenced in app CRs to reduce latency when applying config
changes.

## [2.3.5] - 2020-10-20

### Fixed

- Skip removing finalizer for chart-operator chart CR if its not present.

## [2.3.4] - 2020-10-16

### Fixed

- Skip deleting chart-operator in case of cluster deletion.

## [2.3.3] - 2020-10-15

### Added

- Delete chart-operator helm release and chart CR so it can be re-installed.

## [2.3.2] - 2020-09-29

### Fixed

- Updated Helm to v3.3.4.
- Updated Kubernetes dependencies to v1.18.9.
- Update deployment annotation to use checksum instead of helm revision to
reduce how often pods are rolled.

## [2.3.1] - 2020-09-22

### Added

- Added event count metrics for delete, install, rollback and update of Helm releases.

### Fixed

- Fix YAML comparison for chart configmaps and secrets.
- Fix structs merging error in helmclient.

### Security

- Updated Helm to v3.3.3.

## [2.3.0] - 2020-09-17

### Added

- Add resource version for chart configmaps and secrets to the chart CR to reduce latency of update events.

## [2.2.0] - 2020-09-07

### Added

- Add monitoring label
- Add validation resource that checks if references to other resources exist in
app CRs. A message is added to the app CR status for the user.

### Fixed

- Update the status when failing to merge configMaps or secrets on the initial reconciliation.
- Remove CPU and memory limits from deployment.

## [2.1.1] - 2020-08-26

### Changed

- Delete chart-operator release if it stuck in `pending-install` status.

### Removed

- Removed a collector from the operator.

## [2.1.0] - 2020-08-18

### Added

- Added chartcrd resource for creating chart CRD in tenant clusters.

### Changed

- Removed hardcoded version in app CR version label.
- Updated Helm to v3.3.0.

### Removed

- Don't wait for chart-operator pod since chart CRD is created by the chartcrd resource.

## [2.0.0] - 2020-08-13

### Changed

- Updated backward incompatible Kubernetes dependencies to v1.18.5.
- Updated Helm to v3.2.4.

## [1.1.11] - 2020-08-10

### Changed

- Updated app to team mappings for app alerts.

## [1.1.10] - 2020-08-04

### Added

- Add metrics for ready app-operator instances per app CR version.

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

[Unreleased]: https://github.com/giantswarm/app-operator/compare/v3.1.0...HEAD
[3.1.0]: https://github.com/giantswarm/app-operator/compare/v3.0.0...v3.1.0
[3.0.0]: https://github.com/giantswarm/app-operator/compare/v2.8.0...v3.0.0
[2.8.0]: https://github.com/giantswarm/app-operator/compare/v2.7.0...v2.8.0
[2.7.0]: https://github.com/giantswarm/app-operator/compare/v2.6.0...v2.7.0
[2.6.0]: https://github.com/giantswarm/app-operator/compare/v2.5.0...v2.6.0
[2.5.0]: https://github.com/giantswarm/app-operator/compare/v2.4.1...v2.5.0
[2.4.1]: https://github.com/giantswarm/app-operator/compare/v2.4.0...v2.4.1
[2.4.0]: https://github.com/giantswarm/app-operator/compare/v2.3.5...v2.4.0
[2.3.5]: https://github.com/giantswarm/app-operator/compare/v2.3.4...v2.3.5
[2.3.4]: https://github.com/giantswarm/app-operator/compare/v2.3.3...v2.3.4
[2.3.3]: https://github.com/giantswarm/app-operator/compare/v2.3.2...v2.3.3
[2.3.2]: https://github.com/giantswarm/app-operator/compare/v2.3.1...v2.3.2
[2.3.1]: https://github.com/giantswarm/app-operator/compare/v2.3.0...v2.3.1
[2.3.0]: https://github.com/giantswarm/app-operator/compare/v2.2.0...v2.3.0
[2.2.0]: https://github.com/giantswarm/app-operator/compare/v2.1.1...v2.2.0
[2.1.1]: https://github.com/giantswarm/app-operator/compare/v2.1.0...v2.1.1
[2.1.0]: https://github.com/giantswarm/app-operator/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/giantswarm/app-operator/compare/v1.1.11...v2.0.0
[1.1.11]: https://github.com/giantswarm/app-operator/compare/v1.1.10...v1.1.11
[1.1.10]: https://github.com/giantswarm/app-operator/compare/v1.1.9...v1.1.10
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
