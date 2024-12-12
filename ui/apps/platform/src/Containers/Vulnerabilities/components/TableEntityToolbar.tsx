import React, { ReactNode } from 'react';
import {
    Divider,
    Toolbar,
    ToolbarItem,
    ToolbarContent,
    Pagination,
    ToolbarGroup,
} from '@patternfly/react-core';

import { UseURLPaginationResult } from 'hooks/useURLPagination';

import { DynamicTableLabel } from 'Components/DynamicIcon';

export type TableEntityToolbarProps = {
    /** The toolbar component used for searching, filtering, and displaying filter chips */
    filterToolbar: ReactNode;
    /** The toolbar component used for toggling between different entities for the given CVE context */
    entityToggleGroup: ReactNode;
    /** The current table pagination object */
    pagination: UseURLPaginationResult;
    /** The total number of rows in the table controlled by this toolbar */
    tableRowCount: number;
    /** Whether or not a filter is currently applied to the table */
    isFiltered: boolean;
    /**
     * Any additional children to be rendered in the toolbar.
     *  These will be rendered between the entityToggleGroup and the pagination.
     */
    children?: React.ReactNode;
};

/**
 * The TableEntityToolbar component is a toolbar used throughout VM 2.0 to display the filter toolbar, entity toggle group, and pagination.
 */
function TableEntityToolbar({
    filterToolbar,
    entityToggleGroup,
    pagination,
    tableRowCount,
    isFiltered,
    children,
}: TableEntityToolbarProps) {
    const { page, perPage, setPage, setPerPage } = pagination;
    return (
        <>
            {filterToolbar}
            <Divider component="div" />
            <Toolbar>
                <ToolbarContent className="pf-v5-u-justify-content-space-between">
                    <ToolbarGroup className="pf-v5-u-flex-grow-1">
                        <ToolbarItem>{entityToggleGroup}</ToolbarItem>
                        {isFiltered && (
                            <ToolbarItem alignSelf="center">
                                <DynamicTableLabel />
                            </ToolbarItem>
                        )}
                    </ToolbarGroup>
                    <ToolbarGroup align={{ default: 'alignLeft' }}>
                        {children}
                        <ToolbarItem variant="pagination">
                            <Pagination
                                itemCount={tableRowCount}
                                page={page}
                                perPage={perPage}
                                onSetPage={(_, newPage) => setPage(newPage)}
                                onPerPageSelect={(_, newPerPage) => {
                                    setPerPage(newPerPage);
                                }}
                            />
                        </ToolbarItem>
                    </ToolbarGroup>
                </ToolbarContent>
            </Toolbar>
        </>
    );
}

export default TableEntityToolbar;
