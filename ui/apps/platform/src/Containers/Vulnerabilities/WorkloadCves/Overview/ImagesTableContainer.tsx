import React from 'react';
import { useQuery } from '@apollo/client';
import { Bullseye, Spinner, Divider } from '@patternfly/react-core';

import useURLSort from 'hooks/useURLSort';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { getHasSearchApplied } from 'utils/searchUtils';
import { VulnerabilityState } from 'types/cve.proto';
import ImagesTable, { ImagesTableProps, imageListQuery } from '../Tables/ImagesTable';
import TableErrorComponent from '../components/TableErrorComponent';
import { EntityCounts } from '../components/EntityTypeToggleGroup';
import { getVulnStateScopedQueryString, parseQuerySearchFilter } from '../searchUtils';
import { defaultImageSortFields, imagesDefaultSort } from '../sortUtils';
import { DefaultFilters, EntityTab, VulnerabilitySeverityLabel } from '../types';
import TableEntityToolbar from '../components/TableEntityToolbar';

export { imageListQuery } from '../Tables/ImagesTable';

type ImagesTableContainerProps = {
    defaultFilters: DefaultFilters;
    countsData: EntityCounts;
    vulnerabilityState?: VulnerabilityState; // TODO Make this required when the ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL feature flag is removed
    pagination: ReturnType<typeof useURLPagination>;
    hasWriteAccessForWatchedImage: boolean;
    onWatchImage: ImagesTableProps['onWatchImage'];
    onUnwatchImage: ImagesTableProps['onUnwatchImage'];
    onEntityTabChange: (entityTab: EntityTab) => void;
};

function ImagesTableContainer({
    defaultFilters,
    countsData,
    vulnerabilityState,
    pagination,
    hasWriteAccessForWatchedImage,
    onWatchImage,
    onUnwatchImage,
    onEntityTabChange,
}: ImagesTableContainerProps) {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);
    const { page, perPage, setPage } = pagination;
    const sort = useURLSort({
        sortFields: defaultImageSortFields,
        defaultSortOption: imagesDefaultSort,
        onSort: () => setPage(1),
    });
    const { sortOption, getSortParams, setSortOption } = sort;

    const { error, loading, data, previousData } = useQuery(imageListQuery, {
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
                tableRowCount={countsData.imageCount}
                isFiltered={isFiltered}
                onEntityTabChange={onEntityTabChange}
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
                <div
                    className="workload-cves-table-container"
                    role="region"
                    aria-live="polite"
                    aria-busy={loading ? 'true' : 'false'}
                >
                    <ImagesTable
                        images={tableData.images}
                        getSortParams={getSortParams}
                        isFiltered={isFiltered}
                        filteredSeverities={searchFilter.SEVERITY as VulnerabilitySeverityLabel[]}
                        hasWriteAccessForWatchedImage={hasWriteAccessForWatchedImage}
                        onWatchImage={onWatchImage}
                        onUnwatchImage={onUnwatchImage}
                    />
                </div>
            )}
        </>
    );
}

export default ImagesTableContainer;
