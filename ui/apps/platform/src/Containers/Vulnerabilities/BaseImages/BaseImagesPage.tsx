import React from 'react';
import { Route, Routes } from 'react-router-dom-v5-compat';
import BaseImagesListPage from './BaseImagesListPage';
import BaseImageDetailPage from './BaseImageDetailPage';

/**
 * Base Images routing container
 */
function BaseImagesPage() {
    return (
        <Routes>
            <Route path="/" element={<BaseImagesListPage />} />
            <Route path=":id" element={<BaseImageDetailPage />} />
        </Routes>
    );
}

export default BaseImagesPage;
