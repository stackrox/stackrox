/* eslint-disable no-nested-ternary */
/* eslint-disable react/no-array-index-key */
import React, { ReactElement } from 'react';
import { Bullseye, PageSection, PageSectionVariants, Spinner } from '@patternfly/react-core';

import usePagination from 'hooks/patternfly/usePagination';
import ACSEmptyState from 'Components/ACSEmptyState';
import ObservedCVEsTable from './ObservedCVEsTable';
import useImageVulnerabilities from '../useImageVulnerabilities';

type ObservedCVEsProps = {
    imageId: string;
};

function ObservedCVEs({ imageId }: ObservedCVEsProps): ReactElement {
    const { page, perPage, onSetPage, onPerPageSelect } = usePagination();
    const { isLoading, data, refetchQuery } = useImageVulnerabilities({
        imageId,
        vulnsQuery: 'Vulnerability State:OBSERVED',
        pagination: {
            limit: perPage,
            offset: (page - 1) * perPage,
            sortOption: {
                field: 'Severity',
                reversed: true,
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
            updateTable={refetchQuery}
        />
    );
}

export default ObservedCVEs;
