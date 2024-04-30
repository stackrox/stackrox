import React from 'react';
import { Route, Switch } from 'react-router-dom';

import { complianceEnhancedCoveragePath } from 'routePaths';

import CoveragesPage from './CoveragesPage';

function CoveragePage() {
    /*
     * Examples of urls for CoveragePage:
     * /main/compliance-enhanced/cluster-compliance/coverage
     */

    return (
        <Switch>
            <Route exact path={complianceEnhancedCoveragePath}>
                <CoveragesPage />
            </Route>
        </Switch>
    );
}

export default CoveragePage;
