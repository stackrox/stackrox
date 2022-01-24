/* eslint-disable no-nested-ternary */
/* eslint-disable react/no-array-index-key */
import React, { ReactElement } from 'react';
import { Bullseye, Spinner } from '@patternfly/react-core';

import usePagination from 'hooks/patternfly/usePagination';
import FalsePositiveCVEsTable from './FalsePositiveCVEsTable';
import useImageVulnerabilities from '../useImageVulnerabilities';

type FalsePositiveCVEsProps = {
    imageId: string;
};

function FalsePositiveCVEs({ imageId }: FalsePositiveCVEsProps): ReactElement {
    const { page, perPage, onSetPage, onPerPageSelect } = usePagination();
    const { isLoading, data, refetchQuery } = useImageVulnerabilities({
        imageId,
        vulnsQuery: 'Vulnerability State:FALSE_POSITIVE',
        pagination: {
            limit: perPage,
            offset: (page - 1) * perPage,
            sortOption: {
                field: 'cve',
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
        />
    );
}

export default FalsePositiveCVEs;
