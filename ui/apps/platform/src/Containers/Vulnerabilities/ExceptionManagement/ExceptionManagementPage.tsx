import React from 'react';
import { PageSection } from '@patternfly/react-core';
import { Route, Switch } from 'react-router-dom';

import { exceptionManagementPath } from 'routePaths';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import ExceptionRequestsPage from './ExceptionRequestsPage';
import ExceptionRequestDetailsPage from './ExceptionRequestDetailsPage';

function ExceptionManagementPage() {
    return (
        <Switch>
            <Route
                path={`${exceptionManagementPath}/requests/:requestId`}
                component={ExceptionRequestDetailsPage}
            />
            <Route path={exceptionManagementPath} component={ExceptionRequestsPage} />
            <Route>
                <PageSection variant="light">
                    <PageTitle title="Exception requests - Not Found" />
                    <PageNotFound />
                </PageSection>
            </Route>
        </Switch>
    );
}

export default ExceptionManagementPage;
