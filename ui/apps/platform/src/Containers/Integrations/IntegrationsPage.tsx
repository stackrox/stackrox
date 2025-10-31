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

import { getSourcesEnabled } from './utils/integrationsList';

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
    const sourcesEnabled = getSourcesEnabled({
        isCentralCapabilityAvailable,
        isFeatureFlagEnabled,
    });

    return (
        <Routes>
            <Route index element={<Navigate to={sourcesEnabled[0]} replace />} />
            {sourcesEnabled.map((source) => {
                const Element = integrationsTabElementMap[source];
                return Element ? (
                    <Route
                        key={source}
                        path={source}
                        element={<Element sourcesEnabled={sourcesEnabled} />}
                    />
                ) : null; // just in case
            })}
            <Route path=":source/:type" element={<IntegrationsListPage />} />
            <Route path=":source/:type/create" element={<CreateIntegrationPage />} />
            <Route path=":source/:type/edit/:id" element={<EditIntegrationPage />} />
            <Route path=":source/:type/view/:id" element={<IntegrationDetailsPage />} />
            <Route path="*" element={<IntegrationsNotFoundPage />} />
        </Routes>
    );
};

export default IntegrationsPage;
