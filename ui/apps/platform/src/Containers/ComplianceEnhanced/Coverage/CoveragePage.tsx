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
import ProfileChecksPage from './ProfileChecksPage';
import ProfileClustersPage from './ProfileClustersPage';

function CoveragePage() {
    return (
        <ComplianceProfilesProvider>
            <CoverageContent />
        </ComplianceProfilesProvider>
    );
}

function CoverageContent() {
    const { profileScanStats, isLoading, error } = useContext(ComplianceProfilesContext);

    if (isLoading) {
        return (
            <Bullseye>
                <Spinner />
            </Bullseye>
        );
    }

    if (error) {
        return (
            <Alert variant="warning" title="Unable to fetch profiles" component="div" isInline>
                {getAxiosErrorMessage(error)}
            </Alert>
        );
    }

    if (profileScanStats?.scanStats.length === 0) {
        // TODO: Add a message for when there are no profiles
        return <div>No profiles, create a scan schedule</div>;
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
    const { profileScanStats } = useContext(ComplianceProfilesContext);
    const firstProfile = profileScanStats.scanStats[0];

    return (
        <Redirect
            to={`${complianceEnhancedCoveragePath}/profiles/${firstProfile.profileName}/checks`}
        />
    );
}

export default CoveragePage;
