import React, { ReactElement } from 'react';
import { Route, Switch } from 'react-router-dom';

import {
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
import usePermissions from '../../hooks/usePermissions';
import IntegrationsNoPermission from './IntegrationsNoPermission';

const Page = (): ReactElement => {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForIntegrations = hasReadAccess('Integration');
    return (
        <>
            {hasReadAccessForIntegrations ? (
                <Switch>
                    <Route exact path={integrationsPath} component={IntegrationTilesPage} />
                    <Route exact path={integrationsListPath} component={IntegrationsListPage} />
                    <Route path={integrationCreatePath} component={CreateIntegrationPage} />
                    <Route path={integrationEditPath} component={EditIntegrationPage} />
                    <Route path={integrationDetailsPath} component={IntegrationDetailsPage} />
                    <Route component={IntegrationsNotFoundPage} />
                </Switch>
            ) : (
                <IntegrationsNoPermission />
            )}
        </>
    );
};

export default Page;
