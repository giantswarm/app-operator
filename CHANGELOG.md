# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project's packages adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [6.2.0] - 2022-07-11

### Added

- If no userconfig configmap or secret reference is specified but one is found following the default naming convention (`*-user-values` / `*-user-secrets`) then the App resource is updated to reference the found configmap/secret.
- Watch config maps and secrets listed in the `extraConfigs` section of App CR for multi layer configs, see: https://github.com/giantswarm/rfc/tree/main/multi-layer-app-config#enhancing-app-cr

### Changed

- Bump `github.com/giantswarm/app` to `v6.12.0`

## [6.1.0] - 2022-07-05

### Changed

- Use downward API to set deployment env var `KUBERNETES_SERVICE_HOST` to `status.hostIP`.
- Change `initialBootstrapMode` configuration value to `bootstrapMode`.
- Tighten pod and container security contexts for PSS restricted policies.

### Added

- Allow to set api server pod port when enabling `initialBootstrapMode`.

## [6.0.1] - 2022-06-20

### Added

- Add support for Catalogs that define multiple repository mirrors to be used in case some of them are unreachable.

### Changed

- Only run `PodMonitor` outside of bootstrap mode.

## [6.0.0] - 2022-06-08

### Added

- Added `PodMonitor` to the Helm chart to collect metrics from the running operator pod (instead of via the Service)

### Changed

- This version requires `prometheus-meta-operator` of `v3.6.0` or later to scrape the metrics from the `PodMinitor`
- This version requires `kyverno-policies-observability` of `v0.1.2` or later to have proper labels applied to metrics

### Removed

- Removed Service from the Helm chart

## [5.12.0] - 2022-06-06

### Added

- Add `initialBootstrapMode` flag to allow deploying CNI as managed apps.

## [5.11.0] - 2022-05-23

### Changed

- Only set resource limits on the deployment when the VPA is not available or disabled
- Increase min / max resource limits on VPA

## [5.10.2] - 2022-05-18

### Fixed

- Add missing permissions for `apps/deployments`.

## [5.10.1] - 2022-05-18

### Fixed

- Limit `*-chart` `ClusterRole` and `ClusterRoleBinding` to `giantswarm` namespace deployment.

## [5.10.0] - 2022-05-16

### Fixed

- Fix `app-operator` RBAC to avoid granting excessive permissions to its `ServiceAccount`.

### Removed

- Remove `authtokenmigration` resource.

## [5.9.0] - 2022-04-07

### Changed

- Update `helmclient` to v4.10.0.
- Update giantswarm/appcatalog to `v0.7.0`, adding support for internal OCI chart catalogs.


## [5.8.0] - 2022-03-11

### Added

- Add support for relative URLs in catalog indexes.

### Fixed

- Continue processing `AppCatalogEntry` CRs if an error occurs.
- Only show `AppCatalogEntry` CRs that are compatible with the current provider.
- For internal catalogs generate tarball URLs instead of checking `index.yaml`
to prevent chicken egg problems in new clusters.

## [5.7.5] - 2022-03-01

### Fixed

- Fix label selector in app values watcher so it supports CAPI clusters.
- Strip cluster name from App CR name to determine Chart CR name in `chart/current.go` resource to fix WC app updates.

## [5.7.4] - 2022-03-01

### Fixed

- Allow usage of chart-operator PSP so it can be bootstrapped.

## [5.7.3] - 2022-02-28

### Fixed

- Fixing patch to not reset fields.

## [5.7.2] - 2022-02-25

### Fixed

- Remove compatible providers validation for `AppCatalogEntry` as its overly strict.
- Push image to Docker Hub to not rely on crsync.

## [5.7.1] - 2022-02-22

### Fixed

- Restrict PSP usage to only named resource.

## [5.7.0] - 2022-02-17

## Added

- Annotate App CRs after bootstrapping chart-operator to trigger reconciliation.

## [5.6.0] - 2022-02-02

### Changed

- Get tarball URL for chart CRs from index.yaml for better community app catalog support.

### Fixed

- Fix error handling in chart CR watcher when chart CRD not installed.

## [5.5.2] - 2022-01-28

### Fixed

- Fix getting kubeconfig in chart CR watcher.

## [5.5.1] - 2022-01-20

### Fixed

- When bootstrapping chart-operator the helm release should not include the cluster ID.

## [5.5.0] - 2022-01-19

### Added

- Support watching app CRs in organization namespace with cluster label selector.

## [5.4.1] - 2022-01-14

### Fixed

- Embed Chart CRD in app-operator to prevent hitting GitHub API rate limits.

## [5.4.0] - 2021-12-17

### Changed

- Update Helm to `v3.6.3`.
- Use controller-runtime client to remove CAPI dependency.
- Use `apptestctl` to install CRDs in integration tests to avoid hitting GitHub rate limits.

### Removed

- Remove `releasemigration` resource now migration to Helm 3 is complete.

## [5.3.1] - 2021-12-08

### Added

- Support for App CRs with a `v` prefixed version. This enables Flux to automatically update the version based on its image tag.

## [5.3.0] - 2021-11-11

### Changed

- Use dynamic client instead of generated client for watching chart CRs.
- Validate `.spec.kubeConfig.secret.name` in validation resource.

## [5.2.0] - 2021-08-19

### Changed

- Reject App CRs with version labels with the legacy `1.0.0` value.
- Validate `.spec.catalog` using Catalog CRs instead of AppCatalog CRs.

## [5.1.1] - 2021-08-05

### Fixed

- Fix creating `AppCatalog` CRs in appcatalogsync resource.

## [5.1.0] - 2021-07-29

### Changed

- Create `AppCatalogEntry` CRs into the same namespace of Catalog CR.
- Include `chart.keywords`, `chart.description` and `chart.upstreamChartVersion` in `AppCatalogEntry` CRs.

## [5.0.0] - 2021-07-16

### Changed

- Create `AppCatalog` CRs from `Catalog` CRs for compatibility with existing app-operator releases.
- Prepare helm values to configuration management.
- Use `Catalog` CRs in `App` controller.
- Reconcile to `Catalog` CRs instead of `AppCatalog`.
- Get `Chart` CRD from the GitHub resources.
- Get metadata constants from k8smetadata library not apiextensions.

### Fixed

- For the chart CR watcher get the kubeconfig secret from the chart-operator app
CR to avoid hardcoding it.
- Quote namespace in helm templates to handle numeric workload cluster IDs.

## [4.4.0] - 2021-05-03

### Added

- Add support for skip CRD flag when installing Helm releases.
- Emit events when config maps and secrets referenced in App CRs are updated.

## [4.3.2] - 2021-04-06

### Fixed

- Updated OperatorKit to v4.3.1 for Kubernetes 1.20 support.

## [4.3.1] - 2021-03-30

### Fixed

- Restore chart-operator when it had been deleted.

## [4.3.0] - 2021-03-26

### Added

- Cache k8sclient, helmclient for later use.

### Changed

- Updated Helm to v3.5.3.

## [4.2.0] - 2021-03-19

### Added

- Apply the namespaceConfig to the desired chart.

## [4.1.0] - 2021-03-17

### Added

- Install apps in CAPI Workload Clusters.

## [4.0.2] - 2021-03-09

### Added

- Apply `compatibleProvider`,`namespace` metadata validation based on the relevant `AppCatalogEntry` CR.

## [4.0.1] - 2021-03-05

### Fixed

- Use backoff in chart CR watcher to wait until kubeconfig secret exists.

## [4.0.0] - 2021-02-23

### Added

- Add annotations from Helm charts to AppCatalogEntry CRs.
- Enable Vertical Pod Autoscaler.

### Changed

- Replace status webhook with chart CR status watcher.
- Sort AppCatalogEntry CRs by version and created timestamp.
- Watch cluster namespace for per workload cluster instances of app-operator.

## [3.2.0] - 2021-02-08

### Added

- Include `apiVersion`, `restrictions.compatibleProviders` in appcatalogentry CRs.

### Changed

- Limit the number of AppCatalogEntry per app.
- Delete legacy finalizers on app CRs.
- Reconciling appCatalog CRs only if pod is unique.

### Fixed

- Updating status as cordoned if app CR has cordoned annotation.

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

[Unreleased]: https://github.com/giantswarm/app-operator/compare/v6.2.0...HEAD
[6.2.0]: https://github.com/giantswarm/app-operator/compare/v6.1.0...v6.2.0
[6.1.0]: https://github.com/giantswarm/app-operator/compare/v6.0.1...v6.1.0
[6.0.1]: https://github.com/giantswarm/app-operator/compare/v6.0.0...v6.0.1
[6.0.0]: https://github.com/giantswarm/app-operator/compare/v5.12.0...v6.0.0
[5.12.0]: https://github.com/giantswarm/app-operator/compare/v5.11.0...v5.12.0
[5.11.0]: https://github.com/giantswarm/app-operator/compare/v5.10.2...v5.11.0
[5.10.2]: https://github.com/giantswarm/app-operator/compare/v5.10.1...v5.10.2
[5.10.1]: https://github.com/giantswarm/app-operator/compare/v5.10.0...v5.10.1
[5.10.0]: https://github.com/giantswarm/app-operator/compare/v5.9.0...v5.10.0
[5.9.0]: https://github.com/giantswarm/app-operator/compare/v5.8.0...v5.9.0
[5.8.0]: https://github.com/giantswarm/app-operator/compare/v5.7.5...v5.8.0
[5.7.5]: https://github.com/giantswarm/app-operator/compare/v5.7.4...v5.7.5
[5.7.4]: https://github.com/giantswarm/app-operator/compare/v5.7.3...v5.7.4
[5.7.3]: https://github.com/giantswarm/app-operator/compare/v5.7.2...v5.7.3
[5.7.2]: https://github.com/giantswarm/app-operator/compare/v5.7.1...v5.7.2
[5.7.1]: https://github.com/giantswarm/app-operator/compare/v5.7.0...v5.7.1
[5.7.0]: https://github.com/giantswarm/app-operator/compare/v5.6.0...v5.7.0
[5.6.0]: https://github.com/giantswarm/app-operator/compare/v5.5.2...v5.6.0
[5.5.2]: https://github.com/giantswarm/app-operator/compare/v5.5.1...v5.5.2
[5.5.1]: https://github.com/giantswarm/app-operator/compare/v5.5.0...v5.5.1
[5.5.0]: https://github.com/giantswarm/app-operator/compare/v5.4.1...v5.5.0
[5.4.1]: https://github.com/giantswarm/app-operator/compare/v5.4.0...v5.4.1
[5.4.0]: https://github.com/giantswarm/app-operator/compare/v5.3.1...v5.4.0
[5.3.1]: https://github.com/giantswarm/app-operator/compare/v5.3.0...v5.3.1
[5.3.0]: https://github.com/giantswarm/app-operator/compare/v5.2.0...v5.3.0
[5.2.0]: https://github.com/giantswarm/app-operator/compare/v5.1.1...v5.2.0
[5.1.1]: https://github.com/giantswarm/app-operator/compare/v5.1.0...v5.1.1
[5.1.0]: https://github.com/giantswarm/app-operator/compare/v5.0.0...v5.1.0
[5.0.0]: https://github.com/giantswarm/app-operator/compare/v4.4.0...v5.0.0
[4.4.0]: https://github.com/giantswarm/app-operator/compare/v4.3.2...v4.4.0
[4.3.2]: https://github.com/giantswarm/app-operator/compare/v4.3.1...v4.3.2
[4.3.1]: https://github.com/giantswarm/app-operator/compare/v4.3.0...v4.3.1
[4.3.0]: https://github.com/giantswarm/app-operator/compare/v4.2.0...v4.3.0
[4.2.0]: https://github.com/giantswarm/app-operator/compare/v4.1.0...v4.2.0
[4.1.0]: https://github.com/giantswarm/app-operator/compare/v4.0.2...v4.1.0
[4.0.2]: https://github.com/giantswarm/app-operator/compare/v4.0.1...v4.0.2
[4.0.1]: https://github.com/giantswarm/app-operator/compare/v4.0.0...v4.0.1
[4.0.0]: https://github.com/giantswarm/app-operator/compare/v3.2.0...v4.0.0
[3.2.0]: https://github.com/giantswarm/app-operator/compare/v3.1.0...v3.2.0
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
