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
import DeploymentPage from './DeploymentPage';
import ImagePage from './ImagePage';
import WorkloadCvesOverviewPage from './Overview/WorkloadCvesOverviewPage';
import ImageCvePage from './ImageCvePage';

function WorkloadCvesPage() {
    return (
        <Switch>
            <Route path={vulnerabilitiesWorkloadCveSinglePath} component={ImageCvePage} />
            <Route path={vulnerabilitiesWorkloadCveImageSinglePath} component={ImagePage} />
            <Route
                path={vulnerabilitiesWorkloadCveDeploymentSinglePath}
                component={DeploymentPage}
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
