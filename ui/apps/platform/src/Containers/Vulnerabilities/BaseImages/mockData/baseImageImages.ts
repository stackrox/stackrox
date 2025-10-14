import { BaseImageApplicationImage } from '../types';

/**
 * Mock application images using each base image
 */
export const MOCK_BASE_IMAGE_IMAGES: Record<string, BaseImageApplicationImage[]> = {
    'base-image-1': [
        // ubuntu:22.04
        {
            imageId: 'sha256:app1',
            name: 'myapp:v1.2.3',
            sha: 'sha256:def456789abcdef123456789abcdef123456789abcdef123456789abcdef1234',
            lastScanned: '2025-10-13T10:00:00Z',
            cveCount: {
                critical: 7,
                high: 15,
                medium: 28,
                low: 10,
                total: 60,
                baseImageCves: 48,
                applicationLayerCves: 12,
            },
            deploymentCount: 3,
        },
        {
            imageId: 'sha256:app2',
            name: 'web-frontend:2.1.0',
            sha: 'sha256:abc789def123abc789def123abc789def123abc789def123abc789def1234567',
            lastScanned: '2025-10-13T09:30:00Z',
            cveCount: {
                critical: 5,
                high: 12,
                medium: 25,
                low: 8,
                total: 50,
                baseImageCves: 48,
                applicationLayerCves: 2,
            },
            deploymentCount: 2,
        },
        {
            imageId: 'sha256:app3',
            name: 'data-processor:3.0.1',
            sha: 'sha256:123abc456def789abc123def456abc789def123abc456def789abc123def4567',
            lastScanned: '2025-10-13T08:15:00Z',
            cveCount: {
                critical: 6,
                high: 14,
                medium: 30,
                low: 9,
                total: 59,
                baseImageCves: 48,
                applicationLayerCves: 11,
            },
            deploymentCount: 1,
        },
    ],
    'base-image-2': [
        // alpine:3.18
        {
            imageId: 'sha256:app4',
            name: 'microservice-a:1.0.0',
            sha: 'sha256:fed321cba987fed321cba987fed321cba987fed321cba987fed321cba9876543',
            lastScanned: '2025-10-13T11:00:00Z',
            cveCount: {
                critical: 0,
                high: 5,
                medium: 8,
                low: 3,
                total: 16,
                baseImageCves: 10,
                applicationLayerCves: 6,
            },
            deploymentCount: 4,
        },
        {
            imageId: 'sha256:app5',
            name: 'api-gateway:2.3.1',
            sha: 'sha256:987fed321cba654fed321cba987fed321cba987fed321cba987fed321cba9876',
            lastScanned: '2025-10-13T10:45:00Z',
            cveCount: {
                critical: 1,
                high: 4,
                medium: 7,
                low: 2,
                total: 14,
                baseImageCves: 10,
                applicationLayerCves: 4,
            },
            deploymentCount: 2,
        },
    ],
    'base-image-3': [
        // node:18-alpine (IN_PROGRESS, no images yet)
    ],
    'base-image-4': [
        // nginx:1.25-alpine
        {
            imageId: 'sha256:app6',
            name: 'static-site:1.0.0',
            sha: 'sha256:456def789abc123def456abc789def123abc456def789abc123def456abc7890',
            lastScanned: '2025-10-12T17:00:00Z',
            cveCount: {
                critical: 2,
                high: 7,
                medium: 18,
                low: 7,
                total: 34,
                baseImageCves: 30,
                applicationLayerCves: 4,
            },
            deploymentCount: 5,
        },
        {
            imageId: 'sha256:app7',
            name: 'reverse-proxy:2.5.0',
            sha: 'sha256:789abc123def456abc789def123abc456def789abc123def456abc789def1234',
            lastScanned: '2025-10-12T16:30:00Z',
            cveCount: {
                critical: 2,
                high: 8,
                medium: 16,
                low: 6,
                total: 32,
                baseImageCves: 30,
                applicationLayerCves: 2,
            },
            deploymentCount: 3,
        },
    ],
};
