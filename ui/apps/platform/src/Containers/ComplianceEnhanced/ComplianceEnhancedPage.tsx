import React from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import {
    complianceEnhancedBasePath,
    complianceEnhancedCoveragePath,
    complianceEnhancedSchedulesPath,
    // complianceEnhancedStatusPath,
} from 'routePaths';

// import ComplianceStatusPage from './Status/ComplianceStatusPage';
import CoveragePage from './Coverage/CoveragePage';
import ScanConfigsPage from './Schedules/ScanConfigsPage';

function ComplianceEnhancedPage() {
    // For 4.5 release:
    // 1. Redirect
    //    replace complianceEnhancedClusterComplianceBasePath
    //    with complianceEnhancedStatusPath
    // 2. Route for complianceEnhancedStatusPath
    //    uncomment
    return (
        <Switch>
            <Route exact path={complianceEnhancedBasePath}>
                <Redirect to={complianceEnhancedCoveragePath} />
            </Route>
            {/*
            <Route path={complianceEnhancedStatusPath}>
                <ComplianceStatusPage />
            </Route>
            */}
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
