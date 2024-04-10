import React from 'react';
import { Bullseye, Spinner } from '@patternfly/react-core';

import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import useURLPagination from 'hooks/useURLPagination';

import { QuerySearchFilter } from '../../types';

export type ClustersTableProps = {
    querySearchFilter: QuerySearchFilter;
    isFiltered: boolean;
    pagination: ReturnType<typeof useURLPagination>;
};

function ClustersTable({
    /* eslint-disable @typescript-eslint/no-unused-vars */
    querySearchFilter,
    isFiltered,
    pagination,
    /* eslint-enable @typescript-eslint/no-unused-vars */
}: ClustersTableProps) {
    // TODO - Placeholders for query results
    const data = [];
    const previousData = undefined;
    const error: Error | undefined = undefined;
    const loading = false;

    const tableData = data ?? previousData;

    return (
        <>
            {loading && !tableData && (
                <Bullseye>
                    <Spinner />
                </Bullseye>
            )}
            {error && (
                <TableErrorComponent error={error} message="Adjust your filters and try again" />
            )}
            {!error && tableData && (
                <div role="region" aria-live="polite" aria-busy={loading ? 'true' : 'false'}>
                    Cluster table goes here
                </div>
            )}
        </>
    );
}

export default ClustersTable;
