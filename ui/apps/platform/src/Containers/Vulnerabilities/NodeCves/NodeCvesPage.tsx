import React from 'react';
import { Route, Switch } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import ScannerV4IntegrationBanner from 'Components/ScannerV4IntegrationBanner';
import { vulnerabilitiesNodeCvesPath } from 'routePaths';
import usePermissions from 'hooks/usePermissions';
import NodeCvesOverviewPage from './Overview/NodeCvesOverviewPage';
import NodeCvePage from './NodeCve/NodeCvePage';

const vulnerabilitiesNodeCveSinglePath = `${vulnerabilitiesNodeCvesPath}/cves/:cveId`;

function NodeCvesPage() {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForIntegration = hasReadAccess('Integration');

    return (
        <>
            {hasReadAccessForIntegration && <ScannerV4IntegrationBanner />}
            <Switch>
                <Route path={vulnerabilitiesNodeCveSinglePath} component={NodeCvePage} />
                <Route exact path={vulnerabilitiesNodeCvesPath} component={NodeCvesOverviewPage} />
                <Route>
                    <PageSection variant="light">
                        <PageTitle title="Node CVEs - Not Found" />
                        <PageNotFound />
                    </PageSection>
                </Route>
            </Switch>
        </>
    );
}

export default NodeCvesPage;
