import React from 'react';
import { Route, Routes } from 'react-router-dom';

import ExceptionRequestsPage from './ExceptionRequestsPage';
import ExceptionRequestDetailsPage from './ExceptionRequestDetailsPage';

function ExceptionManagementPage() {
    return (
        <Routes>
            <Route path="requests/:requestId" element={<ExceptionRequestDetailsPage />} />
            <Route path="*" element={<ExceptionRequestsPage />} />
        </Routes>
    );
}

export default ExceptionManagementPage;
