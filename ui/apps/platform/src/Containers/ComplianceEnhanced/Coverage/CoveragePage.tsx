import React from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import { complianceEnhancedCoveragePath } from 'routePaths';

import CheckDetailsPage from './CheckDetailsPage';
import CoveragesPage from './CoveragesPage';

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
            <Route
                exact
                path={[
                    `${complianceEnhancedCoveragePath}/profiles`,
                    `${complianceEnhancedCoveragePath}/profiles/:profileName/checks`,
                    `${complianceEnhancedCoveragePath}/profiles/:profileName/clusters`,
                ]}
                component={CoveragesPage}
            />
            <Redirect
                exact
                from={`${complianceEnhancedCoveragePath}`}
                to={`${complianceEnhancedCoveragePath}/profiles`}
            />
            <Redirect
                exact
                from={`${complianceEnhancedCoveragePath}/profiles/:profileName`}
                to={`${complianceEnhancedCoveragePath}/profiles/:profileName/checks`}
            />
        </Switch>
    );
}

export default CoveragePage;
