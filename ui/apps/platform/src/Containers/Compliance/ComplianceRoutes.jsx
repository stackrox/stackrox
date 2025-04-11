import React from 'react';
import { Route, Routes } from 'react-router-dom';
import isEqual from 'lodash/isEqual';

import PageNotFound from 'Components/PageNotFound';
import searchContext from 'Containers/searchContext';

import ComplianceDashboardPage from './Dashboard/ComplianceDashboardPage';
import EntityPage from './Entity/EntityPage';
import ListPage from './List/ListPage';

const complianceListPath = `:entityId1?/:entityType2?/:entityId2?`;
const complianceEntityPath = `:pageEntityId?/:entityType1?/:entityId1?/:entityType2?/:entityId2?`;

const Page = () => (
    <searchContext.Provider value="s">
        <Routes>
            <Route index element={<ComplianceDashboardPage />} />
            <Route path={`clusters/${complianceListPath}`} element={<ListPage />} />
            <Route path={`controls/${complianceListPath}`} element={<ListPage />} />
            <Route path={`deployments/${complianceListPath}`} element={<ListPage />} />
            <Route path={`namespaces/${complianceListPath}`} element={<ListPage />} />
            <Route path={`nodes/${complianceListPath}`} element={<ListPage />} />

            <Route path={`cluster/${complianceEntityPath}`} element={<EntityPage />} />
            <Route path={`control/${complianceEntityPath}`} element={<EntityPage />} />
            <Route path={`deployment/${complianceEntityPath}`} element={<EntityPage />} />
            <Route path={`namespace/${complianceEntityPath}`} element={<EntityPage />} />
            <Route path={`node/${complianceEntityPath}`} element={<EntityPage />} />
            <Route path={`standard/${complianceEntityPath}`} element={<EntityPage />} />

            <Route path="*" element={<PageNotFound useCase="compliance" />} />
        </Routes>
    </searchContext.Provider>
);

export default React.memo(Page, isEqual);
