/* eslint-disable no-nested-ternary */
/* eslint-disable react/no-array-index-key */
import React, { ReactElement } from 'react';
import { Bullseye, Spinner } from '@patternfly/react-core';

import { useQuery } from '@apollo/client';
import usePagination from 'hooks/patternfly/usePagination';
import {
    GetImageVulnerabilitiesData,
    GetImageVulnerabilitiesVars,
    GET_IMAGE_VULNERABILITIES,
} from '../imageVulnerabilities.graphql';

import ObservedCVEsTable from './ObservedCVEsTable';

type ObservedCVEsProps = {
    imageId: string;
};

function ObservedCVEs({ imageId }: ObservedCVEsProps): ReactElement {
    const { page, perPage, onSetPage, onPerPageSelect } = usePagination();
    const { loading: isLoading, data } = useQuery<
        GetImageVulnerabilitiesData,
        GetImageVulnerabilitiesVars
    >(GET_IMAGE_VULNERABILITIES, {
        variables: {
            imageId,
            vulnsQuery: 'Vulnerability State:OBSERVED',
            pagination: {
                limit: perPage,
                offset: (page - 1) * perPage,
                sortOption: {
                    field: 'cve',
                    reversed: false,
                },
            },
        },
    });

    if (isLoading || !data) {
        return (
            <Bullseye>
                <Spinner size="sm" />
            </Bullseye>
        );
    }

    const itemCount = data.vulnerabilityCount;
    const rows = data.vulnerabilities;
    const { registry } = data.image.name;
    const { remote } = data.image.name;
    const { tag } = data.image.name;

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
        />
    );
}

export default ObservedCVEs;
