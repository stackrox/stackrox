/**
 * Utility to generate consistent mock data for base images
 * Ensures that the counts in summary cards match the actual table data
 */

import type {
    BaseImageCVE,
    BaseImageApplicationImage,
    BaseImageDeployment,
    BaseImage,
} from '../types';

/**
 * Generate CVEs to match the count in the base image summary
 */
export function generateCVEsForBaseImage(baseImage: BaseImage): BaseImageCVE[] {
    const { cveCount, lastBaseLayerIndex } = baseImage;
    const cves: BaseImageCVE[] = [];

    // Generate critical CVEs
    for (let i = 0; i < cveCount.critical; i += 1) {
        cves.push({
            cveId: `CVE-2024-${1000 + cves.length}`,
            severity: 'CRITICAL',
            cvssScore: 9.0 + Math.random(),
            summary: `Critical vulnerability ${i + 1} in base image component`,
            fixedBy: `1.${i}.0-fix`,
            components: [
                {
                    name: `critical-component-${i}`,
                    version: `1.${i}.0`,
                    layerIndex: Math.floor(Math.random() * lastBaseLayerIndex),
                },
            ],
        });
    }

    // Generate high CVEs
    for (let i = 0; i < cveCount.high; i += 1) {
        cves.push({
            cveId: `CVE-2024-${1000 + cves.length}`,
            severity: 'HIGH',
            cvssScore: 7.0 + Math.random() * 2,
            summary: `High severity vulnerability ${i + 1} affecting base components`,
            fixedBy: `2.${i}.0-fix`,
            components: [
                {
                    name: `high-component-${i}`,
                    version: `2.${i}.0`,
                    layerIndex: Math.floor(Math.random() * lastBaseLayerIndex),
                },
            ],
        });
    }

    // Generate medium CVEs
    for (let i = 0; i < cveCount.medium; i += 1) {
        cves.push({
            cveId: `CVE-2024-${1000 + cves.length}`,
            severity: 'MEDIUM',
            cvssScore: 4.0 + Math.random() * 3,
            summary: `Medium severity issue ${i + 1} in base image`,
            fixedBy: `3.${i}.0-fix`,
            components: [
                {
                    name: `medium-component-${i}`,
                    version: `3.${i}.0`,
                    layerIndex: Math.floor(Math.random() * lastBaseLayerIndex),
                },
            ],
        });
    }

    // Generate low CVEs
    for (let i = 0; i < cveCount.low; i += 1) {
        cves.push({
            cveId: `CVE-2024-${1000 + cves.length}`,
            severity: 'LOW',
            cvssScore: 0.1 + Math.random() * 3.9,
            summary: `Low severity vulnerability ${i + 1}`,
            fixedBy: `4.${i}.0-fix`,
            components: [
                {
                    name: `low-component-${i}`,
                    version: `4.${i}.0`,
                    layerIndex: Math.floor(Math.random() * lastBaseLayerIndex),
                },
            ],
        });
    }

    return cves;
}

/**
 * Generate images to match the count in the base image summary
 */
export function generateImagesForBaseImage(baseImage: BaseImage): BaseImageApplicationImage[] {
    const { imageCount, id, cveCount } = baseImage;
    const images: BaseImageApplicationImage[] = [];

    for (let i = 0; i < imageCount; i += 1) {
        // Vary the CVE counts for each image
        const baseCves = cveCount.total;
        const appCves = Math.floor(Math.random() * 15) + 1;
        const totalCves = baseCves + appCves;

        images.push({
            imageId: `sha256:image${id}-${i}`,
            name: `application-${i}:v1.${i}.0`,
            sha: `sha256:${Math.random().toString(36).substring(2, 15)}${Math.random().toString(36).substring(2, 15)}`,
            lastScanned: new Date(Date.now() - Math.random() * 86400000).toISOString(),
            cveCount: {
                critical: Math.floor(cveCount.critical * 1.1),
                high: Math.floor(cveCount.high * 1.1),
                medium: Math.floor(cveCount.medium * 1.2),
                low: Math.floor(cveCount.low * 1.1),
                total: totalCves,
                baseImageCves: baseCves,
                applicationLayerCves: appCves,
            },
            deploymentCount: Math.floor(Math.random() * 5) + 1,
        });
    }

    return images;
}

/**
 * Generate deployments to match the count in the base image summary
 */
export function generateDeploymentsForBaseImage(
    baseImage: BaseImage,
    images: BaseImageApplicationImage[]
): BaseImageDeployment[] {
    const { deploymentCount, id, cveCount } = baseImage;
    const deployments: BaseImageDeployment[] = [];

    const clusters = ['prod-us-west-1', 'prod-us-east-1', 'prod-eu-west-1', 'staging-us-west-1'];
    const namespaces = ['production', 'staging', 'development'];

    for (let i = 0; i < deploymentCount; i += 1) {
        // Pick a random image or create a generic one
        const image =
            images.length > 0
                ? images[Math.floor(Math.random() * images.length)]
                : { name: `app-${i}:latest`, cveCount: { ...cveCount, total: cveCount.total } };

        deployments.push({
            deploymentId: `deploy-${id}-${i}`,
            name: `deployment-${i}`,
            namespace: namespaces[Math.floor(Math.random() * namespaces.length)],
            cluster: clusters[Math.floor(Math.random() * clusters.length)],
            image: image.name,
            cveCount: {
                critical: image.cveCount.critical,
                high: image.cveCount.high,
                medium: image.cveCount.medium,
                low: image.cveCount.low,
                total: image.cveCount.total,
            },
            riskPriority: Math.floor(Math.random() * 60) + 40,
        });
    }

    return deployments;
}
