import React, { useContext } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { Alert, Bullseye, Spinner } from '@patternfly/react-core';

import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import CheckDetailsPage from './CheckDetailsPage';
import ClusterDetailsPage from './ClusterDetailsPage';
import ComplianceNotFoundPage from '../ComplianceNotFoundPage';
import ComplianceProfilesProvider, {
    ComplianceProfilesContext,
} from './ComplianceProfilesProvider';
import CoverageEmptyState from './CoverageEmptyState';
import CoveragesPage from './CoveragesPage';
import ScanConfigurationsProvider from './ScanConfigurationsProvider';

function CoveragePage() {
    return (
        <ScanConfigurationsProvider>
            <ComplianceProfilesProvider>
                <CoverageContent />
            </ComplianceProfilesProvider>
        </ScanConfigurationsProvider>
    );
}

function CoverageContent() {
    const { scanConfigProfilesResponse, isLoading, error } = useContext(ComplianceProfilesContext);

    if (error) {
        return (
            <Alert variant="warning" title="Unable to fetch profiles" component="p" isInline>
                {getAxiosErrorMessage(error)}
            </Alert>
        );
    }

    if (!isLoading && scanConfigProfilesResponse.totalCount === 0) {
        return <CoverageEmptyState />;
    }

    return (
        <Routes>
            <Route index element={<ProfilesRedirectHandler />} />
            <Route path="profiles" element={<ProfilesRedirectHandler />} />
            <Route path="profiles/:profileName/checks/:checkName" element={<CheckDetailsPage />} />
            <Route
                path="profiles/:profileName/clusters/:clusterId"
                element={<ClusterDetailsPage />}
            />
            <Route path="profiles/:profileName/*" element={<CoveragesPage />} />
            <Route path="*" element={<ComplianceNotFoundPage />} />
        </Routes>
    );
}

function ProfilesRedirectHandler() {
    const { scanConfigProfilesResponse, isLoading } = useContext(ComplianceProfilesContext);
    const firstProfile = scanConfigProfilesResponse.profiles[0];

    if (isLoading) {
        return (
            <Bullseye>
                <Spinner />
            </Bullseye>
        );
    }

    return <Navigate to={`profiles/${firstProfile.name}/checks`} replace />;
}

export default CoveragePage;
