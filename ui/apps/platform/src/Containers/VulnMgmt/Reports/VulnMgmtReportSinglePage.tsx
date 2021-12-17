/* eslint-disable no-nested-ternary */
/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { ReactElement, useEffect, useState } from 'react';
import { useLocation, useParams } from 'react-router-dom';
import { Alert, Bullseye, PageSection, Spinner } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import useFetchReport from 'hooks/useFetchReport';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { getQueryObject, ExtendedPageAction } from 'utils/queryStringUtils';
import { VulnMgmtReportQueryObject } from './VulnMgmtReport.utils';
import VulnMgmtReportDetail from './Detail/VulnMgmtReportDetail';

function VulnMgmtReportPage(): ReactElement {
    const { search } = useLocation();
    // TODO: use the action param to determini if we are editing the report
    const queryObject = getQueryObject<VulnMgmtReportQueryObject>(search);
    // eslint-disable-next-line no-unused-vars
    const { action } = queryObject;
    const { reportId } = useParams();

    const result = useFetchReport(reportId);

    const { report, isLoading, error } = result;

    return (
        <>
            <PageTitle title={`Vulnerability Management - Report: ${report?.name || ''}`} />
            {isLoading ? (
                <PageSection isFilled id="report-page">
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                </PageSection>
            ) : error ? (
                <Alert title="Request failure for report" variant="danger" isInline>
                    {getAxiosErrorMessage(error)}
                </Alert>
            ) : (
                !!report && <VulnMgmtReportDetail report={report} />
            )}
        </>
    );
}

export default VulnMgmtReportPage;
