import React, { useState, ReactElement } from 'react';
import { useLocation, useParams } from 'react-router-dom';
import { Alert, Bullseye, PageSection, Spinner } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import useFetchReport from 'hooks/useFetchReport';
import usePermissions from 'hooks/usePermissions';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { getQueryObject } from 'utils/queryStringUtils';

import { getWriteAccessForReport } from './VulnMgmtReport.utils';
import VulnMgmtReportDetail from './Detail/VulnMgmtReportDetail';
import VulnMgmtEditReportPage from './Detail/VulnMgmtEditReportPage';

function VulnMgmtReportPage(): ReactElement {
    const { search } = useLocation();
    const [refresh, setRefresh] = useState<number>(new Date().getTime());

    function refreshQuery() {
        setRefresh(new Date().getTime());
    }

    const { hasReadAccess, hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForReport = getWriteAccessForReport({ hasReadAccess, hasReadWriteAccess });

    const queryObject = getQueryObject(search);
    const { action } = queryObject;
    const { reportId } = useParams();

    const result = useFetchReport(reportId, refresh);

    const { report, reportScope, isLoading, error } = result;

    return (
        <>
            <PageTitle title={`Vulnerability Management - Report: ${report?.name || ''}`} />
            {isLoading && (
                <PageSection isFilled id="report-page">
                    <Bullseye>
                        <Spinner isSVG />
                    </Bullseye>
                </PageSection>
            )}
            {error && (
                <Alert title="Request failure for report" variant="danger" isInline>
                    {getAxiosErrorMessage(error)}
                </Alert>
            )}
            {action === 'edit' && hasWriteAccessForReport && !!report ? (
                <VulnMgmtEditReportPage
                    report={report}
                    reportScope={reportScope}
                    refreshQuery={refreshQuery}
                />
            ) : (
                !!report && <VulnMgmtReportDetail report={report} reportScope={reportScope} />
            )}
        </>
    );
}

export default VulnMgmtReportPage;
