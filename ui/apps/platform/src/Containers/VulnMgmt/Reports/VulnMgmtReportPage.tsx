/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { ReactElement, useEffect, useState } from 'react';
import { useLocation, useParams } from 'react-router-dom';
import { Alert, Bullseye, PageSection, Spinner } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import { fetchReportById } from 'services/ReportsService';
import { ReportConfiguration } from 'types/report.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { getQueryObject, ExtendedPageAction } from 'utils/queryStringUtils';
import { VulnMgmtReportQueryObject } from './VulnMgmtReportsMainPage';
import VulnMgmtReportDetail from './Detail/VulnMgmtReportDetail';

const emptyReportValues: ReportConfiguration = {
    id: '',
    name: '',
    description: '',
    type: 'VULNERABILITY',
    vulnReportFilters: {
        fixability: 'BOTH',
        sinceLastReport: false,
        severities: [],
    },
    scopeId: '',
    notifierConfig: {
        emailConfig: {
            notifierId: '',
            mailingLists: [],
        },
    },
    schedule: {
        intervalType: 'WEEKLY',
        hour: 0,
        minute: 0,
        interval: {
            days: [],
        },
    },
};

type VulnMgmtReportPageProps = {
    pageAction?: ExtendedPageAction;
    reportId?: string;
};

function VulnMgmtReportPage(): ReactElement {
    const [report, setReport] = useState<ReportConfiguration>(emptyReportValues);
    const [reportError, setReportError] = useState<ReactElement | null>(null);
    const [isLoading, setIsLoading] = useState(false);

    const { search } = useLocation();
    const queryObject = getQueryObject<VulnMgmtReportQueryObject>(search);
    const { action } = queryObject;
    const { reportId } = useParams();

    useEffect(() => {
        setReportError(null);
        if (reportId) {
            setIsLoading(true);
            fetchReportById(reportId)
                .then((data) => {
                    setReport(data);
                })
                .catch((error) => {
                    setReport(emptyReportValues);
                    setReportError(
                        <Alert title="Request failure for report" variant="danger" isInline>
                            {getAxiosErrorMessage(error)}
                        </Alert>
                    );
                })
                .finally(() => {
                    setIsLoading(false);
                });
        }
    }, [action, reportId]);

    return (
        <PageSection variant="light" isFilled id="report-page">
            <PageTitle title={`Reports - ${report?.name}`} />
            {isLoading ? (
                <Bullseye>
                    <Spinner />
                </Bullseye>
            ) : (
                <VulnMgmtReportDetail report={report} />
            )}
        </PageSection>
    );
}

export default VulnMgmtReportPage;
