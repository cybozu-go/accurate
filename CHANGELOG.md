# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

## [0.3.0] - 2021-12-06

### Added

- Propagating ServiceAccount [#33](https://github.com/cybozu-go/accurate/pull/33)
- Set labels/annotations of sub-namespaces [#7](https://github.com/cybozu-go/accurate/pull/7)

### Changed

- Adjust QPS for querying kube-apiserver to 50 [#36](https://github.com/cybozu-go/accurate/pull/36)
- Support Kubernetes 1.22 [#38](https://github.com/cybozu-go/accurate/pull/38)

## [0.2.0] - 2021-10-26

### Added

- Add `kubectl-accurate template list` sub command [#9](https://github.com/cybozu-go/accurate/pull/9)
- Add Helm chart [#11](https://github.com/cybozu-go/accurate/pull/11)

### Changed

- Change LICENSE from MIT to Apache 2 [#5](https://github.com/cybozu-go/accurate/pull/5)
- Allow shell glob patterns for label/annotation keys [#8](https://github.com/cybozu-go/accurate/pull/8)
- Add ResourceQuota propagation to the default setting [#15](https://github.com/cybozu-go/accurate/pull/15)

### Fixed

- Add role to view all resources for propagate resource [#21](https://github.com/cybozu-go/accurate/pull/21)
- Import auth plugin in kubectl-accurate [#27](https://github.com/cybozu-go/accurate/pull/27)
- Fix infinite reconciliation on non-existent namespaces [#28](https://github.com/cybozu-go/accurate/pull/28)
- Do not delete non-propagated resources in template/root namespaces [#29](https://github.com/cybozu-go/accurate/pull/29)

## [0.1.0]

This is the first public release.

[Unreleased]: https://github.com/cybozu-go/accurate/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/cybozu-go/accurate/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/cybozu-go/accurate/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/cybozu-go/accurate/compare/4b825dc642cb6eb9a060e54bf8d69288fbee4904...v0.1.0
