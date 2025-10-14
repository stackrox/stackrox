/**
 * Utility functions for determining base image layer information
 *
 * In production, this information would come from the backend API.
 * For the prototype, we provide utilities to help demonstrate the feature.
 */

import type { BaseImageInfo } from '../components/BaseImageInfoCard';

/**
 * Extract base image information from an image name
 * This is a simplified version for the prototype.
 * In production, this would come from the API.
 */
export function extractBaseImageInfo(imageName: string): BaseImageInfo | null {
    // Common base image patterns
    const baseImagePatterns = [
        /^(ubuntu):([0-9.]+)/i,
        /^(alpine):([0-9.]+)/i,
        /^(node):([0-9.]+)/i,
        /^(nginx):([0-9.]+)/i,
        /^(python):([0-9.]+)/i,
        /^(postgres):([0-9.]+)/i,
        /^(redis):([0-9.]+)/i,
        /^(mysql):([0-9.]+)/i,
    ];

    let baseImageInfo: BaseImageInfo | null = null;
    baseImagePatterns.some((pattern) => {
        const match = imageName.match(pattern);
        if (match) {
            baseImageInfo = {
                name: `${match[1]}:${match[2]}`,
                isTracked: false, // Will be updated based on tracking status
                baseImageId: undefined,
            };
            return true; // Stop iteration
        }
        return false;
    });

    if (baseImageInfo) {
        return baseImageInfo;
    }

    return null;
}

/**
 * Check if a base image is currently being tracked
 * This would query the tracking state in production
 */
export function isBaseImageTracked(baseImageName: string, trackedBaseImages: string[]): boolean {
    return trackedBaseImages.some(
        (tracked) => tracked.toLowerCase() === baseImageName.toLowerCase()
    );
}

/**
 * Get the ID for a tracked base image
 * In production, this would be returned from the API
 */
export function getBaseImageId(baseImageName: string): string | undefined {
    // For prototype: create a simple ID from the name
    return baseImageName.replace(/[:.]/g, '-');
}

/**
 * Determine if a component/CVE is from the base image layer
 * Based on comparing the component's layer index with the last base image layer index
 */
export function isFromBaseImageLayer(
    componentLayerIndex: number | undefined,
    lastBaseImageLayerIndex: number
): boolean {
    if (componentLayerIndex === undefined) {
        return false;
    }
    return componentLayerIndex <= lastBaseImageLayerIndex;
}

/**
 * Get a mock last base image layer index based on image name
 * In production, this would come from the image metadata
 *
 * For prototype: we'll estimate based on common patterns
 * - Base images typically have 3-7 layers
 * - Application layers are added on top
 */
export function estimateLastBaseImageLayerIndex(imageName: string): number {
    // Extract base image from name
    const baseImageInfo = extractBaseImageInfo(imageName);

    if (!baseImageInfo) {
        // If we can't identify a base image, assume first 3 layers
        return 2; // 0-indexed, so this means layers 0, 1, 2
    }

    // Estimate based on base image type
    const baseImageName = baseImageInfo.name.toLowerCase();

    if (baseImageName.startsWith('ubuntu')) {
        return 4; // Ubuntu typically has 5 layers (0-4)
    }
    if (baseImageName.startsWith('alpine')) {
        return 2; // Alpine is minimal, usually 3 layers (0-2)
    }
    if (baseImageName.startsWith('node')) {
        return 5; // Node builds on a base, usually 6 layers (0-5)
    }
    if (baseImageName.startsWith('nginx')) {
        return 4; // Nginx typically has 5 layers (0-4)
    }

    // Default estimate
    return 3; // 4 layers (0-3)
}
