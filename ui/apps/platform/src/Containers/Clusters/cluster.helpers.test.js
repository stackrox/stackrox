import {
    formatBuildDate,
    formatCloudProvider,
    formatKubernetesVersion,
    getCredentialExpirationStatus,
} from './cluster.helpers';

describe('cluster helpers', () => {
    describe('formatKubernetesVersion', () => {
        it('should return version of Kubernetes from the orchestrator metadata response', () => {
            const orchestratorMetadata = {
                version: 'v1.24.7-gke.900',
                buildDate: '2022-10-26T09:25:34Z',
                apiVersions: [
                    'admissionregistration.k8s.io/v1',
                    'apiextensions.k8s.io/v1',
                    'apiregistration.k8s.io/v1',
                    'apps/v1',
                    'authentication.k8s.io/v1',
                    'authorization.k8s.io/v1',
                    'auto.gke.io/v1',
                    'auto.gke.io/v1alpha1',
                    'autoscaling/v1',
                    'autoscaling/v2',
                    'autoscaling/v2beta1',
                    'autoscaling/v2beta2',
                    'batch/v1',
                    'batch/v1beta1',
                    'certificates.k8s.io/v1',
                    'cloud.google.com/v1',
                    'cloud.google.com/v1beta1',
                    'coordination.k8s.io/v1',
                    'crd.projectcalico.org/v1',
                    'discovery.k8s.io/v1',
                    'discovery.k8s.io/v1beta1',
                    'events.k8s.io/v1',
                    'flowcontrol.apiserver.k8s.io/v1beta1',
                    'flowcontrol.apiserver.k8s.io/v1beta2',
                    'hub.gke.io/v1',
                    'internal.autoscaling.gke.io/v1alpha1',
                    'metrics.k8s.io/v1beta1',
                    'migration.k8s.io/v1alpha1',
                    'networking.gke.io/v1',
                    'networking.gke.io/v1beta1',
                    'networking.gke.io/v1beta2',
                    'networking.k8s.io/v1',
                    'node.k8s.io/v1',
                    'node.k8s.io/v1beta1',
                    'nodemanagement.gke.io/v1alpha1',
                    'policy/v1',
                    'policy/v1beta1',
                    'rbac.authorization.k8s.io/v1',
                    'scheduling.k8s.io/v1',
                    'snapshot.storage.k8s.io/v1',
                    'snapshot.storage.k8s.io/v1beta1',
                    'storage.k8s.io/v1',
                    'storage.k8s.io/v1beta1',
                    'v1',
                ],
            };

            const displayValue = formatKubernetesVersion(orchestratorMetadata);

            expect(displayValue).toEqual('v1.24.7-gke.900');
        });

        it('should return appropriate message if orchestrator metadata response not available', () => {
            const orchestratorMetadata = null;

            const displayValue = formatKubernetesVersion(orchestratorMetadata);

            expect(displayValue).toEqual('Not available');
        });
    });

    describe('formatBuildDate', () => {
        it('should return formatted build date from the orchestrator metadata response', () => {
            const orchestratorMetadata = {
                version: 'v1.24.7-gke.900',
                buildDate: '2022-10-26T09:25:34Z',
                apiVersions: [
                    'admissionregistration.k8s.io/v1',
                    'apiextensions.k8s.io/v1',
                    'apiregistration.k8s.io/v1',
                    'apps/v1',
                    'authentication.k8s.io/v1',
                    'authorization.k8s.io/v1',
                    'auto.gke.io/v1',
                    'auto.gke.io/v1alpha1',
                    'autoscaling/v1',
                    'autoscaling/v2',
                    'autoscaling/v2beta1',
                    'autoscaling/v2beta2',
                    'batch/v1',
                    'batch/v1beta1',
                    'certificates.k8s.io/v1',
                    'cloud.google.com/v1',
                    'cloud.google.com/v1beta1',
                    'coordination.k8s.io/v1',
                    'crd.projectcalico.org/v1',
                    'discovery.k8s.io/v1',
                    'discovery.k8s.io/v1beta1',
                    'events.k8s.io/v1',
                    'flowcontrol.apiserver.k8s.io/v1beta1',
                    'flowcontrol.apiserver.k8s.io/v1beta2',
                    'hub.gke.io/v1',
                    'internal.autoscaling.gke.io/v1alpha1',
                    'metrics.k8s.io/v1beta1',
                    'migration.k8s.io/v1alpha1',
                    'networking.gke.io/v1',
                    'networking.gke.io/v1beta1',
                    'networking.gke.io/v1beta2',
                    'networking.k8s.io/v1',
                    'node.k8s.io/v1',
                    'node.k8s.io/v1beta1',
                    'nodemanagement.gke.io/v1alpha1',
                    'policy/v1',
                    'policy/v1beta1',
                    'rbac.authorization.k8s.io/v1',
                    'scheduling.k8s.io/v1',
                    'snapshot.storage.k8s.io/v1',
                    'snapshot.storage.k8s.io/v1beta1',
                    'storage.k8s.io/v1',
                    'storage.k8s.io/v1beta1',
                    'v1',
                ],
            };

            const displayValue = formatBuildDate(orchestratorMetadata);

            expect(displayValue).toEqual('Oct 26, 2022');
        });

        it('should return appropriate message if orchestrator metadata response not available', () => {
            const orchestratorMetadata = null;

            const displayValue = formatBuildDate(orchestratorMetadata);

            expect(displayValue).toEqual('Not available');
        });
    });

    describe('formatCloudProvider', () => {
        it('should return GCP from the provider metadata response', () => {
            const providerMetadata = {
                region: 'us-central1',
                zone: 'us-central1-b',
                google: {
                    project: 'ultra-current-825',
                    clusterName: 'dyjkitia-prod',
                },
                verified: true,
            };

            const displayValue = formatCloudProvider(providerMetadata);

            expect(displayValue).toEqual('GCP us-central1');
        });

        it('should return Azure from the provider metadata response', () => {
            const providerMetadata = {
                region: 'us-central2',
                zone: 'us-central2-c',
                azure: {
                    project: 'ultra-current-825',
                    clusterName: 'dyjkitia-prod',
                },
                verified: true,
            };

            const displayValue = formatCloudProvider(providerMetadata);

            expect(displayValue).toEqual('Azure us-central2');
        });

        it('should return AWX from the provider metadata response', () => {
            const providerMetadata = {
                region: 'us-east1',
                zone: 'us-east1-c',
                aws: {
                    project: 'ultra-current-825',
                    clusterName: 'dyjkitia-prod',
                },
                verified: true,
            };

            const displayValue = formatCloudProvider(providerMetadata);

            expect(displayValue).toEqual('AWS us-east1');
        });

        it('should return appropriate message if provider metadata response not available', () => {
            const providerMetadata = null;

            const displayValue = formatCloudProvider(providerMetadata);

            expect(displayValue).toEqual('Not available');
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
        it('should return UNHEALTHY when less than a month before expiry and sensorCertNotBefore is undefined', () => {
            const status = getCredentialExpirationStatus(
                {
                    sensorCertExpiry: '2022-12-31T23:59:59Z',
                },
                new Date('2022-12-30T12:00:00Z')
            );

            expect(status).toBe('UNHEALTHY');
        });
    });
});
