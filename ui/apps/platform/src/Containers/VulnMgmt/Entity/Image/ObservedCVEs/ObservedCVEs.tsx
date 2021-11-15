/* eslint-disable no-nested-ternary */
/* eslint-disable react/no-array-index-key */
import React, { ReactElement } from 'react';
import { Bullseye, Spinner } from '@patternfly/react-core';

import { useQuery } from '@apollo/client';
import {
    GetObservedCVEsData,
    GetObservedCVEsVars,
    GET_OBSERVED_CVES,
    Vulnerability,
} from './observedCVEs.graphql';

import ObservedCVEsTable from './ObservedCVEsTable';

type ObservedCVEsProps = {
    imageId: string;
};

function ObservedCVEs({ imageId }: ObservedCVEsProps): ReactElement {
    const { loading: isLoading, data } = useQuery<GetObservedCVEsData, GetObservedCVEsVars>(
        GET_OBSERVED_CVES,
        {
            variables: {
                imageId,
                vulnsQuery: '',
            },
        }
    );

    if (isLoading) {
        return (
            <Bullseye>
                <Spinner size="sm" />
            </Bullseye>
        );
    }

    // @TODO: handle error returned from API
    const rows: Vulnerability[] = data?.result?.vulns || [];

    return <ObservedCVEsTable rows={rows} isLoading={isLoading} />;
}

export default ObservedCVEs;
