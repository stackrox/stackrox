import React, { createContext, useContext, useState, useCallback, ReactNode } from 'react';
import { BaseImage, ScanningStatus } from '../types';
import { MOCK_BASE_IMAGES } from '../mockData';

type BaseImagesContextType = {
    baseImages: BaseImage[];
    addBaseImage: (name: string) => void;
    removeBaseImage: (id: string) => void;
    updateBaseImageStatus: (id: string, status: ScanningStatus) => void;
};

const BaseImagesContext = createContext<BaseImagesContextType | undefined>(undefined);

export function BaseImagesProvider({ children }: { children: ReactNode }) {
    const [baseImages, setBaseImages] = useState<BaseImage[]>(MOCK_BASE_IMAGES);

    const addBaseImage = useCallback((name: string) => {
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
            lastBaseLayerIndex: 0,
        };

        setBaseImages((prev) => [newBaseImage, ...prev]);

        // Simulate scan completion after 2 seconds
        setTimeout(() => {
            setBaseImages((prev) =>
                prev.map((img) =>
                    img.id === newBaseImage.id
                        ? {
                              ...img,
                              scanningStatus: 'COMPLETED',
                              lastScanned: new Date().toISOString(),
                          }
                        : img
                )
            );
        }, 2000);
    }, []);

    const removeBaseImage = useCallback((id: string) => {
        setBaseImages((prev) => prev.filter((img) => img.id !== id));
    }, []);

    const updateBaseImageStatus = useCallback((id: string, status: ScanningStatus) => {
        setBaseImages((prev) =>
            prev.map((img) => (img.id === id ? { ...img, scanningStatus: status } : img))
        );
    }, []);

    return (
        <BaseImagesContext.Provider
            value={{ baseImages, addBaseImage, removeBaseImage, updateBaseImageStatus }}
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
