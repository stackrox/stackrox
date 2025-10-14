import React from 'react';
import { Route, Routes } from 'react-router-dom-v5-compat';
import BaseImagesListPage from './BaseImagesListPage';
import BaseImageDetailPage from './BaseImageDetailPage';
import BaseImagePrototypeTest from './BaseImagePrototypeTest';
import { BaseImagesProvider } from './hooks/useBaseImages';

/**
 * Base Images routing container with state provider
 */
function BaseImagesPage() {
    return (
        <BaseImagesProvider>
            <Routes>
                <Route index element={<BaseImagesListPage />} />
                <Route path="test" element={<BaseImagePrototypeTest />} />
                <Route path=":id" element={<BaseImageDetailPage />} />
            </Routes>
        </BaseImagesProvider>
    );
}

export default BaseImagesPage;
