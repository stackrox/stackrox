# Changelog
All notable changes to this project that require documentation updates will be documented in this file.


## [23.0]
### Added
- Installer prompt to configure the size of the external volume for central.
### Changed
- Prometheus endpoint changed from https://localhost:8443 to http://localhost:9090.
- Scanner is now given certificates, and Central<->Scanner communication secured via mTLS.

## [22.0]
### Changed
- Default size of central's PV changed from 10Gi to 100Gi.