import React, { ReactElement } from 'react';
import { Route, Routes } from 'react-router-dom';

import ViolationsTablePage from './ViolationsTablePage';
import ViolationDetailsPage from './Details/ViolationDetailsPage';
import ViolationNotFoundPage from './ViolationNotFoundPage';

function ViolationsPage(): ReactElement {
    return (
        <Routes>
            <Route index element={<ViolationsTablePage />} />
            <Route path=":alertId" element={<ViolationDetailsPage />} />
            <Route path="*" element={<ViolationNotFoundPage />} />
        </Routes>
    );
}

export default ViolationsPage;
