import React, { useCallback } from 'react';
import { Breadcrumb, BreadcrumbItem, Divider, PageSection } from '@patternfly/react-core';
import { useParams } from 'react-router-dom';

import { complianceEnhancedCoveragePath } from 'routePaths';
import useRestQuery from 'hooks/useRestQuery';
import { ComplianceCheckStatus, ComplianceCheckStatusCount } from 'services/ComplianceCommon';
import { getComplianceProfileCheckStats } from 'services/ComplianceResultsStatsService';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import NotFoundMessage from 'Components/NotFoundMessage';
import DetailsPageHeader, { PageHeaderLabel } from './components/DetailsPageHeader';
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

    const complianceCoverageChecksURL = `${complianceEnhancedCoveragePath}/profiles/${profileName}/checks`;

    const profileCheckByIdFn = useCallback(
        () => getComplianceProfileCheckStats(profileName, checkName),
        [profileName, checkName]
    );
    const { data, loading: isLoading, error } = useRestQuery(profileCheckByIdFn);

    if (error) {
        return (
            <PageSection variant="light">
                <TableErrorComponent
                    error={error}
                    message="An error occurred. Try refreshing again"
                />
            </PageSection>
        );
    }

    if (!isLoading && !data) {
        return (
            <NotFoundMessage
                title="404: Profile check does not exist."
                message={`A profile check called "${checkName}" could not be found.`}
            />
        );
    }

    const checkStatsLabels =
        data?.checkStats
            .sort(sortCheckStats)
            .reduce((acc, checkStat) => {
                const statusObject = getClusterResultsStatusObject(checkStat.status);
                if (statusObject) {
                    const label: PageHeaderLabel = {
                        text: statusObject.statusText,
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
                    <BreadcrumbItemLink to={complianceCoverageChecksURL}>
                        {profileName}
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{checkName}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <DetailsPageHeader
                    isLoading={isLoading}
                    name={checkName}
                    labels={checkStatsLabels}
                    summary={data?.rationale}
                    nameScreenReaderText="Loading profile check details"
                    metadataScreenReaderText="Loading profile check details"
                />
            </PageSection>
            <Divider component="div" />
        </>
    );
}

export default CheckDetails;
