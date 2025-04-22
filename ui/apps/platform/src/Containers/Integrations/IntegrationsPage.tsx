import React, { ReactElement } from 'react';
import { Route, Routes } from 'react-router-dom';

import IntegrationsNotFoundPage from './IntegrationsNotFoundPage';
import IntegrationTilesPage from './IntegrationTiles/IntegrationTilesPage';
import IntegrationsListPage from './IntegrationsListPage';
import CreateIntegrationPage from './CreateIntegrationPage';
import EditIntegrationPage from './EditIntegrationPage';
import IntegrationDetailsPage from './IntegrationDetailsPage';

const IntegrationsPage = (): ReactElement => {
    return (
        <Routes>
            <Route index element={<IntegrationTilesPage />} />
            <Route path=":source/:type" element={<IntegrationsListPage />} />
            <Route path=":source/:type/create" element={<CreateIntegrationPage />} />
            <Route path=":source/:type/edit/:id" element={<EditIntegrationPage />} />
            <Route path=":source/:type/view/:id" element={<IntegrationDetailsPage />} />
            <Route path="*" element={<IntegrationsNotFoundPage />} />
        </Routes>
    );
};

export default IntegrationsPage;
