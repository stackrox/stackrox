import type { ReactElement } from 'react';
import { Route, Routes } from 'react-router-dom-v5-compat';

import { Subnav } from 'Components/Navigation/SubnavContext';
import ViolationsTablePage from './ViolationsTablePage';
import ViolationDetailsPage from './Details/ViolationDetailsPage';
import ViolationNotFoundPage from './ViolationNotFoundPage';
import ViolationsSubnav from './ViolationsSubnav';

function ViolationsPage(): ReactElement {
    return (
        <>
            <Subnav>
                <ViolationsSubnav />
            </Subnav>
            <Routes>
                <Route index element={<ViolationsTablePage />} />
                <Route path=":alertId" element={<ViolationDetailsPage />} />
                <Route path="*" element={<ViolationNotFoundPage />} />
            </Routes>
        </>
    );
}

export default ViolationsPage;
