import React from 'react';
import { useQuery } from '@apollo/client';
import { Bullseye, Spinner, Divider } from '@patternfly/react-core';

import useURLSort from 'hooks/useURLSort';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { getHasSearchApplied } from 'utils/searchUtils';
import { VulnerabilityState } from 'types/cve.proto';
import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';

import DeploymentsTable, { Deployment, deploymentListQuery } from '../Tables/DeploymentsTable';
import TableEntityToolbar, { TableEntityToolbarProps } from '../../components/TableEntityToolbar';
import {
    getVulnStateScopedQueryString,
    parseWorkloadQuerySearchFilter,
} from '../../utils/searchUtils';
import { VulnerabilitySeverityLabel } from '../../types';

type DeploymentsTableContainerProps = {
    filterToolbar: TableEntityToolbarProps['filterToolbar'];
    entityToggleGroup: TableEntityToolbarProps['entityToggleGroup'];
    rowCount: number;
    vulnerabilityState?: VulnerabilityState; // TODO Make this required when the ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL feature flag is removed
    pagination: ReturnType<typeof useURLPagination>;
    sort: ReturnType<typeof useURLSort>;
};

function DeploymentsTableContainer({
    filterToolbar,
    entityToggleGroup,
    rowCount,
    vulnerabilityState,
    pagination,
    sort,
}: DeploymentsTableContainerProps) {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseWorkloadQuerySearchFilter(searchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);
    const { page, perPage } = pagination;
    const { sortOption, getSortParams } = sort;

    const { error, loading, data, previousData } = useQuery<{
        deployments: Deployment[];
    }>(deploymentListQuery, {
        variables: {
            query: getVulnStateScopedQueryString(querySearchFilter, vulnerabilityState),
            pagination: {
                offset: (page - 1) * perPage,
                limit: perPage,
                sortOption,
            },
        },
    });

    const tableData = data ?? previousData;
    return (
        <>
            <TableEntityToolbar
                filterToolbar={filterToolbar}
                entityToggleGroup={entityToggleGroup}
                pagination={pagination}
                tableRowCount={rowCount}
                isFiltered={isFiltered}
            />
            <Divider component="div" />
            {loading && !tableData && (
                <Bullseye>
                    <Spinner  />
                </Bullseye>
            )}
            {error && (
                <TableErrorComponent error={error} message="Adjust your filters and try again" />
            )}
            {!error && tableData && (
                <div
                    className="workload-cves-table-container"
                    role="region"
                    aria-live="polite"
                    aria-busy={loading ? 'true' : 'false'}
                >
                    <DeploymentsTable
                        deployments={tableData.deployments}
                        getSortParams={getSortParams}
                        isFiltered={isFiltered}
                        filteredSeverities={searchFilter.SEVERITY as VulnerabilitySeverityLabel[]}
                    />
                </div>
            )}
        </>
    );
}

export default DeploymentsTableContainer;
