import React from 'react';
import { Bullseye, Divider, Spinner } from '@patternfly/react-core';

import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import useURLPagination from 'hooks/useURLPagination';

import TableEntityToolbar, { TableEntityToolbarProps } from '../../components/TableEntityToolbar';
import { QuerySearchFilter } from '../../types';

export type CVEsTableContainerProps = {
    filterToolbar: TableEntityToolbarProps['filterToolbar'];
    entityToggleGroup: TableEntityToolbarProps['entityToggleGroup'];
    querySearchFilter: QuerySearchFilter;
    isFiltered: boolean;
    rowCount: number;
    pagination: ReturnType<typeof useURLPagination>;
};

function CVEsTableContainer({
    filterToolbar,
    entityToggleGroup,
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    querySearchFilter,
    isFiltered,
    rowCount,
    pagination,
}: CVEsTableContainerProps) {
    // TODO - Placeholders for query results
    const data = [];
    const previousData = undefined;
    const error: Error | undefined = undefined;
    const loading = false;

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
                    <Spinner isSVG />
                </Bullseye>
            )}
            {error && (
                <TableErrorComponent error={error} message="Adjust your filters and try again" />
            )}
            {!error && tableData && (
                <div role="region" aria-live="polite" aria-busy={loading ? 'true' : 'false'}>
                    Table goes here
                </div>
            )}
        </>
    );
}

export default CVEsTableContainer;
