import React from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import {
    complianceEnhancedBasePath,
    complianceEnhancedClusterComplianceBasePath,
    // complianceEnhancedStatusPath,
} from 'routePaths';

// import ComplianceStatusPage from './Status/ComplianceStatusPage';
import ClusterCompliancePage from './ClusterCompliance/ClusterCompliancePage';

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
                <Redirect to={complianceEnhancedClusterComplianceBasePath} />
            </Route>
            {/*
            <Route path={complianceEnhancedStatusPath}>
                <ComplianceStatusPage />
            </Route>
            */}
            <Route path={complianceEnhancedClusterComplianceBasePath}>
                <ClusterCompliancePage />
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
