/* eslint-disable no-nested-ternary */
/* eslint-disable react/no-array-index-key */
import React, { ReactElement } from 'react';
import { Bullseye, Spinner } from '@patternfly/react-core';

import usePagination from 'hooks/patternfly/usePagination';
import DeferredCVEsTable from './DeferredCVEsTable';
import useImageVulnerabilities from '../useImageVulnerabilities';
import { VulnerabilityWithRequest } from '../imageVulnerabilities.graphql';

type DeferredCVEsProps = {
    imageId: string;
};

function DeferredCVEs({ imageId }: DeferredCVEsProps): ReactElement {
    const { page, perPage, onSetPage, onPerPageSelect } = usePagination();
    const { isLoading, data, refetchQuery } = useImageVulnerabilities({
        imageId,
        vulnsQuery: 'Vulnerability State:DEFERRED',
        pagination: {
            limit: perPage,
            offset: (page - 1) * perPage,
            sortOption: {
                field: 'cve',
                reversed: false,
            },
        },
    });

    if (isLoading) {
        return (
            <Bullseye>
                <Spinner size="sm" />
            </Bullseye>
        );
    }

    const itemCount = data?.image?.vulnCount || 0;
    const rows = (data?.image?.vulns || []) as VulnerabilityWithRequest[];

    return (
        <DeferredCVEsTable
            rows={rows}
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

export default DeferredCVEs;
