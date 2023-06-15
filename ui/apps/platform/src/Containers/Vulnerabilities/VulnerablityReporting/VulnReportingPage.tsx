import React from 'react';
import { Route, Switch } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

import { vulnerabilityReportingPath } from 'routePaths';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import VulnReportsPage from './VulnReports/VulnReportsPage';

import './VulnReportingPage.css';

function VulnReportingPage() {
    return (
        <Switch>
            <Route exact path={vulnerabilityReportingPath} component={VulnReportsPage} />
            <Route>
                <PageSection variant="light">
                    <PageTitle title="Vulnerability Reporting - Not Found" />
                    <PageNotFound />
                </PageSection>
            </Route>
        </Switch>
    );
}

export default VulnReportingPage;
