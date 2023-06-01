import React from 'react';
import { Link, Route, Switch } from 'react-router-dom';
import { Alert, PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';

import {
    vulnManagementPath,
    vulnerabilitiesWorkloadCveDeploymentSinglePath,
    vulnerabilitiesWorkloadCveImageSinglePath,
    vulnerabilitiesWorkloadCveSinglePath,
    vulnerabilitiesWorkloadCvesPath,
} from 'routePaths';
import DeploymentPage from './Deployment/DeploymentPage';
import ImagePage from './Image/ImagePage';
import WorkloadCvesOverviewPage from './Overview/WorkloadCvesOverviewPage';
import ImageCvePage from './ImageCve/ImageCvePage';

import './WorkloadCvesPage.css';

function WorkloadCvesPage() {
    return (
        <>
            <Alert
                variant="warning"
                isInline
                title={
                    <span>
                        This is a Technology Preview of this feature. For all production
                        requirements we recommend using{' '}
                        <Link to={vulnManagementPath}>Vulnerability Management (1.0)</Link>
                    </span>
                }
            />
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
        </>
    );
}

export default WorkloadCvesPage;
