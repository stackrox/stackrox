import React from 'react';
import { Divider, Toolbar, ToolbarItem, ToolbarContent, Pagination } from '@patternfly/react-core';

import { SortOption } from 'types/table';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import {
    CLUSTER_SEARCH_OPTION,
    COMPONENT_SEARCH_OPTION,
    COMPONENT_SOURCE_SEARCH_OPTION,
    DEPLOYMENT_SEARCH_OPTION,
    IMAGE_CVE_SEARCH_OPTION,
    IMAGE_SEARCH_OPTION,
    NAMESPACE_SEARCH_OPTION,
    SearchOption,
} from 'Containers/Vulnerabilities/searchOptions';
import WorkloadTableToolbar from './WorkloadTableToolbar';
import { DynamicTableLabel } from './DynamicIcon';
import EntityTypeToggleGroup, { EntityCounts } from './EntityTypeToggleGroup';
import { DefaultFilters } from '../types';

type TableEntityToolbarProps = {
    defaultFilters: DefaultFilters;
    countsData: EntityCounts;
    setSortOption: (sortOption: SortOption) => void;
    pagination: UseURLPaginationResult;
    tableRowCount: number;
    isFiltered: boolean;
    children?: React.ReactNode;
};

const searchOptions: SearchOption[] = [
    IMAGE_CVE_SEARCH_OPTION,
    IMAGE_SEARCH_OPTION,
    DEPLOYMENT_SEARCH_OPTION,
    NAMESPACE_SEARCH_OPTION,
    CLUSTER_SEARCH_OPTION,
    COMPONENT_SEARCH_OPTION,
    COMPONENT_SOURCE_SEARCH_OPTION,
];

function TableEntityToolbar({
    defaultFilters,
    countsData,
    setSortOption,
    pagination,
    tableRowCount,
    isFiltered,
    children,
}: TableEntityToolbarProps) {
    const { page, perPage, setPage, setPerPage } = pagination;
    return (
        <>
            <WorkloadTableToolbar
                defaultFilters={defaultFilters}
                onFilterChange={() => setPage(1)}
                searchOptions={searchOptions}
            />
            <Divider component="div" />
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem>
                        <EntityTypeToggleGroup
                            imageCount={countsData.imageCount}
                            cveCount={countsData.imageCVECount}
                            deploymentCount={countsData.deploymentCount}
                            setSortOption={setSortOption}
                            setPage={setPage}
                        />
                    </ToolbarItem>
                    {isFiltered && (
                        <ToolbarItem>
                            <DynamicTableLabel />
                        </ToolbarItem>
                    )}
                    {children}
                    <ToolbarItem alignment={{ default: 'alignRight' }} variant="pagination">
                        <Pagination
                            itemCount={tableRowCount}
                            page={page}
                            perPage={perPage}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => {
                                if (tableRowCount < (page - 1) * newPerPage) {
                                    setPage(1);
                                }
                                setPerPage(newPerPage);
                            }}
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
        </>
    );
}

export default TableEntityToolbar;
