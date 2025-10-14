import { BaseImageDeployment } from '../types';

/**
 * Mock deployments using each base image
 */
export const MOCK_BASE_IMAGE_DEPLOYMENTS: Record<string, BaseImageDeployment[]> = {
    'base-image-1': [
        // ubuntu:22.04
        {
            deploymentId: 'deploy-1',
            name: 'web-frontend',
            namespace: 'production',
            cluster: 'prod-us-west-1',
            image: 'myapp:v1.2.3',
            cveCount: {
                critical: 7,
                high: 15,
                medium: 28,
                low: 10,
                total: 60,
            },
            riskPriority: 85,
        },
        {
            deploymentId: 'deploy-2',
            name: 'api-server',
            namespace: 'production',
            cluster: 'prod-us-east-1',
            image: 'web-frontend:2.1.0',
            cveCount: {
                critical: 5,
                high: 12,
                medium: 25,
                low: 8,
                total: 50,
            },
            riskPriority: 72,
        },
        {
            deploymentId: 'deploy-3',
            name: 'background-worker',
            namespace: 'production',
            cluster: 'prod-us-west-1',
            image: 'myapp:v1.2.3',
            cveCount: {
                critical: 7,
                high: 15,
                medium: 28,
                low: 10,
                total: 60,
            },
            riskPriority: 68,
        },
        {
            deploymentId: 'deploy-4',
            name: 'data-processor',
            namespace: 'production',
            cluster: 'prod-eu-west-1',
            image: 'data-processor:3.0.1',
            cveCount: {
                critical: 6,
                high: 14,
                medium: 30,
                low: 9,
                total: 59,
            },
            riskPriority: 75,
        },
        {
            deploymentId: 'deploy-5',
            name: 'web-frontend-staging',
            namespace: 'staging',
            cluster: 'staging-us-west-1',
            image: 'web-frontend:2.1.0',
            cveCount: {
                critical: 5,
                high: 12,
                medium: 25,
                low: 8,
                total: 50,
            },
            riskPriority: 45,
        },
    ],
    'base-image-2': [
        // alpine:3.18
        {
            deploymentId: 'deploy-6',
            name: 'microservice-a',
            namespace: 'production',
            cluster: 'prod-us-west-1',
            image: 'microservice-a:1.0.0',
            cveCount: {
                critical: 0,
                high: 5,
                medium: 8,
                low: 3,
                total: 16,
            },
            riskPriority: 42,
        },
        {
            deploymentId: 'deploy-7',
            name: 'api-gateway',
            namespace: 'production',
            cluster: 'prod-us-east-1',
            image: 'api-gateway:2.3.1',
            cveCount: {
                critical: 1,
                high: 4,
                medium: 7,
                low: 2,
                total: 14,
            },
            riskPriority: 55,
        },
        {
            deploymentId: 'deploy-8',
            name: 'microservice-a-replica',
            namespace: 'production',
            cluster: 'prod-eu-west-1',
            image: 'microservice-a:1.0.0',
            cveCount: {
                critical: 0,
                high: 5,
                medium: 8,
                low: 3,
                total: 16,
            },
            riskPriority: 38,
        },
    ],
    'base-image-3': [
        // node:18-alpine (IN_PROGRESS, no deployments yet)
    ],
    'base-image-4': [
        // nginx:1.25-alpine
        {
            deploymentId: 'deploy-9',
            name: 'static-site',
            namespace: 'production',
            cluster: 'prod-us-west-1',
            image: 'static-site:1.0.0',
            cveCount: {
                critical: 2,
                high: 7,
                medium: 18,
                low: 7,
                total: 34,
            },
            riskPriority: 65,
        },
        {
            deploymentId: 'deploy-10',
            name: 'reverse-proxy',
            namespace: 'production',
            cluster: 'prod-us-east-1',
            image: 'reverse-proxy:2.5.0',
            cveCount: {
                critical: 2,
                high: 8,
                medium: 16,
                low: 6,
                total: 32,
            },
            riskPriority: 70,
        },
        {
            deploymentId: 'deploy-11',
            name: 'static-site-cdn',
            namespace: 'production',
            cluster: 'prod-eu-west-1',
            image: 'static-site:1.0.0',
            cveCount: {
                critical: 2,
                high: 7,
                medium: 18,
                low: 7,
                total: 34,
            },
            riskPriority: 58,
        },
        {
            deploymentId: 'deploy-12',
            name: 'reverse-proxy-backup',
            namespace: 'production',
            cluster: 'prod-us-west-1',
            image: 'reverse-proxy:2.5.0',
            cveCount: {
                critical: 2,
                high: 8,
                medium: 16,
                low: 6,
                total: 32,
            },
            riskPriority: 62,
        },
    ],
};
