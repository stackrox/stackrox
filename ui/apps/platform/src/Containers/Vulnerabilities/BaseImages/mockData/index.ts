/**
 * Mock data exports for Base Images feature
 * Uses generated data to ensure consistency between summary cards and tables
 */

import { MOCK_BASE_IMAGES } from './baseImages';
import {
    generateCVEsForBaseImage,
    generateImagesForBaseImage,
    generateDeploymentsForBaseImage,
} from './mockDataGenerator';
import type { BaseImageCVE, BaseImageApplicationImage, BaseImageDeployment } from '../types';

// Generate consistent mock data from base images
export { MOCK_BASE_IMAGES } from './baseImages';

export const MOCK_BASE_IMAGE_CVES: Record<string, BaseImageCVE[]> = {};
export const MOCK_BASE_IMAGE_IMAGES: Record<string, BaseImageApplicationImage[]> = {};
export const MOCK_BASE_IMAGE_DEPLOYMENTS: Record<string, BaseImageDeployment[]> = {};

// Generate data for each base image
MOCK_BASE_IMAGES.forEach((baseImage) => {
    MOCK_BASE_IMAGE_CVES[baseImage.id] = generateCVEsForBaseImage(baseImage);
    MOCK_BASE_IMAGE_IMAGES[baseImage.id] = generateImagesForBaseImage(baseImage);
    MOCK_BASE_IMAGE_DEPLOYMENTS[baseImage.id] = generateDeploymentsForBaseImage(
        baseImage,
        MOCK_BASE_IMAGE_IMAGES[baseImage.id]
    );
});

export type {
    BaseImage,
    BaseImageCVE,
    BaseImageApplicationImage,
    BaseImageDeployment,
} from '../types';
