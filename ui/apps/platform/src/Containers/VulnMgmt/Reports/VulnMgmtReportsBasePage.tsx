/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { ReactElement } from 'react';
import { useLocation } from 'react-router-dom';

import { getQueryObject } from 'utils/queryStringUtils';
import VulnMgmtCreateReportPage from './VulnMgmtCreateReportPage';
import VulnMgmtReportTablePage from './VulnMgmtReportTablePage';

function VulnMgmtReportsMainPage(): ReactElement {
    const { search } = useLocation();
    const queryObject = getQueryObject(search);
    const { action } = queryObject;

    if (action === 'create') {
        return <VulnMgmtCreateReportPage />;
    }

    return <VulnMgmtReportTablePage query={queryObject} />;
}

export default VulnMgmtReportsMainPage;
