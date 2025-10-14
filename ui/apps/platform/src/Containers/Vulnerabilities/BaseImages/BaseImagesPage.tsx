import React from 'react';
import { Route, Routes } from 'react-router-dom-v5-compat';
import BaseImagesListPage from './BaseImagesListPage';
import BaseImageDetailPage from './BaseImageDetailPage';
import { BaseImagesProvider } from './hooks/useBaseImages';

/**
 * Base Images routing container with state provider
 */
function BaseImagesPage() {
    return (
        <BaseImagesProvider>
            <Routes>
                <Route path="/" element={<BaseImagesListPage />} />
                <Route path=":id" element={<BaseImageDetailPage />} />
            </Routes>
        </BaseImagesProvider>
    );
}

export default BaseImagesPage;
