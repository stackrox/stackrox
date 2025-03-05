import React from 'react';
import { Route, Routes } from 'react-router-dom';
import isEqual from 'lodash/isEqual';

import PageNotFound from 'Components/PageNotFound';
import searchContext from 'Containers/searchContext';

import Dashboard from './Dashboard/ComplianceDashboardPage';
import Entity from './Entity/Page';
import List from './List/Page';

const complianceListPath = `:entityId1?/:entityType2?/:entityId2?`;
const complianceEntityPath = `:pageEntityId?/:entityType1?/:entityId1?/:entityType2?/:entityId2?`;

const Page = () => (
    <searchContext.Provider value="s">
        <Routes>
            <Route index element={<Dashboard />} />
            <Route path={`clusters/${complianceListPath}`} element={<List />} />
            <Route path={`controls/${complianceListPath}`} element={<List />} />
            <Route path={`deployments/${complianceListPath}`} element={<List />} />
            <Route path={`namespaces/${complianceListPath}`} element={<List />} />
            <Route path={`nodes/${complianceListPath}`} element={<List />} />

            <Route path={`cluster/${complianceEntityPath}`} element={<Entity />} />
            <Route path={`control/${complianceEntityPath}`} element={<Entity />} />
            <Route path={`deployment/${complianceEntityPath}`} element={<Entity />} />
            <Route path={`namespace/${complianceEntityPath}`} element={<Entity />} />
            <Route path={`node/${complianceEntityPath}`} element={<Entity />} />
            <Route path={`standard/${complianceEntityPath}`} element={<Entity />} />

            <Route path="*" element={<PageNotFound useCase="compliance" />} />
        </Routes>
    </searchContext.Provider>
);

export default React.memo(Page, isEqual);
