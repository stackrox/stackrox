import React from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import {
    complianceEnhancedBasePath,
    complianceEnhancedCoveragePath,
    complianceEnhancedSchedulesPath,
} from 'routePaths';

import CoveragePage from './Coverage/CoveragePage';
import ScanConfigsPage from './Schedules/ScanConfigsPage';

function ComplianceEnhancedPage() {
    return (
        <Switch>
            <Route
                exact
                path={complianceEnhancedBasePath}
                render={() => <Redirect to={complianceEnhancedCoveragePath} />}
            />
            <Route path={complianceEnhancedCoveragePath}>
                <CoveragePage />
            </Route>
            <Route path={complianceEnhancedSchedulesPath}>
                <ScanConfigsPage />
            </Route>
            <Route>
                <PageSection variant="light">
                    <PageTitle title="Compliance - Not Found" />
                    <PageNotFound />
                </PageSection>
            </Route>
        </Switch>
    );
}

export default ComplianceEnhancedPage;
