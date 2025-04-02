import React from 'react';
import { Route, Routes } from 'react-router-dom';
import isEqual from 'lodash/isEqual';
import PageNotFound from 'Components/PageNotFound';
import searchContext from 'Containers/searchContext';
import { searchParams } from 'constants/searchParams';
import useCases from 'constants/useCaseTypes';

import DashboardPage from './Dashboard/Page';
import ListPage from './List/Page';
import EntityPage from './Entity/Page';

const listPath = `:entityId1?/:entityType2?/:entityId2?`;
const entityPath = `:pageEntityId?/:entityType1?/:entityId1?/:entityType2?/:entityId2?`;

const Page = () => (
    <searchContext.Provider value={searchParams.page}>
        <Routes>
            <Route index element={<DashboardPage />} />
            <Route path={`namespace/${entityPath}`} element={<EntityPage />} />
            <Route path={`cluster/${entityPath}`} element={<EntityPage />} />
            <Route path={`node/${entityPath}`} element={<EntityPage />} />
            <Route path={`deployment/${entityPath}`} element={<EntityPage />} />
            <Route path={`image/${entityPath}`} element={<EntityPage />} />
            <Route path={`secret/${entityPath}`} element={<EntityPage />} />
            <Route path={`policy/${entityPath}`} element={<EntityPage />} />
            <Route path={`control/${entityPath}`} element={<EntityPage />} />
            <Route path={`serviceaccount/${entityPath}`} element={<EntityPage />} />
            <Route path={`subject/${entityPath}`} element={<EntityPage />} />
            <Route path={`role/${entityPath}`} element={<EntityPage />} />

            <Route path={`namespaces/${listPath}`} element={<ListPage />} />
            <Route path={`clusters/${listPath}`} element={<ListPage />} />
            <Route path={`nodes/${listPath}`} element={<ListPage />} />
            <Route path={`deployments/${listPath}`} element={<ListPage />} />
            <Route path={`images/${listPath}`} element={<ListPage />} />
            <Route path={`secrets/${listPath}`} element={<ListPage />} />
            <Route path={`policies/${listPath}`} element={<ListPage />} />
            <Route path={`controls/${listPath}`} element={<ListPage />} />
            <Route path={`serviceaccounts/${listPath}`} element={<ListPage />} />
            <Route path={`subjects/${listPath}`} element={<ListPage />} />
            <Route path={`roles/${listPath}`} element={<ListPage />} />
            <Route path="*" element={<PageNotFound useCase={useCases.CONFIG_MANAGEMENT} />} />
        </Routes>
    </searchContext.Provider>
);

export default React.memo(Page, isEqual);
