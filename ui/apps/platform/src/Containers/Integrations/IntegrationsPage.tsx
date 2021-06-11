import React, { ReactElement } from 'react';
import { Route, Switch } from 'react-router-dom';

import { integrationsPath } from 'routePaths';

import IntegrationTilesPage from './IntegrationTilesPage';
import IntegrationsNotFoundPage from './IntegrationsNotFoundPage';

const integrationsListPath = `${integrationsPath}/:source/:type`;

// @TODO: As part of the UI/UX redesign of integrations page, we will remove the modal and have
// a separate view for the list and form. For now we'll have both of these use the same component
const Page = (): ReactElement => (
    <Switch>
        <Route exact path={integrationsPath} component={IntegrationTilesPage} />
        <Route path={integrationsListPath} component={IntegrationTilesPage} />
        <Route component={IntegrationsNotFoundPage} />
    </Switch>
);

export default Page;
