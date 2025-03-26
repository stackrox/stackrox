import React from 'react';
import { Route, Routes } from 'react-router-dom';
import isEqual from 'lodash/isEqual';

import PageNotFound from 'Components/PageNotFound';

import DashboardPage from './Dashboard/WorkflowDashboardLayout';
import ListPage from './List/WorkflowListPageLayout';
import EntityPage from './Entity/WorkflowEntityPageLayout';

const vulnerabilitiesListPath = `:entityId1?/:entityType2?/:entityId2?`;
const vulnerabilitiesEntityPath = `:pageEntityId?/:entityType1?/:entityId1?/:entityType2?/:entityId2?`;

const Page = () => (
    <Routes>
        <Route index element={<DashboardPage />} />
        <Route path={`namespace/${vulnerabilitiesEntityPath}`} element={<EntityPage />} />
        <Route path={`cluster/${vulnerabilitiesEntityPath}`} element={<EntityPage />} />
        <Route path={`node/${vulnerabilitiesEntityPath}`} element={<EntityPage />} />
        <Route path={`deployment/${vulnerabilitiesEntityPath}`} element={<EntityPage />} />
        <Route path={`image/${vulnerabilitiesEntityPath}`} element={<EntityPage />} />
        <Route path={`image-cve/${vulnerabilitiesEntityPath}`} element={<EntityPage />} />
        <Route path={`node-cve/${vulnerabilitiesEntityPath}`} element={<EntityPage />} />
        <Route path={`cluster-cve/${vulnerabilitiesEntityPath}`} element={<EntityPage />} />
        <Route path={`node-component/${vulnerabilitiesEntityPath}`} element={<EntityPage />} />
        <Route path={`image-component/${vulnerabilitiesEntityPath}`} element={<EntityPage />} />

        <Route path={`namespaces/${vulnerabilitiesListPath}`} element={<ListPage />} />
        <Route path={`clusters/${vulnerabilitiesListPath}`} element={<ListPage />} />
        <Route path={`nodes/${vulnerabilitiesListPath}`} element={<ListPage />} />
        <Route path={`deployments/${vulnerabilitiesListPath}`} element={<ListPage />} />
        <Route path={`images/${vulnerabilitiesListPath}`} element={<ListPage />} />
        <Route path={`image-cves/${vulnerabilitiesListPath}`} element={<ListPage />} />
        <Route path={`node-cves/${vulnerabilitiesListPath}`} element={<ListPage />} />
        <Route path={`cluster-cves/${vulnerabilitiesListPath}`} element={<ListPage />} />
        <Route path={`node-components/${vulnerabilitiesListPath}`} element={<ListPage />} />
        <Route path={`image-components/${vulnerabilitiesListPath}`} element={<ListPage />} />
        <Route path="*" element={<PageNotFound />} />
    </Routes>
);

export default React.memo(Page, isEqual);
