import React from 'react';
import { Route, Switch } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import ScannerV4IntegrationBanner from 'Components/ScannerV4IntegrationBanner';
import { vulnerabilitiesPlatformCvesPath } from 'routePaths';
import usePermissions from 'hooks/usePermissions';

import PlatformCvesOverviewPage from './Overview/PlatformCvesOverviewPage';
import PlatformCvePage from './PlatformCve/PlatformCvePage';
import ClusterPage from './Cluster/ClusterPage';

const vulnerabilitiesPlatformCveSinglePath = `${vulnerabilitiesPlatformCvesPath}/cves/:cveId`;
const vulnerabilitiesPlatformClusterSinglePath = `${vulnerabilitiesPlatformCvesPath}/clusters/:clusterId`;

function PlatformCvesPage() {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForIntegration = hasReadAccess('Integration');

    return (
        <>
            {hasReadAccessForIntegration && <ScannerV4IntegrationBanner />}
            <Switch>
                <Route path={vulnerabilitiesPlatformCveSinglePath} component={PlatformCvePage} />
                <Route path={vulnerabilitiesPlatformClusterSinglePath} component={ClusterPage} />
                <Route
                    exact
                    path={vulnerabilitiesPlatformCvesPath}
                    component={PlatformCvesOverviewPage}
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
