import React, { ReactElement } from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import {
    clustersInitBundlesPath,
    integrationsPath,
    integrationsListPath,
    integrationCreatePath,
    integrationEditPath,
    integrationDetailsPath,
} from 'routePaths';

import IntegrationsNotFoundPage from './IntegrationsNotFoundPage';
import IntegrationTilesPage from './IntegrationTiles/IntegrationTilesPage';
import IntegrationsListPage from './IntegrationsListPage';
import CreateIntegrationPage from './CreateIntegrationPage';
import EditIntegrationPage from './EditIntegrationPage';
import IntegrationDetailsPage from './IntegrationDetailsPage';

const Page = (): ReactElement => {
    // Redirect from list or view page to cluster init bundles list.
    return (
        <Switch>
            <Route exact path={integrationsPath}>
                <IntegrationTilesPage />
            </Route>
            <Route
                path={[
                    `${integrationsPath}/authProviders/clusterInitBundle`,
                    `${integrationsPath}/authProviders/clusterInitBundle/:action/:id`,
                ]}
                render={() => <Redirect to={clustersInitBundlesPath} />}
            />
            <Route exact path={integrationsListPath}>
                <IntegrationsListPage />
            </Route>
            <Route path={integrationCreatePath}>
                <CreateIntegrationPage />
            </Route>
            <Route path={integrationEditPath}>
                <EditIntegrationPage />
            </Route>
            <Route path={integrationDetailsPath}>
                <IntegrationDetailsPage />
            </Route>
            <Route>
                <IntegrationsNotFoundPage />
            </Route>
        </Switch>
    );
};

export default Page;
