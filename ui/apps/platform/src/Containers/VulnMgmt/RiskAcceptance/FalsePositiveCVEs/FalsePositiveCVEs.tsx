/* eslint-disable no-nested-ternary */
/* eslint-disable react/no-array-index-key */
import React, { ReactElement, useState } from 'react';

import usePagination from 'hooks/patternfly/usePagination';
import { SearchFilter } from 'types/search';
import queryService from 'utils/queryService';
import useTableSort from 'hooks/patternfly/useTableSort';
import { SortOption } from 'types/table';
import FalsePositiveCVEsTable from './FalsePositiveCVEsTable';
import useImageVulnerabilities from '../useImageVulnerabilities';

type FalsePositiveCVEsProps = {
    imageId: string;
};

const sortFields = ['Severity'];
const defaultSortOption: SortOption = {
    field: 'Severity',
    direction: 'desc',
};

function FalsePositiveCVEs({ imageId }: FalsePositiveCVEsProps): ReactElement {
    const [searchFilter, setSearchFilter] = useState<SearchFilter>({});
    const { page, perPage, onSetPage, onPerPageSelect } = usePagination();
    const { sortOption, getSortParams } = useTableSort({
        sortFields,
        defaultSortOption,
    });

    const vulnsQuery = queryService.objectToWhereClause({
        ...searchFilter,
        'Vulnerability State': 'FALSE_POSITIVE',
    });

    const { isLoading, data, refetchQuery } = useImageVulnerabilities({
        imageId,
        vulnsQuery,
        pagination: {
            limit: perPage,
            offset: (page - 1) * perPage,
            sortOption,
        },
    });

    const itemCount = data?.image?.vulnCount || 0;
    const rows = data?.image?.imageVulnerabilities || [];

    return (
        <FalsePositiveCVEsTable
            rows={rows}
            isLoading={isLoading}
            itemCount={itemCount}
            page={page}
            perPage={perPage}
            onSetPage={onSetPage}
            onPerPageSelect={onPerPageSelect}
            updateTable={refetchQuery}
            searchFilter={searchFilter}
            setSearchFilter={setSearchFilter}
            getSortParams={getSortParams}
        />
    );
}

export default FalsePositiveCVEs;
