import React, { useCallback, useEffect, useState } from 'react';
import { Breadcrumb, BreadcrumbItem, Divider, PageSection } from '@patternfly/react-core';
import { generatePath, useParams } from 'react-router-dom';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import useRestQuery from 'hooks/useRestQuery';
import { ComplianceCheckStatus, ComplianceCheckStatusCount } from 'services/ComplianceCommon';
import { getComplianceProfileCheckStats } from 'services/ComplianceResultsStatsService';
import { GetComplianceProfileCheckResult } from 'services/ComplianceResultsService';
import { getTableUIState } from 'utils/getTableUIState';

import CheckDetailsTable from './CheckDetailsTable';
import DetailsPageHeader, { PageHeaderLabel } from './components/DetailsPageHeader';
import { coverageProfileChecksPath } from './compliance.coverage.routes';
import { getClusterResultsStatusObject } from './compliance.coverage.utils';

function sortCheckStats(a: ComplianceCheckStatusCount, b: ComplianceCheckStatusCount) {
    const order: ComplianceCheckStatus[] = [
        'PASS',
        'FAIL',
        'MANUAL',
        'ERROR',
        'INFO',
        'NOT_APPLICABLE',
        'INCONSISTENT',
    ];
    return order.indexOf(a.status) - order.indexOf(b.status);
}

function CheckDetails() {
    const { checkName, profileName } = useParams();
    const [currentDatetime, setCurrentDatetime] = useState(new Date());

    const fetchCheckStats = useCallback(
        () => getComplianceProfileCheckStats(profileName, checkName),
        [profileName, checkName]
    );
    const {
        data: checkStats,
        loading: isLoadingCheckStats,
        error: checkStatsError,
    } = useRestQuery(fetchCheckStats);

    const fetchCheckResults = useCallback(
        () => GetComplianceProfileCheckResult(profileName, checkName),
        [checkName, profileName]
    );
    const {
        data: checkResults,
        loading: isLoadingCheckResults,
        error: checkResultsError,
    } = useRestQuery(fetchCheckResults);

    const tableState = getTableUIState({
        isLoading: isLoadingCheckResults,
        data: checkResults?.checkResults,
        error: checkResultsError,
        searchFilter: {},
    });

    useEffect(() => {
        if (checkResults) {
            setCurrentDatetime(new Date());
        }
    }, [checkResults]);

    const checkStatsLabels =
        checkStats?.checkStats
            .sort(sortCheckStats)
            .reduce((acc, checkStat) => {
                const statusObject = getClusterResultsStatusObject(checkStat.status);
                if (statusObject && checkStat.count > 0) {
                    const label: PageHeaderLabel = {
                        text: `${statusObject.statusText}: ${checkStat.count}`,
                        icon: statusObject.icon,
                        color: statusObject.color,
                    };
                    return [...acc, label];
                }
                return acc;
            }, [] as PageHeaderLabel[])
            .filter((component) => component !== null) || [];

    return (
        <>
            <PageTitle title="Compliance coverage - Check" />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItem>Compliance coverage</BreadcrumbItem>
                    <BreadcrumbItemLink
                        to={generatePath(coverageProfileChecksPath, {
                            profileName,
                        })}
                    >
                        {profileName}
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{checkName}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <DetailsPageHeader
                    isLoading={isLoadingCheckStats}
                    name={checkName}
                    labels={checkStatsLabels}
                    summary={checkStats?.rationale}
                    nameScreenReaderText="Loading profile check details"
                    metadataScreenReaderText="Loading profile check details"
                    error={checkStatsError}
                    errorAlertTitle="Unable to fetch profile check stats"
                />
            </PageSection>
            <Divider component="div" />
            <PageSection>
                <CheckDetailsTable
                    currentDatetime={currentDatetime}
                    tableState={tableState}
                    profileName={profileName}
                />
            </PageSection>
        </>
    );
}

export default CheckDetails;
