import { ensureExhaustive } from 'utils/type.utils';
import { EntityTab } from './types';

export const defaultImageSortFields = [
    'Image',
    'Operating system',
    'Deployment count',
    'Age',
    'Scan time',
];

export const imagesDefaultSort = {
    field: 'Image',
    direction: 'desc',
} as const;

export const defaultCVESortFields = ['Deployment', 'Cluster', 'Namespace'];

export const CVEsDefaultSort = {
    field: 'CVE',
    direction: 'asc',
} as const;

export const defaultDeploymentSortFields = ['Deployment', 'Cluster', 'Namespace'];

export const deploymentsDefaultSort = {
    field: 'Deployment',
    direction: 'asc',
} as const;

export function getDefaultSortOption(entityTab: EntityTab) {
    switch (entityTab) {
        case 'CVE':
            return CVEsDefaultSort;
        case 'Deployment':
            return deploymentsDefaultSort;
        case 'Image':
            return imagesDefaultSort;
        default:
            return ensureExhaustive(entityTab);
    }
}
