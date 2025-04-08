import React, { ReactElement } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';

import { clustersInitBundlesPath } from 'routePaths';

import IntegrationsNotFoundPage from './IntegrationsNotFoundPage';
import IntegrationTilesPage from './IntegrationTiles/IntegrationTilesPage';
import IntegrationsListPage from './IntegrationsListPage';
import CreateIntegrationPage from './CreateIntegrationPage';
import EditIntegrationPage from './EditIntegrationPage';
import IntegrationDetailsPage from './IntegrationDetailsPage';

const IntegrationsPage = (): ReactElement => {
    // Redirect from list or view page to cluster init bundles list.
    return (
        <Routes>
            <Route index element={<IntegrationTilesPage />} />
            <Route
                path="authProviders/clusterInitBundle"
                element={<Navigate to={clustersInitBundlesPath} />}
            />
            <Route
                path="authProviders/clusterInitBundle/:action/:id"
                element={<Navigate to={clustersInitBundlesPath} />}
            />
            <Route path=":source/:type" element={<IntegrationsListPage />} />
            <Route path=":source/:type/create" element={<CreateIntegrationPage />} />
            <Route path=":source/:type/edit/:id" element={<EditIntegrationPage />} />
            <Route path=":source/:type/view/:id" element={<IntegrationDetailsPage />} />
            <Route path="*" element={<IntegrationsNotFoundPage />} />
        </Routes>
    );
};

export default IntegrationsPage;
