import React, { useContext } from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';
import { Alert, Bullseye, Spinner } from '@patternfly/react-core';

import { complianceEnhancedCoveragePath } from 'routePaths';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import {
    coverageCheckDetailsPath,
    coverageClusterDetailsPath,
    coverageProfileChecksPath,
    coverageProfileClustersPath,
} from './compliance.coverage.routes';
import CheckDetailsPage from './CheckDetailsPage';
import ClusterDetailsPage from './ClusterDetailsPage';
import ComplianceProfilesProvider, {
    ComplianceProfilesContext,
} from './ComplianceProfilesProvider';
import CoverageEmptyState from './CoverageEmptyState';
import ProfileChecksPage from './ProfileChecksPage';
import ProfileClustersPage from './ProfileClustersPage';
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
            <Alert variant="warning" title="Unable to fetch profiles" component="div" isInline>
                {getAxiosErrorMessage(error)}
            </Alert>
        );
    }

    if (!isLoading && scanConfigProfilesResponse.totalCount === 0) {
        return <CoverageEmptyState />;
    }

    return (
        <Switch>
            <Route exact path={coverageProfileChecksPath} component={ProfileChecksPage} />
            <Route exact path={coverageProfileClustersPath} component={ProfileClustersPage} />
            <Route exact path={coverageCheckDetailsPath} component={CheckDetailsPage} />
            <Route exact path={coverageClusterDetailsPath} component={ClusterDetailsPage} />
            <Route
                exact
                path={[
                    `${complianceEnhancedCoveragePath}`,
                    `${complianceEnhancedCoveragePath}/profiles`,
                ]}
                component={ProfilesRedirectHandler}
            />
        </Switch>
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

    return (
        <Redirect to={`${complianceEnhancedCoveragePath}/profiles/${firstProfile.name}/checks`} />
    );
}

export default CoveragePage;
