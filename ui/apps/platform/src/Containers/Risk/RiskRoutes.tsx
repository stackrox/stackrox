import type { ReactElement } from 'react';
import { Route, Routes } from 'react-router-dom-v5-compat';
import PageNotFound from 'Components/PageNotFound';

import RiskTablePage from './RiskTablePage';
import RiskDetailsPage from './RiskDetailsPage';

function RiskRoutes(): ReactElement {
    return (
        <>
            <Routes>
                <Route index element={<RiskTablePage />} />
                <Route path=":deploymentId" element={<RiskDetailsPage />} />
                <Route path="*" element={<PageNotFound />} />
            </Routes>
        </>
    );
}

export default RiskRoutes;
