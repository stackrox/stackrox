import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';

import {
    formatClusterType,
    formatEnabledDisabledField,
    formatCollectionMethod,
    formatLastCheckIn,
    formatSensorVersion,
    parseUpgradeStatus,
    getUpgradeableClusters
} from './cluster.helpers';

describe('cluster helpers', () => {
    describe('formatClusterType', () => {
        it('should return the string "Kubernetes" if passed a value of KUBERNETES_CLUSTER', () => {
            const testCluster = {
                type: 'KUBERNETES_CLUSTER'
            };

            const displayValue = formatClusterType(testCluster.type);

            expect(displayValue).toEqual('Kubernetes');
        });

        it('should return the string "OpenShift" if passed a value of OPENSHIFT_CLUSTER', () => {
            const testCluster = {
                type: 'OPENSHIFT_CLUSTER'
            };

            const displayValue = formatClusterType(testCluster.type);

            expect(displayValue).toEqual('OpenShift');
        });
    });

    describe('formatCollectionMethod', () => {
        it('should return the string "None" if passed a value of NO_COLLECTION', () => {
            const testCluster = {
                collectionMethod: 'NO_COLLECTION'
            };

            const displayValue = formatCollectionMethod(testCluster.collectionMethod);

            expect(displayValue).toEqual('None');
        });

        it('should return the string "Kernel Module" if passed a value of KERNEL_MODULE', () => {
            const testCluster = {
                collectionMethod: 'KERNEL_MODULE'
            };

            const displayValue = formatCollectionMethod(testCluster.collectionMethod);

            expect(displayValue).toEqual('Kernel Module');
        });

        it('should return the string "eBPF" if passed a value of EBPF', () => {
            const testCluster = {
                collectionMethod: 'EBPF'
            };

            const displayValue = formatCollectionMethod(testCluster.collectionMethod);

            expect(displayValue).toEqual('eBPF');
        });
    });

    describe('formatEnabledDisabledField', () => {
        it('should return the string "Disabled" if passed a value of false', () => {
            const testCluster = {
                admissionController: false
            };

            const displayValue = formatEnabledDisabledField(testCluster.admissionController);

            expect(displayValue).toEqual('Disabled');
        });

        it('should return the string "Enabled" if passed a value of false', () => {
            const testCluster = {
                admissionController: true
            };

            const displayValue = formatEnabledDisabledField(testCluster.admissionController);

            expect(displayValue).toEqual('Enabled');
        });
    });

    describe('formatLastCheckIn', () => {
        it('should return a formatted date string if passed a status object with a lastContact field', () => {
            const testCluster = {
                status: {
                    lastContact: '2019-08-28T17:20:29.156602Z'
                }
            };

            const displayValue = formatLastCheckIn(testCluster.status);

            const expectedDateFormat = dateFns.format(
                testCluster.status.lastContact,
                dateTimeFormat
            );
            expect(displayValue).toEqual(expectedDateFormat);
        });

        it('should return a "N/A" if passed a status object with null lastContact field', () => {
            const testCluster = {
                status: {
                    lastContact: null
                }
            };

            const displayValue = formatLastCheckIn(testCluster.status);

            expect(displayValue).toEqual('N/A');
        });

        it('should return a "N/A" if passed a status object with null status field', () => {
            const testCluster = {
                status: null
            };

            const displayValue = formatLastCheckIn(testCluster.status);

            expect(displayValue).toEqual('N/A');
        });
    });

    describe('formatSensorVersion', () => {
        it('should return sensor version string if passed a status object with a sensorVersion field', () => {
            const testCluster = {
                status: {
                    sensorVersion: 'sensorVersion'
                }
            };

            const displayValue = formatSensorVersion(testCluster.status);

            expect(displayValue).toEqual(testCluster.status.sensorVersion);
        });

        it('should return a "Not Running" if passed a status object with null sensorVersion field', () => {
            const testCluster = {
                status: {
                    sensorVersion: null
                }
            };

            const displayValue = formatSensorVersion(testCluster.status);

            expect(displayValue).toEqual('Not Running');
        });

        it('should return a "Not Running" if passed a status object with null status field', () => {
            const testCluster = {
                status: null
            };

            const displayValue = formatSensorVersion(testCluster.status);

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
                                    upgradeState: 'UPGRADE_COMPLETE'
                                }
                            }
                        }
                    }
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
                                    upgradeState: 'UPGRADE_COMPLETE'
                                }
                            }
                        }
                    }
                }
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
                                    upgradeState: 'UPGRADE_COMPLETE'
                                }
                            }
                        }
                    }
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
                                    upgradeState: 'UPGRADE_COMPLETE'
                                }
                            }
                        }
                    }
                }
            ];

            const upgradeableClusters = getUpgradeableClusters(clusters);

            expect(upgradeableClusters.length).toEqual(1);
        });
    });

    describe('parseUpgradeStatus', () => {
        it('should return null if upgradeStatus is null', () => {
            const testUpgradeStatus = null;

            const displayValue = parseUpgradeStatus(testUpgradeStatus);

            expect(displayValue).toEqual(null);
        });

        it('should return "On the latest version" if upgradeStatus -> upgradability is UP_TO_DATE', () => {
            const testUpgradeStatus = {
                upgradability: 'UP_TO_DATE'
            };

            const displayValue = parseUpgradeStatus(testUpgradeStatus);

            const expected = { displayValue: 'On the latest version', type: 'current' };
            expect(displayValue).toEqual(expected);
        });

        it('should return "Manual upgrade required" if upgradeStatus -> upgradability is MANUAL_UPGRADE_REQUIRED', () => {
            const testUpgradeStatus = {
                upgradability: 'MANUAL_UPGRADE_REQUIRED'
            };

            const displayValue = parseUpgradeStatus(testUpgradeStatus);

            const expected = { displayValue: 'Manual upgrade required', type: 'intervention' };
            expect(displayValue).toEqual(expected);
        });

        it('should return "Upgrade available" if there is no mostRecentProcess ', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE'
            };

            const displayValue = parseUpgradeStatus(testUpgradeStatus);

            const expected = {
                action: {
                    actionText: 'Upgrade available'
                },
                type: 'download'
            };
            expect(displayValue).toEqual(expected);
        });

        it('should return "Upgrade available" if upgradeStatus -> upgradability is AUTO_UPGRADE_POSSIBLE but mostRecentProgress is not active and is COMPLETE', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: false,
                    progress: {
                        upgradeState: 'UPGRADE_COMPLETE'
                    }
                }
            };

            const displayValue = parseUpgradeStatus(testUpgradeStatus);

            const expected = {
                action: {
                    actionText: 'Upgrade available'
                },
                type: 'download'
            };
            expect(displayValue).toEqual(expected);
        });

        it('should print the error (and "Retry Upgrade") if upgradeStatus -> upgradability is AUTO_UPGRADE_POSSIBLE but mostRecentProgress is not active and failed', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: false,
                    progress: {
                        upgradeState: 'PRE_FLIGHT_CHECKS_FAILED'
                    }
                }
            };

            const displayValue = parseUpgradeStatus(testUpgradeStatus);

            const expected = {
                displayValue: 'Pre-flight checks failed',
                type: 'failure',
                action: {
                    actionText: 'Retry upgrade'
                }
            };
            expect(displayValue).toEqual(expected);
        });

        it('should return "Upgrade initializing" if upgradeStatus -> upgradability is AUTO_UPGRADE_POSSIBLE and upgradeState is UPGRADE_INITIALIZING', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'UPGRADE_INITIALIZING'
                    }
                }
            };

            const displayValue = parseUpgradeStatus(testUpgradeStatus);

            const expected = {
                displayValue: 'Upgrade initializing',
                type: 'progress'
            };
            expect(displayValue).toEqual(expected);
        });

        it('should return "Upgrader launching" if upgradeState is UPGRADER_LAUNCHING', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'UPGRADER_LAUNCHING'
                    }
                }
            };

            const displayValue = parseUpgradeStatus(testUpgradeStatus);

            const expected = { displayValue: 'Upgrader launching', type: 'progress' };
            expect(displayValue).toEqual(expected);
        });

        it('should return "Upgrader launched" if upgradeState is UPGRADER_LAUNCHED', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'UPGRADER_LAUNCHED'
                    }
                }
            };

            const displayValue = parseUpgradeStatus(testUpgradeStatus);

            const expected = { displayValue: 'Upgrader launched', type: 'progress' };
            expect(displayValue).toEqual(expected);
        });

        it('should return "Pre-flight checks complete" if upgradeState is PRE_FLIGHT_CHECKS_COMPLETE', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'PRE_FLIGHT_CHECKS_COMPLETE'
                    }
                }
            };

            const displayValue = parseUpgradeStatus(testUpgradeStatus);

            const expected = { displayValue: 'Pre-flight checks complete', type: 'progress' };
            expect(displayValue).toEqual(expected);
        });

        it('should return "Pre-flight checks failed." if upgradeState is PRE_FLIGHT_CHECKS_FAILED', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'PRE_FLIGHT_CHECKS_FAILED'
                    }
                }
            };

            const displayValue = parseUpgradeStatus(testUpgradeStatus);

            const expected = {
                action: {
                    actionText: 'Retry upgrade'
                },
                displayValue: 'Pre-flight checks failed',
                type: 'failure'
            };
            expect(displayValue).toEqual(expected);
        });

        it('should return "Upgrade Operations Done" if upgradeState is UPGRADE_OPERATIONS_DONE', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'UPGRADE_OPERATIONS_DONE'
                    }
                }
            };

            const displayValue = parseUpgradeStatus(testUpgradeStatus);

            const expected = { displayValue: 'Upgrade operations done', type: 'progress' };
            expect(displayValue).toEqual(expected);
        });

        it('should return "Upgrade Operations Complete" if upgradeState is UPGRADE_COMPLETE', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'UPGRADE_COMPLETE'
                    }
                }
            };

            const displayValue = parseUpgradeStatus(testUpgradeStatus);

            const expected = { displayValue: 'Upgrade complete', type: 'current' };
            expect(displayValue).toEqual(expected);
        });

        it('should return "Upgrade failed. Rolled back." if upgradeState is UPGRADE_ERROR_ROLLED_BACK', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'UPGRADE_ERROR_ROLLED_BACK'
                    }
                }
            };

            const displayValue = parseUpgradeStatus(testUpgradeStatus);

            const expected = {
                action: {
                    actionText: 'Retry upgrade'
                },
                displayValue: 'Upgrade failed. Rolled back.',
                type: 'failure'
            };
            expect(displayValue).toEqual(expected);
        });

        it('should return "Upgrade failed. Rollback failed." if upgradeState is UPGRADE_ERROR_ROLLBACK_FAILED', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'UPGRADE_ERROR_ROLLBACK_FAILED'
                    }
                }
            };

            const displayValue = parseUpgradeStatus(testUpgradeStatus);

            const expected = {
                action: {
                    actionText: 'Retry upgrade'
                },
                displayValue: 'Upgrade failed. Rollback failed.',
                type: 'failure'
            };
            expect(displayValue).toEqual(expected);
        });

        it('should return "Upgrade timed out." if upgradeState is UPGRADE_TIMED_OUT', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'UPGRADE_TIMED_OUT'
                    }
                }
            };

            const displayValue = parseUpgradeStatus(testUpgradeStatus);

            const expected = {
                action: {
                    actionText: 'Retry upgrade'
                },
                displayValue: 'Upgrade timed out.',
                type: 'failure'
            };
            expect(displayValue).toEqual(expected);
        });

        it('should return "Upgrade timed out." if upgradeState is UPGRADE_TIMED_OUT', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'UPGRADE_ERROR_UNKNOWN'
                    }
                }
            };

            const displayValue = parseUpgradeStatus(testUpgradeStatus);

            const expected = {
                action: {
                    actionText: 'Retry upgrade'
                },
                displayValue: 'Upgrade error unknown',
                type: 'failure'
            };
            expect(displayValue).toEqual(expected);
        });

        it('should return "Indeterminate upgrade state!" if upgradeState does not match known progress', () => {
            const testUpgradeStatus = {
                upgradability: 'AUTO_UPGRADE_POSSIBLE',
                mostRecentProcess: {
                    active: true,
                    progress: {
                        upgradeState: 'SNAFU'
                    }
                }
            };

            const displayValue = parseUpgradeStatus(testUpgradeStatus);

            const expected = { displayValue: 'Indeterminate upgrade state!', type: 'intervention' };
            expect(displayValue).toEqual(expected);
        });
    });
});
