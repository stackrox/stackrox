import React, { ReactElement } from 'react';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Bullseye, Card, Divider, Spinner } from '@patternfly/react-core';
import { CubesIcon } from '@patternfly/react-icons';

import { SlimUser } from 'types/user.proto';
import { ReportConfiguration, ReportRequestType } from 'services/ReportsService.types';
import { getReportFormValuesFromConfiguration } from 'Containers/Vulnerabilities/VulnerablityReporting/utils';
import { getDateTime } from 'utils/dateUtils';
import useSet from 'hooks/useSet';
import useFetchReportHistory from 'Containers/Vulnerabilities/VulnerablityReporting/api/useFetchReportHistory';

import NotFoundMessage from 'Components/NotFoundMessage/NotFoundMessage';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate/EmptyStateTemplate';
import LastRunStatusState from '../VulnReports/LastRunStatusState';
import ReportParametersDetails from '../components/ReportParametersDetails';
import DeliveryDestinationsDetails from '../components/DeliveryDestinationsDetails';
import ScheduleDetails from '../components/ScheduleDetails';

export type RunTypeProps = {
    reportRequestType: ReportRequestType;
    user: SlimUser;
};

function RunType({ reportRequestType, user }: RunTypeProps): ReactElement {
    if (reportRequestType === 'ON_DEMAND') {
        return (
            <div>
                On-demand / <span className="pf-u-color-200">Requested by {user.name}</span>
            </div>
        );
    }
    if (reportRequestType === 'SCHEDULED') {
        return <div>Scheduled</div>;
    }
    return <div>-</div>;
}

export type RunHistoryProps = {
    reportId: string;
};

function RunHistory({ reportId }: RunHistoryProps) {
    const { reportSnapshots, isLoading, error } = useFetchReportHistory({
        id: reportId,
        // @TODO: Replace these with actual variables
        query: '',
        page: 1,
        perPage: 10,
        showMyHistory: false,
    });
    const expandedRowSet = useSet<string>();

    if (isLoading) {
        return (
            <Bullseye>
                <Spinner isSVG />
            </Bullseye>
        );
    }

    if (error) {
        return (
            <NotFoundMessage
                title="Error fetching report history"
                message={error || 'No data available'}
            />
        );
    }

    if (!reportSnapshots.length) {
        return (
            <Bullseye>
                <EmptyStateTemplate title="No run history" headingLevel="h2" icon={CubesIcon} />
            </Bullseye>
        );
    }

    return (
        <TableComposable aria-label="Simple table" variant="compact">
            <Thead>
                <Tr>
                    <Td>{/* Header for expanded column */}</Td>
                    <Th>Run time</Th>
                    <Th>Status</Th>
                    <Th>Run type</Th>
                </Tr>
            </Thead>
            {reportSnapshots.map(
                (
                    {
                        id,
                        name,
                        description,
                        vulnReportFilters,
                        collectionSnapshot,
                        schedule,
                        notifiers,
                        reportStatus,
                        user,
                    },
                    rowIndex
                ) => {
                    const isExpanded = expandedRowSet.has(id);
                    const reportConfiguration: ReportConfiguration = {
                        id,
                        name,
                        description,
                        type: 'VULNERABILITY',
                        vulnReportFilters,
                        notifiers,
                        schedule,
                        resourceScope: {
                            collectionScope: {
                                collectionId: collectionSnapshot.id,
                                collectionName: collectionSnapshot.name,
                            },
                        },
                    };
                    const formValues = getReportFormValuesFromConfiguration(reportConfiguration);

                    return (
                        <Tbody key={id} isExpanded={isExpanded}>
                            <Tr>
                                <Td
                                    expand={{
                                        rowIndex,
                                        isExpanded,
                                        onToggle: () => expandedRowSet.toggle(id),
                                    }}
                                />
                                <Td dataLabel="Run time">
                                    {getDateTime(reportStatus.completedAt)}
                                </Td>
                                <Td dataLabel="Status">
                                    <LastRunStatusState reportStatus={reportStatus} />
                                </Td>
                                <Td dataLabel="Run type">
                                    <RunType
                                        reportRequestType={reportStatus.reportRequestType}
                                        user={user}
                                    />
                                </Td>
                            </Tr>
                            <Tr isExpanded={isExpanded}>
                                <Td colSpan={4}>
                                    <Card className="pf-u-m-md pf-u-p-md" isFlat>
                                        <ReportParametersDetails formValues={formValues} />
                                        <Divider component="div" className="pf-u-py-md" />
                                        <DeliveryDestinationsDetails formValues={formValues} />
                                        <Divider component="div" className="pf-u-py-md" />
                                        <ScheduleDetails formValues={formValues} />
                                    </Card>
                                </Td>
                            </Tr>
                        </Tbody>
                    );
                }
            )}
        </TableComposable>
    );
}

export default RunHistory;
