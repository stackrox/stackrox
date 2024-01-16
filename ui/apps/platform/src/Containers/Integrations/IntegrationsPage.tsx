import React, { ReactElement } from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import useFeatureFlags from 'hooks/useFeatureFlags';
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
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isMoveInitBundlesEnabled = isFeatureFlagEnabled('ROX_MOVE_INIT_BUNDLES_UI');
    return (
        <Switch>
            <Route exact path={integrationsPath} component={IntegrationTilesPage} />
            {isMoveInitBundlesEnabled && (
                <Route
                    path={`${integrationsPath}/authProviders/clusterInitBundle/:action/:id`}
                    render={() => <Redirect to={clustersInitBundlesPath} />}
                />
            )}
            <Route exact path={integrationsListPath} component={IntegrationsListPage} />
            <Route path={integrationCreatePath} component={CreateIntegrationPage} />
            <Route path={integrationEditPath} component={EditIntegrationPage} />
            <Route path={integrationDetailsPath} component={IntegrationDetailsPage} />
            <Route component={IntegrationsNotFoundPage} />
        </Switch>
    );
};

export default Page;
