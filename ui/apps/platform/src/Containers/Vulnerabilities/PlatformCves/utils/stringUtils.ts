import { ClusterType } from 'types/cluster.proto';
import { ensureExhaustive } from 'utils/type.utils';

export function displayClusterType(type: ClusterType): string {
    switch (type) {
        case 'GENERIC_CLUSTER':
            return 'Generic';
        case 'KUBERNETES_CLUSTER':
            return 'Kubernetes';
        case 'OPENSHIFT_CLUSTER':
        case 'OPENSHIFT4_CLUSTER':
            return 'OCP';
        default:
            return ensureExhaustive(type);
    }
}

export function displayCveType(cveType: string): string {
    switch (cveType) {
        case 'K8S_CVE':
            return 'Kubernetes';
        case 'ISTIO_CVE':
            return 'Istio';
        case 'OPENSHIFT_CVE':
            return 'Openshift';
        default:
            return cveType;
    }
}
