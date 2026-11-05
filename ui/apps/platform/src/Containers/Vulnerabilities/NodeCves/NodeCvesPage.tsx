import { Route, Routes } from 'react-router-dom-v5-compat';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import NodeCvesOverviewPage from './Overview/NodeCvesOverviewPage';
import NodeCvePage from './NodeCve/NodeCvePage';
import NodePage from './Node/NodePage';

function NodeCvesPage() {
    return (
        <>
            <Routes>
                <Route index element={<NodeCvesOverviewPage />} />
                <Route path="cves/:cveId" element={<NodeCvePage />} />
                <Route path="nodes/:nodeId" element={<NodePage />} />
                <Route
                    path="*"
                    element={
                        <PageSection hasBodyWrapper={false}>
                            <PageTitle title="Node CVEs - Not Found" />
                            <PageNotFound />
                        </PageSection>
                    }
                />
            </Routes>
        </>
    );
}

export default NodeCvesPage;
