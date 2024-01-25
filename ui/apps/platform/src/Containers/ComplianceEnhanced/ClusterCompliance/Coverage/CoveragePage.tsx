import React from 'react';
import { Route, Switch } from 'react-router-dom';

import { complianceEnhancedCoveragePath, complianceEnhancedCoverageClustersPath } from 'routePaths';

import CoveragesPage from './CoveragesPage';
import ClusterDetails from './Clusters/ClusterDetails';

function CoveragePage() {
    /*
     * Examples of urls for CoveragePage:
     * /main/compliance-enhanced/cluster-compliance/coverage
     * /main/compliance-enhanced/cluster-compliance/coverage/clusters/:clusterId
     */

    return (
        <Switch>
            <Route exact path={complianceEnhancedCoveragePath}>
                <CoveragesPage />
            </Route>
            <Route path={complianceEnhancedCoverageClustersPath}>
                <ClusterDetails />
            </Route>
        </Switch>
    );
}

export default CoveragePage;
