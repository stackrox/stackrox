import React, { useContext } from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import { complianceEnhancedCoveragePath } from 'routePaths';

import CheckDetailsPage from './CheckDetailsPage';
import ClusterDetailsPage from './ClusterDetailsPage';
import ProfileChecksPage from './ProfileChecksPage';
import ProfileClustersPage from './ProfileClustersPage';
import ComplianceProfilesProvider, {
    ComplianceProfilesContext,
} from './ComplianceProfilesProvider';

function CoveragePage() {
    return (
        <ComplianceProfilesProvider>
            <Switch>
                <Route
                    exact
                    path={`${complianceEnhancedCoveragePath}/profiles/:profileName/checks`}
                    component={ProfileChecksPage}
                />
                <Route
                    exact
                    path={`${complianceEnhancedCoveragePath}/profiles/:profileName/clusters`}
                    component={ProfileClustersPage}
                />
                <Route
                    exact
                    path={`${complianceEnhancedCoveragePath}/profiles/:profileName/checks/:checkName`}
                    component={CheckDetailsPage}
                />
                <Route
                    exact
                    path={`${complianceEnhancedCoveragePath}/profiles/:profileName/clusters/:clusterName`}
                    component={ClusterDetailsPage}
                />
                <Route
                    exact
                    path={[
                        `${complianceEnhancedCoveragePath}`,
                        `${complianceEnhancedCoveragePath}/profiles`,
                    ]}
                    component={ProfilesRedirectHandler}
                />
            </Switch>
        </ComplianceProfilesProvider>
    );
}

function ProfilesRedirectHandler() {
    const profileScanStats = useContext(ComplianceProfilesContext);
    const firstProfile = profileScanStats.scanStats[0];

    return (
        <Redirect
            to={`${complianceEnhancedCoveragePath}/profiles/${firstProfile.profileName}/checks`}
        />
    );
}

export default CoveragePage;
