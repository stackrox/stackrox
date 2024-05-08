import React, { useCallback } from 'react';
import { Breadcrumb, BreadcrumbItem, Divider, PageSection } from '@patternfly/react-core';
import { useParams } from 'react-router-dom';
import {
    BanIcon,
    CheckCircleIcon,
    ExclamationTriangleIcon,
    InfoIcon,
    SecurityIcon,
    UnknownIcon,
    WrenchIcon,
} from '@patternfly/react-icons';

import { complianceEnhancedCoveragePath } from 'routePaths';
import useRestQuery from 'hooks/useRestQuery';
import { ComplianceCheckStatus, ComplianceCheckStatusCount } from 'services/ComplianceCommon';
import { getComplianceProfileCheckStats } from 'services/ComplianceResultsStatsService';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import NotFoundMessage from 'Components/NotFoundMessage';
import DetailsPageHeader, { PageHeaderLabel } from './components/DetailsPageHeader';

const STATUS_LABELS: Record<ComplianceCheckStatus, string> = {
    FAIL: 'Fail',
    INFO: 'Info',
    PASS: 'Pass',
    ERROR: 'Error',
    MANUAL: 'Manual',
    INCONSISTENT: 'Inconsistent',
    NOT_APPLICABLE: 'Not applicable',
    UNSET_CHECK_STATUS: 'Unset check status',
};

function getCheckStatIcon(status: ComplianceCheckStatus) {
    switch (status) {
        case 'FAIL':
            return <SecurityIcon color="red" />;
        case 'INFO':
            return <InfoIcon />;
        case 'PASS':
            return <CheckCircleIcon color="blue" />;
        case 'ERROR':
            return <ExclamationTriangleIcon />;
        case 'MANUAL':
            return <WrenchIcon />;
        case 'INCONSISTENT':
            return <UnknownIcon />;
        case 'NOT_APPLICABLE':
            return <BanIcon />;
        default:
            return null;
    }
}

function getCheckStatColor(status: ComplianceCheckStatus) {
    switch (status) {
        case 'FAIL':
            return 'red';
        case 'PASS':
            return 'blue';
        default:
            return 'grey';
    }
}

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
                const statusLabel = STATUS_LABELS[checkStat.status];
                if (statusLabel) {
                    const label: PageHeaderLabel = {
                        text: `${statusLabel}: ${checkStat.count}`,
                        icon: getCheckStatIcon(checkStat.status),
                        color: getCheckStatColor(checkStat.status),
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
