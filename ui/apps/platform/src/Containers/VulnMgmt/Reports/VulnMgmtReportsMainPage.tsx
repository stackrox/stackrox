/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { ReactElement } from 'react';
import { useHistory, useLocation, useParams } from 'react-router-dom';

import {
    AccessControlQueryAction,
    getQueryObject,
} from 'Containers/AccessControl/accessControlPaths';
import VulnMgmtReportTablePage from './VulnMgmtReportTablePage';

function VulnMgmtReportsMainPage(): ReactElement {
    const { search } = useLocation();
    const queryObject = getQueryObject(search);
    const { action } = queryObject;

    if (action === 'create') {
        return <h1>Vulnerability reporting</h1>;
    }

    return <VulnMgmtReportTablePage />;
}

export default VulnMgmtReportsMainPage;
