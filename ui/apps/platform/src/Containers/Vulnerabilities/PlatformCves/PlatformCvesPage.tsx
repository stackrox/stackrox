import React from 'react';
import { Route, Switch } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import ScannerV4IntegrationBanner from 'Components/ScannerV4IntegrationBanner';
import { vulnerabilitiesPlatformCvesPath } from 'routePaths';
import usePermissions from 'hooks/usePermissions';

function TmpPlatformCvesOverviewPage() {
    return <div>PlatformCvesOverviewPage</div>;
}

function PlatformCvesPage() {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForIntegration = hasReadAccess('Integration');

    return (
        <>
            {hasReadAccessForIntegration && <ScannerV4IntegrationBanner />}
            <Switch>
                <Route
                    exact
                    path={vulnerabilitiesPlatformCvesPath}
                    component={TmpPlatformCvesOverviewPage}
                />
                <Route>
                    <PageSection variant="light">
                        <PageTitle title="Platform CVEs - Not Found" />
                        <PageNotFound />
                    </PageSection>
                </Route>
            </Switch>
        </>
    );
}

export default PlatformCvesPage;
