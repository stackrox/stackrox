import React, { useCallback } from 'react';
import { Card, CardBody, PageSection, Text } from '@patternfly/react-core';

import { getTableUIState } from 'utils/getTableUIState';
import useRestQuery from 'hooks/useRestQuery';
import { fetchOnDemandReportHistory } from 'services/ReportsService';
import PageTitle from 'Components/PageTitle';
import OnDemandReportsTable from './OnDemandReportsTable';

function OnDemandReportsTab() {
    // @TODO: Pass query, pagination, sorting to API call
    const fetchOnDemandReportsHistoryCallback = useCallback(() => fetchOnDemandReportHistory(), []);
    const { data, isLoading, error } = useRestQuery(fetchOnDemandReportsHistoryCallback);

    // @TODO: Add polling

    const tableState = getTableUIState({
        isLoading,
        data,
        error,
        searchFilter: {},
        isPolling: false,
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
                        <OnDemandReportsTable
                            tableState={tableState}
                            onClearFilters={() => {
                                // @TODO: Clear search filter and reset pagination
                            }}
                        />
                    </CardBody>
                </Card>
            </PageSection>
        </>
    );
}

export default OnDemandReportsTab;
