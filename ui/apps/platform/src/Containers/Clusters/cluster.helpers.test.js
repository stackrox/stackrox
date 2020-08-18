import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';

import {
    findUpgradeState,
    formatClusterType,
    formatConfiguredField,
    formatCollectionMethod,
    formatLastCheckIn,
    formatSensorVersion,
    getUpgradeableClusters,
    getCredentialExpirationProps,
} from './cluster.helpers';

describe('cluster helpers', () => {
    describe('formatClusterType', () => {
        it('should return the string "Kubernetes" if passed a value of KUBERNETES_CLUSTER', () => {
            const testCluster = {
                type: 'KUBERNETES_CLUSTER',
            };

            const displayValue = formatClusterType(testCluster.type);

            expect(displayValue).toEqual('Kubernetes');
        });

        it('should return the string "OpenShift" if passed a value of OPENSHIFT_CLUSTER', () => {
            const testCluster = {
                type: 'OPENSHIFT_CLUSTER',
            };

            const displayValue = formatClusterType(testCluster.type);

            expect(displayValue).toEqual('OpenShift');
        });
    });

    describe('formatCollectionMethod', () => {
        it('should return the string "None" if passed a value of NO_COLLECTION', () => {
            const testCluster = {
                collectionMethod: 'NO_COLLECTION',
            };

            const displayValue = formatCollectionMethod(testCluster.collectionMethod);

            expect(displayValue).toEqual('None');
        });

        it('should return the string "Kernel Module" if passed a value of KERNEL_MODULE', () => {
            const testCluster = {
                collectionMethod: 'KERNEL_MODULE',
            };

            const displayValue = formatCollectionMethod(testCluster.collectionMethod);

            expect(displayValue).toEqual('Kernel Module');
        });

        it('should return the string "eBPF" if passed a value of EBPF', () => {
            const testCluster = {
                collectionMethod: 'EBPF',
            };

            const displayValue = formatCollectionMethod(testCluster.collectionMethod);

            expect(displayValue).toEqual('eBPF');
        });
    });

    describe('formatConfiguredField', () => {
        it('should return the string "Not configured" if passed a value of false', () => {
            const testCluster = {
                admissionController: false,
            };

            const displayValue = formatConfiguredField(testCluster.admissionController);

            expect(displayValue).toEqual('Not configured');
        });

        it('should return the string "Configured" if passed a value of false', () => {
            const testCluster = {
                admissionController: true,
            };

            const displayValue = formatConfiguredField(testCluster.admissionController);

            expect(displayValue).toEqual('Configured');
        });
    });

    describe('formatLastCheckIn', () => {
        it('should return a formatted date string if passed a status object with a lastContact field', () => {
            const testCluster = {
                status: {
                    lastContact: '2019-08-28T17:20:29.156602Z',
                },
            };

            const displayValue = formatLastCheckIn(testCluster.status);

            const expectedDateFormat = dateFns.format(
                testCluster.status.lastContact,
                dateTimeFormat
            );
            expect(displayValue).toEqual(expectedDateFormat);
        });

        it('should return a "Not applicable" if passed a status object with null lastContact field', () => {
            const testCluster = {
                status: {
                    lastContact: null,
                },
            };

            const displayValue = formatLastCheckIn(testCluster.status);

            expect(displayValue).toEqual('Not applicable');
        });

        it('should return a "Not applicable" if passed a status object with null status field', () => {
            const testCluster = {
                status: null,
            };

            const displayValue = formatLastCheckIn(testCluster.status);

            expect(displayValue).toEqual('Not applicable');
        });
    });

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

        it('should return "Upgrade available" if there is no mostRecentProcess ', () => {
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

    describe('get credential expiration props', () => {
        function callWithExpiryAfterDays(days) {
            // Add a minute to account for any time that will pass before we call the function.
            const expiry = dateFns.addMinutes(dateFns.addDays(new Date(), days), 1);
            const props = getCredentialExpirationProps({ sensorCertExpiry: expiry });
            expect(props.sensorCertExpiry).toBe(expiry);
            return props;
        }

        it('should return null if null status', () => {
            expect(getCredentialExpirationProps(null)).toBe(null);
        });
        it('should return null if undefined expiry', () => {
            expect(getCredentialExpirationProps({})).toBe(null);
        });
        it('should return info if expiry is more than 30 days away', () => {
            const props = callWithExpiryAfterDays(31);
            expect(props.showExpiringSoon).toBe(false);
            expect(props.messageType).toBe('info');
            expect(props.diffInWords).toBe('1 month');
        });
        it('should return warn if expiry is less than 30 days away, but more than 7 days away', () => {
            const props = callWithExpiryAfterDays(9);
            expect(props.showExpiringSoon).toBe(true);
            expect(props.messageType).toBe('warn');
            expect(props.diffInWords).toBe('9 days');
        });
        it('should return error if expiry is less than 7 days away', () => {
            const props = callWithExpiryAfterDays(6);
            expect(props.showExpiringSoon).toBe(true);
            expect(props.messageType).toBe('error');
            expect(props.diffInWords).toBe('6 days');
        });
    });
});
