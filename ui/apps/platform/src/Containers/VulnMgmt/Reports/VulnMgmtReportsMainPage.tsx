/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { ReactElement } from 'react';
import { useLocation } from 'react-router-dom';

import {
    AccessControlQueryAction,
    getQueryObject,
} from 'Containers/AccessControl/accessControlPaths';
import VulnMgmtCreateReportPage from './VulnMgmtCreateReportPage';
import VulnMgmtReportTablePage from './VulnMgmtReportTablePage';

function VulnMgmtReportsMainPage(): ReactElement {
    const { search } = useLocation();
    const queryObject = getQueryObject(search);
    const { action } = queryObject;

    if (action === 'create') {
        return <VulnMgmtCreateReportPage />;
    }

    return <VulnMgmtReportTablePage />;
}

export default VulnMgmtReportsMainPage;
