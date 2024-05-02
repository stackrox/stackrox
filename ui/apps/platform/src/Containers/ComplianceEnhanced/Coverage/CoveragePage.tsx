import React, { useContext } from 'react';
import { Redirect, Route, Switch, useParams } from 'react-router-dom';

import { complianceEnhancedCoveragePath } from 'routePaths';

import ProfileChecksPage from './ProfileChecksPage';
import CheckDetailsPage from './CheckDetailsPage';
import CoveragesPage from './CoveragesPage';
import ProfileClustersPage from './ProfileClustersPage';
import ComplianceProfilesProvider, {
    ComplianceProfilesContext,
} from './ComplianceProfilesProvider';

function CoveragePage() {
    /*
     * Examples of urls for CoveragePage:
     * /main/compliance-enhanced/cluster-compliance/coverage
     */

    return (
        <Switch>
            <Route
                exact
                path={`${complianceEnhancedCoveragePath}/profiles/:profileName/checks/:checkName`}
                component={CheckDetailsPage}
            />
            <Route path={`${complianceEnhancedCoveragePath}`}>
                <ComplianceProfilesProvider>
                    <CoveragesPage>
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
                                path={[
                                    `${complianceEnhancedCoveragePath}`,
                                    `${complianceEnhancedCoveragePath}/profiles`,
                                    `${complianceEnhancedCoveragePath}/profiles/:profileName`,
                                ]}
                                component={CoverageRedirectHandler}
                            />
                        </Switch>
                    </CoveragesPage>
                </ComplianceProfilesProvider>
            </Route>
        </Switch>
    );
}

function CoverageRedirectHandler() {
    const context = useContext(ComplianceProfilesContext);
    const { profileName } = useParams();

    if (!context) {
        return null;
    }

    const { profileScanStats } = context;

    if (!profileScanStats || profileScanStats.scanStats.length === 0) {
        return null;
    }

    if (profileName) {
        const profileParamExists = profileScanStats.scanStats.some(
            (profile) => profile.profileName === profileName
        );
        if (profileParamExists) {
            return (
                <Redirect to={`${complianceEnhancedCoveragePath}/profiles/${profileName}/checks`} />
            );
        }
        return <>No results found for {profileName}</>;
    }

    const firstProfile = profileScanStats.scanStats[0];

    return (
        <Redirect
            to={`${complianceEnhancedCoveragePath}/profiles/${firstProfile.profileName}/checks`}
        />
    );
}

export default CoveragePage;
