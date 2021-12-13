/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { ReactElement } from 'react';
import { useLocation, useParams } from 'react-router-dom';

import { getQueryObject, ExtendedPageAction } from 'utils/queryStringUtils';
import VulnMgmtCreateReportPage from './VulnMgmtCreateReportPage';
import VulnMgmtReportTablePage from './VulnMgmtReportTablePage';

export type VulnMgmtReportQueryObject = {
    action: ExtendedPageAction;
};

function VulnMgmtReportsMainPage(): ReactElement {
    const { search } = useLocation();
    const queryObject = getQueryObject<VulnMgmtReportQueryObject>(search);
    const { action } = queryObject;

    if (action === 'create') {
        return <VulnMgmtCreateReportPage />;
    }

    return <VulnMgmtReportTablePage />;
}

export default VulnMgmtReportsMainPage;
