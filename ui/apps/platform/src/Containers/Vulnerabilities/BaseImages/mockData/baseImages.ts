import { BaseImage } from '../types';

/**
 * Mock data for tracked base images
 */
export const MOCK_BASE_IMAGES: BaseImage[] = [
    {
        id: 'base-image-1',
        name: 'ubuntu:22.04',
        normalizedName: 'docker.io/library/ubuntu:22.04',
        scanningStatus: 'COMPLETED',
        lastScanned: '2025-10-13T10:30:00Z',
        createdAt: '2025-10-10T08:00:00Z',
        cveCount: {
            critical: 5,
            high: 12,
            medium: 23,
            low: 8,
            total: 48,
        },
        imageCount: 15,
        deploymentCount: 12,
        lastBaseLayerIndex: 5,
    },
    {
        id: 'base-image-2',
        name: 'alpine:3.18',
        normalizedName: 'docker.io/library/alpine:3.18',
        scanningStatus: 'COMPLETED',
        lastScanned: '2025-10-13T09:15:00Z',
        createdAt: '2025-10-09T14:20:00Z',
        cveCount: {
            critical: 0,
            high: 3,
            medium: 5,
            low: 2,
            total: 10,
        },
        imageCount: 8,
        deploymentCount: 5,
        lastBaseLayerIndex: 3,
    },
    {
        id: 'base-image-3',
        name: 'node:18-alpine',
        normalizedName: 'docker.io/library/node:18-alpine',
        scanningStatus: 'IN_PROGRESS',
        lastScanned: null,
        createdAt: '2025-10-13T11:00:00Z',
        cveCount: {
            critical: 0,
            high: 0,
            medium: 0,
            low: 0,
            total: 0,
        },
        imageCount: 0,
        deploymentCount: 0,
        lastBaseLayerIndex: 0,
    },
    {
        id: 'base-image-4',
        name: 'nginx:1.25-alpine',
        normalizedName: 'docker.io/library/nginx:1.25-alpine',
        scanningStatus: 'COMPLETED',
        lastScanned: '2025-10-12T16:45:00Z',
        createdAt: '2025-10-08T10:30:00Z',
        cveCount: {
            critical: 2,
            high: 7,
            medium: 15,
            low: 6,
            total: 30,
        },
        imageCount: 6,
        deploymentCount: 8,
        lastBaseLayerIndex: 4,
    },
];
