/* eslint-disable no-nested-ternary */
/* eslint-disable react/no-array-index-key */
import React, { ReactElement } from 'react';
import { Bullseye, PageSection, PageSectionVariants, Spinner } from '@patternfly/react-core';

import ACSEmptyState from 'Components/ACSEmptyState';
import useURLSort, { SortOption } from 'hooks/patternfly/useURLSort';
import useURLPagination from 'hooks/patternfly/useURLPagination';
import ObservedCVEsTable from './ObservedCVEsTable';
import useImageVulnerabilities from '../useImageVulnerabilities';

type ObservedCVEsProps = {
    imageId: string;
};

const sortFields = ['Severity', 'CVSS', 'Discovered'];
const defaultSortOption: SortOption = {
    field: 'Severity',
    direction: 'desc',
};

function ObservedCVEs({ imageId }: ObservedCVEsProps): ReactElement {
    const { page, perPage, onSetPage, onPerPageSelect } = useURLPagination();
    const { sortOption, getSortParams } = useURLSort({
        sortFields,
        defaultSortOption,
    });
    const { isLoading, data, refetchQuery } = useImageVulnerabilities({
        imageId,
        vulnsQuery: 'Vulnerability State:OBSERVED',
        pagination: {
            limit: perPage,
            offset: (page - 1) * perPage,
            sortOption: {
                field: sortOption.field,
                reversed: sortOption.direction === 'desc',
            },
        },
    });

    if (isLoading) {
        return (
            <Bullseye>
                <Spinner isSVG size="sm" />
            </Bullseye>
        );
    }

    const itemCount = data?.image?.vulnCount || 0;
    const rows = data?.image?.vulns || [];
    const registry = data?.image?.name?.registry || '';
    const remote = data?.image?.name?.remote || '';
    const tag = data?.image?.name?.tag || '';

    if (!isLoading && rows && rows.length === 0) {
        return (
            <PageSection variant={PageSectionVariants.light} isFilled>
                <ACSEmptyState title="No CVEs available" />
            </PageSection>
        );
    }

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
            getSortParams={getSortParams}
            updateTable={refetchQuery}
        />
    );
}

export default ObservedCVEs;
