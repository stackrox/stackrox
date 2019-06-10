# Changelog
All notable changes to this project that require documentation updates will be documented in this file.


## [23.0]
### Added
- Installer prompt to configure the size of the external volume for central.
### Changed
- Prometheus endpoint changed from https://localhost:8443 to http://localhost:9090.
- Scanner is now given certificates, and Central<->Scanner communication secured via mTLS.
- Central CPU Request changed from 1 core to 1.5 cores
- Central Memory Request changed from 2Gi to 4Gi
- Sensor CPU Request changed from .2 cores to .5 cores
- Sensor Memory Request changes from 250Mi to 500Mi
- Sensor CPU Limit changed from .5 cores to 1 core

## [22.0]
### Changed
- Default size of central's PV changed from 10Gi to 100Gi.
