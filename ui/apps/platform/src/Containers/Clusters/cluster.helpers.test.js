import {
    findUpgradeState,
    formatSensorVersion,
    getCredentialExpirationStatus,
    getUpgradeableClusters,
} from './cluster.helpers';

describe('cluster helpers', () => {
    describe('formatSensorVersion', () => {
        it('should return sensor version string if passed a status object with a sensorVersion field', () => {
            const sensorVersion = 'sensorVersion';
            const testCluster = {
                status: {
                    sensorVersion,
                },
            };

            const displayValue = formatSensorVersion(testCluster.status?.sensorVersion);

            expect(displayValue).toEqual(sensorVersion);
        });

        it('should return a "Not Running" if passed a status object with null sensorVersion field', () => {
            const testCluster = {
                status: {
                    sensorVersion: null,
                },
            };

            const displayValue = formatSensorVersion(testCluster.status?.sensorVersion);

            expect(displayValue).toEqual('Not Running');
        });

        it('should return a "Not Running" if passed a status object with null status field', () => {
            const testCluster = {
                status: null,
            };

            const displayValue = formatSensorVersion(testCluster.status?.sensorVersion);

            expect(displayValue).toEqual('Not Running');
        });
    });

    describe('getUpgradeableClusters', () => {
        it('should return 0 when no clusters are unpgradeable', () => {
            const clusters = [
                {
                    id: 'f7ae6b5f-6329-4ed9-a439-83181991a526',
                    name: 'K8S',
                    status: {
                        upgradeStatus: {
                            upgradability: 'UP_TO_DATE',
                            mostRecentProcess: {
                                active: false,
                                progress: {
                                    upgradeState: 'UPGRADE_COMPLETE',
                                },
                            },
                        },
                    },
                },
                {
                    id: '26eac883-1f09-4123-971b-8b00ee63f5fd',
                    name: 'remote1',
                    status: {
                        upgradeStatus: {
                            upgradability: 'UP_TO_DATE',
                            mostRecentProcess: {
                                active: false,
                                progress: {
                                    upgradeState: 'UPGRADE_COMPLETE',
                                },
                            },
                        },
                    },
                },
            ];

            const upgradeableClusters = getUpgradeableClusters(clusters);

            expect(upgradeableClusters.length).toEqual(0);
        });

        it('should the number of ugradeable clusters', () => {
            const clusters = [
                {
                    id: 'f7ae6b5f-6329-4ed9-a439-83181991a526',
                    name: 'K8S',
                    status: {
                        upgradeStatus: {
                            upgradability: 'UP_TO_DATE',
                            mostRecentProcess: {
                                active: false,
                                progress: {
                                    upgradeState: 'UPGRADE_COMPLETE',
                                },
                            },
                        },
                    },
                },
                {
                    id: '26eac883-1f09-4123-971b-8b00ee63f5fd',
                    name: 'remote1',
                    status: {
                        upgradeStatus: {
                            upgradability: 'AUTO_UPGRADE_POSSIBLE',
                            mostRecentProcess: {
                                active: false,
                                progress: {
                                    upgradeState: 'UPGRADE_COMPLETE',
                                },
                            },
                        },
                    },
                },
            ];

            const upgradeableClusters = getUpgradeableClusters(clusters);

            expect(upgradeableClusters.length).toEqual(1);
        });
    });

    describe('findUpgradeState', () => {
        it('should return null if upgradeStatus is null', () => {
            const testUpgradeStatus = null;

            const received = findUpgradeState(testUpgradeStatus);

            expect(received).toEqual(null);
        });

        it('should return "Up to date with Central" if upgradeStatus -> upgradability is UP_TO_DATE', () => {
            const testUpgradeStatus = {
                upgradability: 'UP_TO_DATE',
            };

            const received = findUpgradeState(testUpgradeStatus);

            const expected = { displayValue: 'Up to date with Central', type: 'current' };
            expect(received).toEqual(expected);
        });

        it('should return "Manual upgrade required" if upgradeStatus -> upgradability is MANUAL_UPGRADE_REQUIRED', () => {
            const testUpgradeStatus = {
                upgradability: 'MANUAL_UPGRADE_REQUIRED',
            };

            const received = findUpgradeState(testUpgradeStatus);

            const expected = { displayValue: 'Manual upgrade required', type: 'intervention' };
            expect(received).toEqual(expected);
        });

        it('should return "Upgrade available" if there is no mostRecentProcess', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
            };

            const received = findUpgradeState(testUpgradeStatus);

            const expected = {
                type: 'download',
                actionText: 'Upgrade available',
            };
            expect(received).toEqual(expected);
        });

        it('should return "Upgrade available" if upgradeStatus -> upgradability is AUTO_UPGRADE_POSSIBLE but mostRecentProgress is not active and is COMPLETE', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: false,
                    progress: {
                        upgradeState: 'UPGRADE_COMPLETE',
                    },
                },
            };

            const received = findUpgradeState(testUpgradeStatus);

            const expected = {
                type: 'download',
                actionText: 'Upgrade available',
            };
            expect(received).toEqual(expected);
        });

        it('should print the error (and "Retry Upgrade") if upgradeStatus -> upgradability is AUTO_UPGRADE_POSSIBLE but mostRecentProgress is not active and failed', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: false,
                    progress: {
                        upgradeState: 'PRE_FLIGHT_CHECKS_FAILED',
                    },
                },
            };

            const received = findUpgradeState(testUpgradeStatus);

            const expected = {
                displayValue: 'Pre-flight checks failed',
                type: 'failure',
                actionText: 'Retry upgrade',
            };
            expect(received).toEqual(expected);
        });

        it('should return "Upgrade initializing" if upgradeStatus -> upgradability is AUTO_UPGRADE_POSSIBLE and upgradeState is UPGRADE_INITIALIZING', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'UPGRADE_INITIALIZING',
                    },
                },
            };

            const received = findUpgradeState(testUpgradeStatus);

            const expected = {
                displayValue: 'Upgrade initializing',
                type: 'progress',
            };
            expect(received).toEqual(expected);
        });

        it('should return "Upgrader launching" if upgradeState is UPGRADER_LAUNCHING', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'UPGRADER_LAUNCHING',
                    },
                },
            };

            const received = findUpgradeState(testUpgradeStatus);

            const expected = { displayValue: 'Upgrader launching', type: 'progress' };
            expect(received).toEqual(expected);
        });

        it('should return "Upgrader launched" if upgradeState is UPGRADER_LAUNCHED', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'UPGRADER_LAUNCHED',
                    },
                },
            };

            const received = findUpgradeState(testUpgradeStatus);

            const expected = { displayValue: 'Upgrader launched', type: 'progress' };
            expect(received).toEqual(expected);
        });

        it('should return "Pre-flight checks complete" if upgradeState is PRE_FLIGHT_CHECKS_COMPLETE', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'PRE_FLIGHT_CHECKS_COMPLETE',
                    },
                },
            };

            const received = findUpgradeState(testUpgradeStatus);

            const expected = { displayValue: 'Pre-flight checks complete', type: 'progress' };
            expect(received).toEqual(expected);
        });

        it('should return "Pre-flight checks failed." if upgradeState is PRE_FLIGHT_CHECKS_FAILED', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'PRE_FLIGHT_CHECKS_FAILED',
                    },
                },
            };

            const received = findUpgradeState(testUpgradeStatus);

            const expected = {
                displayValue: 'Pre-flight checks failed',
                type: 'failure',
                actionText: 'Retry upgrade',
            };
            expect(received).toEqual(expected);
        });

        it('should return "Upgrade Operations Done" if upgradeState is UPGRADE_OPERATIONS_DONE', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'UPGRADE_OPERATIONS_DONE',
                    },
                },
            };

            const received = findUpgradeState(testUpgradeStatus);

            const expected = { displayValue: 'Upgrade operations done', type: 'progress' };
            expect(received).toEqual(expected);
        });

        it('should return "Upgrade Operations Complete" if upgradeState is UPGRADE_COMPLETE', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'UPGRADE_COMPLETE',
                    },
                },
            };

            const received = findUpgradeState(testUpgradeStatus);

            const expected = { displayValue: 'Upgrade complete', type: 'current' };
            expect(received).toEqual(expected);
        });

        it('should return "Upgrade failed. Rolled back." if upgradeState is UPGRADE_ERROR_ROLLED_BACK', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'UPGRADE_ERROR_ROLLED_BACK',
                    },
                },
            };

            const received = findUpgradeState(testUpgradeStatus);

            const expected = {
                displayValue: 'Upgrade failed. Rolled back.',
                type: 'failure',
                actionText: 'Retry upgrade',
            };
            expect(received).toEqual(expected);
        });

        it('should return "Upgrade failed. Rollback failed." if upgradeState is UPGRADE_ERROR_ROLLBACK_FAILED', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'UPGRADE_ERROR_ROLLBACK_FAILED',
                    },
                },
            };

            const received = findUpgradeState(testUpgradeStatus);

            const expected = {
                displayValue: 'Upgrade failed. Rollback failed.',
                type: 'failure',
                actionText: 'Retry upgrade',
            };
            expect(received).toEqual(expected);
        });

        it('should return "Upgrade timed out." if upgradeState is UPGRADE_TIMED_OUT', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'UPGRADE_TIMED_OUT',
                    },
                },
            };

            const received = findUpgradeState(testUpgradeStatus);

            const expected = {
                displayValue: 'Upgrade timed out.',
                type: 'failure',
                actionText: 'Retry upgrade',
            };
            expect(received).toEqual(expected);
        });

        it('should return "Upgrade error unknown." if upgradeState is UPGRADE_ERROR_UNKNOWN', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'UPGRADE_ERROR_UNKNOWN',
                    },
                },
            };

            const received = findUpgradeState(testUpgradeStatus);

            const expected = {
                displayValue: 'Upgrade error unknown',
                type: 'failure',
                actionText: 'Retry upgrade',
            };
            expect(received).toEqual(expected);
        });

        it('should return "Unknown upgrade state. Contact Support." if upgradeState does not match known progress', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'SNAFU',
                    },
                },
            };

            const received = findUpgradeState(testUpgradeStatus);

            const expected = {
                displayValue: 'Unknown upgrade state. Contact Support.',
                type: 'intervention',
            };
            expect(received).toEqual(expected);
        });
    });

    describe('get credential expiration status', () => {
        it('should return HEALTHY when more than a month before expiration', () => {
            const status = getCredentialExpirationStatus(
                {
                    sensorCertNotBefore: '2022-01-01T00:00:00Z',
                    sensorCertExpiry: '2022-12-31T23:59:59Z',
                },
                new Date('2022-03-10T08:51:18Z')
            );

            expect(status).toBe('HEALTHY');
        });
        it('should return DEGRADED when less than a month before expiration', () => {
            const status = getCredentialExpirationStatus(
                {
                    sensorCertNotBefore: '2022-01-01T00:00:00Z',
                    sensorCertExpiry: '2022-12-31T23:59:59Z',
                },
                new Date('2022-12-20T09:15:23Z')
            );

            expect(status).toBe('DEGRADED');
        });
        it('should return UNHEALTHY when less than a week before expiration', () => {
            const status = getCredentialExpirationStatus(
                {
                    sensorCertNotBefore: '2022-01-01T00:00:00Z',
                    sensorCertExpiry: '2022-12-31T23:59:59Z',
                },
                new Date('2022-12-30T09:15:23Z')
            );

            expect(status).toBe('UNHEALTHY');
        });
        it('should return UNHEALTHY when expired', () => {
            const status = getCredentialExpirationStatus(
                {
                    sensorCertNotBefore: '2022-12-30T00:00:00Z',
                    sensorCertExpiry: '2022-12-31T23:59:59Z',
                },
                new Date('2023-01-01T09:15:23Z')
            );

            expect(status).toBe('UNHEALTHY');
        });
        it('should return HEALTHY when more than an hour before expiration for short lived certs', () => {
            const status = getCredentialExpirationStatus(
                {
                    sensorCertNotBefore: '2022-03-10T00:00:00Z',
                    sensorCertExpiry: '2022-03-10T03:00:00Z',
                },
                new Date('2022-03-10T01:30:45Z')
            );

            expect(status).toBe('HEALTHY');
        });
        it('should return DEGRADED when more than an hour before expiration for short lived certs', () => {
            const status = getCredentialExpirationStatus(
                {
                    sensorCertNotBefore: '2022-03-10T00:00:00Z',
                    sensorCertExpiry: '2022-03-10T03:00:00Z',
                },
                new Date('2022-03-10T02:30:45Z')
            );

            expect(status).toBe('DEGRADED');
        });
        it('should return UNHEALTHY when less than an 15 min before expiration for short lived certs', () => {
            const status = getCredentialExpirationStatus(
                {
                    sensorCertNotBefore: '2022-03-10T00:00:00Z',
                    sensorCertExpiry: '2022-03-10T03:00:00Z',
                },
                new Date('2022-03-10T02:50:45Z')
            );

            expect(status).toBe('UNHEALTHY');
        });
        it('should return UNHEALTHY when the short lived cert expired', () => {
            const status = getCredentialExpirationStatus(
                {
                    sensorCertNotBefore: '2022-03-10T00:00:00Z',
                    sensorCertExpiry: '2022-03-10T03:00:00Z',
                },
                new Date('2022-03-10T04:12:43Z')
            );

            expect(status).toBe('UNHEALTHY');
        });
    });
});
