/* eslint-disable no-nested-ternary */
/* eslint-disable react/no-array-index-key */
import React, { ReactElement } from 'react';
import { Bullseye, Spinner } from '@patternfly/react-core';

import { useQuery } from '@apollo/client';
import {
    GetObservedCVEsData,
    GetObservedCVEsVars,
    GET_OBSERVED_CVES,
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

    const rows = data?.result?.vulns || [];
    const imageName = data?.result?.name?.fullName.split(':')[0] || '';
    const tag = data?.result?.name?.tag || '';

    return <ObservedCVEsTable rows={rows} imageName={imageName} tag={tag} isLoading={isLoading} />;
}

export default ObservedCVEs;
