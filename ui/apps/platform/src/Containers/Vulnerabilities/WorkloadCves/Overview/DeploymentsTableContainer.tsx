import React from 'react';
import { useQuery } from '@apollo/client';
import { Bullseye, Spinner, Divider } from '@patternfly/react-core';

import useURLSort from 'hooks/useURLSort';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { getHasSearchApplied } from 'utils/searchUtils';
import { VulnerabilityState } from 'types/cve.proto';
import DeploymentsTable, { Deployment, deploymentListQuery } from '../Tables/DeploymentsTable';
import TableErrorComponent from '../components/TableErrorComponent';
import TableEntityToolbar from '../components/TableEntityToolbar';
import { EntityCounts } from '../components/EntityTypeToggleGroup';
import { getVulnStateScopedQueryString, parseQuerySearchFilter } from '../searchUtils';
import { defaultDeploymentSortFields, deploymentsDefaultSort } from '../sortUtils';
import { DefaultFilters, VulnerabilitySeverityLabel } from '../types';

type DeploymentsTableContainerProps = {
    defaultFilters: DefaultFilters;
    countsData: EntityCounts;
    vulnerabilityState?: VulnerabilityState; // TODO Make this required when the ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL feature flag is removed
    pagination: ReturnType<typeof useURLPagination>;
};

function DeploymentsTableContainer({
    defaultFilters,
    countsData,
    vulnerabilityState,
    pagination,
}: DeploymentsTableContainerProps) {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);
    const { page, perPage, setPage } = pagination;
    const { sortOption, getSortParams, setSortOption } = useURLSort({
        sortFields: defaultDeploymentSortFields,
        defaultSortOption: deploymentsDefaultSort,
        onSort: () => setPage(1),
    });

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
                defaultFilters={defaultFilters}
                countsData={countsData}
                setSortOption={setSortOption}
                pagination={pagination}
                tableRowCount={countsData.deploymentCount}
                isFiltered={isFiltered}
            />
            <Divider component="div" />
            {loading && !tableData && (
                <Bullseye>
                    <Spinner isSVG />
                </Bullseye>
            )}
            {error && (
                <TableErrorComponent error={error} message="Adjust your filters and try again" />
            )}
            {!error && tableData && (
                <div className="workload-cves-table-container">
                    <DeploymentsTable
                        deployments={tableData.deployments}
                        getSortParams={getSortParams}
                        isFiltered={isFiltered}
                        filteredSeverities={searchFilter.Severity as VulnerabilitySeverityLabel[]}
                    />
                </div>
            )}
        </>
    );
}

export default DeploymentsTableContainer;
