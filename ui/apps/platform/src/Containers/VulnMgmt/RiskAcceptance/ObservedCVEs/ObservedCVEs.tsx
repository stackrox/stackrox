/* eslint-disable no-nested-ternary */
/* eslint-disable react/no-array-index-key */
import React, { ReactElement, useState } from 'react';

import usePagination from 'hooks/patternfly/usePagination';
import { SearchFilter } from 'types/search';
import queryService from 'utils/queryService';
import useTableSort from 'hooks/patternfly/useTableSort';
import { SortOption } from 'types/table';
import ObservedCVEsTable from './ObservedCVEsTable';
import useImageVulnerabilities from '../useImageVulnerabilities';
import { EmbeddedImageScanComponent } from '../imageVulnerabilities.graphql';

type ObservedCVEsProps = {
    imageId: string;
    showComponentDetails: (components: EmbeddedImageScanComponent[], cveName: string) => void;
};

const sortFields = ['Severity', 'CVSS', 'Discovered'];
const defaultSortOption: SortOption = {
    field: 'Severity',
    direction: 'desc',
};

function ObservedCVEs({ imageId, showComponentDetails }: ObservedCVEsProps): ReactElement {
    const [searchFilter, setSearchFilter] = useState<SearchFilter>({});
    const { page, perPage, onSetPage, onPerPageSelect } = usePagination();
    const { sortOption, getSortParams } = useTableSort({
        sortFields,
        defaultSortOption,
    });

    const vulnsQuery = queryService.objectToWhereClause({
        ...searchFilter,
        'Vulnerability State': 'OBSERVED',
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
    const rows = data?.image?.vulns || [];
    const registry = data?.image?.name?.registry || '';
    const remote = data?.image?.name?.remote || '';
    const tag = data?.image?.name?.tag || '';

    return (
        <ObservedCVEsTable
            rows={rows}
            registry={registry}
            remote={remote}
            tag={tag}
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
            showComponentDetails={showComponentDetails}
        />
    );
}

export default ObservedCVEs;
