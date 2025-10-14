/**
 * TypeScript type definitions for Base Images feature
 */

/**
 * Status of base image scanning
 */
export type ScanningStatus = 'IN_PROGRESS' | 'COMPLETED' | 'FAILED';

/**
 * CVE severity levels
 */
export type CVESeverity = 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW';

/**
 * CVE count breakdown by severity
 */
export interface CVECount {
    critical: number;
    high: number;
    medium: number;
    low: number;
    total: number;
}

/**
 * Extended CVE count with base/app layer distinction
 */
export interface ExtendedCVECount extends CVECount {
    baseImageCves: number;
    applicationLayerCves: number;
}

/**
 * Base image entity
 */
export interface BaseImage {
    id: string;
    name: string;
    normalizedName: string;
    scanningStatus: ScanningStatus;
    lastScanned: string | null;
    createdAt: string;
    cveCount: CVECount;
    imageCount: number;
    deploymentCount: number;
    lastBaseLayerIndex: number;
}

/**
 * Component affected by a CVE
 */
export interface CVEComponent {
    name: string;
    version: string;
    layerIndex: number;
}

/**
 * CVE found in a base image
 */
export interface BaseImageCVE {
    cveId: string;
    severity: CVESeverity;
    cvssScore: number;
    summary: string;
    fixedBy: string;
    components: CVEComponent[];
}

/**
 * Application image using a base image
 */
export interface BaseImageApplicationImage {
    imageId: string;
    name: string;
    sha: string;
    lastScanned: string;
    cveCount: ExtendedCVECount;
    deploymentCount: number;
}

/**
 * Deployment using a base image
 */
export interface BaseImageDeployment {
    deploymentId: string;
    name: string;
    namespace: string;
    cluster: string;
    image: string;
    cveCount: CVECount;
    riskPriority: number;
}

/**
 * Base image information embedded in application image
 */
export interface ImageBaseInfo {
    name: string;
    isManaged: boolean;
    lastLayerIndex: number;
    baseImageId: string;
}

/**
 * Tab names for base image detail page
 */
export type BaseImageDetailTab = 'cves' | 'images' | 'deployments';

/**
 * Mock data collections
 */
export interface BaseImageMockData {
    baseImages: BaseImage[];
    baseImageCVEs: Record<string, BaseImageCVE[]>;
    baseImageImages: Record<string, BaseImageApplicationImage[]>;
    baseImageDeployments: Record<string, BaseImageDeployment[]>;
}
