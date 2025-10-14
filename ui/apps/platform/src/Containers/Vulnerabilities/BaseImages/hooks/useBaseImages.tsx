import React, { createContext, useContext, useState, useCallback, ReactNode } from 'react';
import { BaseImage, ScanningStatus } from '../types';
import { MOCK_BASE_IMAGES } from '../mockData';

type BaseImagesContextType = {
    baseImages: BaseImage[];
    addBaseImage: (name: string) => void;
    removeBaseImage: (id: string) => void;
    updateBaseImageStatus: (id: string, status: ScanningStatus) => void;
    getBaseImageById: (id: string) => BaseImage | undefined;
};

const BaseImagesContext = createContext<BaseImagesContextType | undefined>(undefined);

export function BaseImagesProvider({ children }: { children: ReactNode }) {
    const [baseImages, setBaseImages] = useState<BaseImage[]>(MOCK_BASE_IMAGES);

    const addBaseImage = useCallback((name: string) => {
        // Generate realistic counts based on base image name
        const generateCounts = (baseName: string) => {
            const lowerName = baseName.toLowerCase();

            // Different base images have different typical CVE profiles
            if (lowerName.includes('ubuntu')) {
                return {
                    cveCount: {
                        critical: Math.floor(Math.random() * 5) + 3,
                        high: Math.floor(Math.random() * 15) + 8,
                        medium: Math.floor(Math.random() * 20) + 15,
                        low: Math.floor(Math.random() * 10) + 5,
                        total: 0, // Will be calculated
                    },
                    imageCount: Math.floor(Math.random() * 10) + 5,
                    deploymentCount: Math.floor(Math.random() * 15) + 8,
                    lastBaseLayerIndex: 4,
                };
            }
            if (lowerName.includes('alpine')) {
                return {
                    cveCount: {
                        critical: Math.floor(Math.random() * 2) + 1,
                        high: Math.floor(Math.random() * 5) + 2,
                        medium: Math.floor(Math.random() * 8) + 3,
                        low: Math.floor(Math.random() * 5) + 2,
                        total: 0,
                    },
                    imageCount: Math.floor(Math.random() * 8) + 3,
                    deploymentCount: Math.floor(Math.random() * 12) + 5,
                    lastBaseLayerIndex: 2,
                };
            }
            if (lowerName.includes('node')) {
                return {
                    cveCount: {
                        critical: Math.floor(Math.random() * 4) + 2,
                        high: Math.floor(Math.random() * 12) + 6,
                        medium: Math.floor(Math.random() * 18) + 10,
                        low: Math.floor(Math.random() * 8) + 4,
                        total: 0,
                    },
                    imageCount: Math.floor(Math.random() * 12) + 6,
                    deploymentCount: Math.floor(Math.random() * 18) + 10,
                    lastBaseLayerIndex: 5,
                };
            }

            // Default for other base images
            return {
                cveCount: {
                    critical: Math.floor(Math.random() * 3) + 1,
                    high: Math.floor(Math.random() * 10) + 5,
                    medium: Math.floor(Math.random() * 15) + 8,
                    low: Math.floor(Math.random() * 8) + 3,
                    total: 0,
                },
                imageCount: Math.floor(Math.random() * 8) + 4,
                deploymentCount: Math.floor(Math.random() * 12) + 6,
                lastBaseLayerIndex: 3,
            };
        };

        const counts = generateCounts(name);
        const totalCves =
            counts.cveCount.critical +
            counts.cveCount.high +
            counts.cveCount.medium +
            counts.cveCount.low;

        const newBaseImage: BaseImage = {
            id: `base-image-${Date.now()}`,
            name,
            normalizedName: `docker.io/library/${name}`,
            scanningStatus: 'IN_PROGRESS',
            lastScanned: null,
            createdAt: new Date().toISOString(),
            cveCount: {
                critical: 0,
                high: 0,
                medium: 0,
                low: 0,
                total: 0,
            },
            imageCount: 0,
            deploymentCount: 0,
            lastBaseLayerIndex: counts.lastBaseLayerIndex,
        };

        setBaseImages((prev) => [newBaseImage, ...prev]);

        // Simulate scan completion after 5 seconds with realistic counts
        setTimeout(() => {
            setBaseImages((prev) =>
                prev.map((img) =>
                    img.id === newBaseImage.id
                        ? {
                              ...img,
                              scanningStatus: 'COMPLETED',
                              lastScanned: new Date().toISOString(),
                              cveCount: {
                                  ...counts.cveCount,
                                  total: totalCves,
                              },
                              imageCount: counts.imageCount,
                              deploymentCount: counts.deploymentCount,
                          }
                        : img
                )
            );
        }, 5000);
    }, []);

    const removeBaseImage = useCallback((id: string) => {
        setBaseImages((prev) => prev.filter((img) => img.id !== id));
    }, []);

    const updateBaseImageStatus = useCallback((id: string, status: ScanningStatus) => {
        setBaseImages((prev) =>
            prev.map((img) => (img.id === id ? { ...img, scanningStatus: status } : img))
        );
    }, []);

    const getBaseImageById = useCallback(
        (id: string) => {
            return baseImages.find((img) => img.id === id);
        },
        [baseImages]
    );

    return (
        <BaseImagesContext.Provider
            value={{
                baseImages,
                addBaseImage,
                removeBaseImage,
                updateBaseImageStatus,
                getBaseImageById,
            }}
        >
            {children}
        </BaseImagesContext.Provider>
    );
}

export function useBaseImages() {
    const context = useContext(BaseImagesContext);
    if (!context) {
        throw new Error('useBaseImages must be used within a BaseImagesProvider');
    }
    return context;
}
