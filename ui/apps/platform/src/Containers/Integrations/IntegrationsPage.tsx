import type { ReactElement } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom-v5-compat';

import useCentralCapabilities from 'hooks/useCentralCapabilities';
import useFeatureFlags from 'hooks/useFeatureFlags';
import type { IntegrationSource } from 'types/integration';

import IntegrationsNotFoundPage from './IntegrationsNotFoundPage';
import IntegrationsListPage from './IntegrationsListPage';
import CreateIntegrationPage from './CreateIntegrationPage';
import EditIntegrationPage from './EditIntegrationPage';
import IntegrationDetailsPage from './IntegrationDetailsPage';

import AuthenticationIntegrationsTab from './IntegrationTiles/AuthenticationIntegrationsTab';
import BackupIntegrationsTab from './IntegrationTiles/BackupIntegrationsTab';
import CloudSourceIntegrationsTab from './IntegrationTiles/CloudSourceIntegrationsTab';
import ImageIntegrationsTab from './IntegrationTiles/ImageIntegrationsTab';
import NotifierIntegrationsTab from './IntegrationTiles/NotifierIntegrationsTab';
import SignatureIntegrationsTab from './IntegrationTiles/SignatureIntegrationsTab';
import type { IntegrationsTabElement } from './IntegrationTiles/IntegrationsTab.types';

import { getSourcesEnabled, getTypesEnabled } from './utils/integrationsList';
import type { IntegrationsRoutePredicates } from './utils/integrationsList';

// Adapted from routeComponentMap from Body.tsx file.

const integrationsTabElementMap: Record<IntegrationSource, IntegrationsTabElement> = {
    imageIntegrations: ImageIntegrationsTab,
    signatureIntegrations: SignatureIntegrationsTab,
    notifiers: NotifierIntegrationsTab,
    backups: BackupIntegrationsTab,
    cloudSources: CloudSourceIntegrationsTab,
    authProviders: AuthenticationIntegrationsTab,
};

const IntegrationsPage = (): ReactElement => {
    const { isCentralCapabilityAvailable } = useCentralCapabilities();
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const predicates: IntegrationsRoutePredicates = {
        isCentralCapabilityAvailable,
        isFeatureFlagEnabled,
    };
    const sourcesEnabled = getSourcesEnabled(predicates);

    return (
        <Routes>
            <Route index element={<Navigate to={sourcesEnabled[0]} replace />} />
            {sourcesEnabled.flatMap((source) => {
                const Element = integrationsTabElementMap[source];
                const sourceRoute = Element ? (
                    <Route
                        key={source}
                        path={source}
                        element={<Element sourcesEnabled={sourcesEnabled} />}
                    />
                ) : null; // just in case
                const typeRoutes = getTypesEnabled(predicates, source).flatMap((type) => {
                    const pathSourceType = `${source}/${type}`;
                    const pathSourceTypeCreate = `${pathSourceType}/create`;
                    const pathSourceTypeEdit = `${pathSourceType}/edit/:id`;
                    const pathSourceTypeView = `${pathSourceType}/view/:id`;
                    return [
                        <Route
                            key={pathSourceType}
                            path={pathSourceType}
                            element={<IntegrationsListPage source={source} type={type} />}
                        />,
                        <Route
                            key={pathSourceTypeCreate}
                            path={pathSourceTypeCreate}
                            element={<CreateIntegrationPage source={source} type={type} />}
                        />,
                        <Route
                            key={pathSourceTypeEdit}
                            path={pathSourceTypeEdit}
                            element={<EditIntegrationPage source={source} type={type} />}
                        />,
                        <Route
                            key={pathSourceTypeView}
                            path={pathSourceTypeView}
                            element={<IntegrationDetailsPage source={source} type={type} />}
                        />,
                    ];
                });
                return [sourceRoute, ...typeRoutes];
            })}
            <Route path="*" element={<IntegrationsNotFoundPage />} />
        </Routes>
    );
};

export default IntegrationsPage;
