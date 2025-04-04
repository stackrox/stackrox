import React from 'react';
import {
    Card,
    CardBody,
    Divider,
    Flex,
    PageSection,
    Pagination,
    Text,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';

import useURLPagination from 'hooks/useURLPagination';
import PageTitle from 'Components/PageTitle';
import OnDemandReportsTable from './OnDemandReportsTable';
import { getTableUIState } from 'utils/getTableUIState';
import useURLSort from 'hooks/useURLSort';
import useURLSearch from 'hooks/useURLSearch';
import { OnDemandReportSnapshot } from 'services/ReportsService.types';

const mockOnDemandReportJobs: OnDemandReportSnapshot[] = [
    {
        reportJobId: '3dde30b0-179b-49b4-922d-0d05606c21fb',
        isOnDemand: true,
        name: '',
        requestName: 'SC-040925-01',
        areaOfConcern: 'User workloads',
        vulnReportFilters: {
            imageTypes: ['DEPLOYED'],
            includeNvdCvss: false,
            includeEpssProbability: false,
            query: '',
        },
        reportStatus: {
            runState: 'GENERATED',
            completedAt: '2024-11-13T18:45:32.997367670Z',
            errorMsg: '',
            reportRequestType: 'ON_DEMAND',
            reportNotificationMethod: 'DOWNLOAD',
        },
        user: {
            id: 'sso:4df1b98c-24ed-4073-a9ad-356aec6bb62d:admin',
            name: 'admin',
        },
        isDownloadAvailable: true,
    },
];

const sortOptions = {
    sortFields: ['Compliance Report Completed Time'],
    defaultSortOption: { field: 'Compliance Report Completed Time', direction: 'desc' } as const,
};

function OnDemandReportsTab() {
    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { sortOption, getSortParams } = useURLSort(sortOptions);
    const { searchFilter, setSearchFilter } = useURLSearch();

    const isLoading = false;
    const error = undefined;
    const data = mockOnDemandReportJobs;

    const tableState = getTableUIState({
        isLoading,
        data,
        error,
        searchFilter,
        isPolling: true,
    });

    return (
        <>
            <PageTitle title="Vulnerability reporting - On-demand reports" />
            <PageSection variant="light">
                <Text>
                    Check job status and download on-demand reports in CSV format. Requests are
                    purged according to retention settings.
                </Text>
            </PageSection>
            <PageSection>
                <Card>
                    <CardBody className="pf-v5-u-p-0">
                        <Toolbar>
                            <ToolbarContent>
                                <ToolbarItem variant="pagination" align={{ default: 'alignRight' }}>
                                    <Pagination
                                        toggleTemplate={({ firstIndex, lastIndex }) => (
                                            <span>
                                                <b>
                                                    {firstIndex} - {lastIndex}
                                                </b>{' '}
                                                of <b>many</b>
                                            </span>
                                        )}
                                        page={page}
                                        perPage={perPage}
                                        onSetPage={(_, newPage) => setPage(newPage)}
                                        onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                                        isCompact
                                    />
                                </ToolbarItem>
                            </ToolbarContent>
                        </Toolbar>
                        <OnDemandReportsTable
                            tableState={tableState}
                            getSortParams={getSortParams}
                            onClearFilters={() => {
                                setSearchFilter({});
                                setPage(1);
                            }}
                        />
                    </CardBody>
                </Card>
            </PageSection>
        </>
    );
}

export default OnDemandReportsTab;
