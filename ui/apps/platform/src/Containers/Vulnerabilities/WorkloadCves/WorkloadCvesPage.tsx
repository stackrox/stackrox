import React from 'react';
import { Route, Switch } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';

import {
    vulnerabilitiesWorkloadCveDeploymentSinglePath,
    vulnerabilitiesWorkloadCveImageSinglePath,
    vulnerabilitiesWorkloadCveSinglePath,
    vulnerabilitiesWorkloadCvesPath,
} from 'routePaths';
import WorkloadCvesDeploymentSinglePage from './WorkloadCvesDeploymentSinglePage';
import WorkloadCvesImageSinglePage from './WorkloadCvesImageSinglePage';
import WorkloadCvesOverviewPage from './WorkloadCvesOverviewPage';
import WorkloadCvesSinglePage from './WorkloadCvesSinglePage';

function WorkloadCvesPage() {
    return (
        <Switch>
            <Route path={vulnerabilitiesWorkloadCveSinglePath} component={WorkloadCvesSinglePage} />
            <Route
                path={vulnerabilitiesWorkloadCveImageSinglePath}
                component={WorkloadCvesImageSinglePage}
            />
            <Route
                path={vulnerabilitiesWorkloadCveDeploymentSinglePath}
                component={WorkloadCvesDeploymentSinglePage}
            />
            <Route
                exact
                path={vulnerabilitiesWorkloadCvesPath}
                component={WorkloadCvesOverviewPage}
            />
            <Route>
                <PageSection variant="light">
                    <PageTitle title="Workload CVEs - Not Found" />
                    <PageNotFound />
                </PageSection>
            </Route>
        </Switch>
    );
}

export default WorkloadCvesPage;
