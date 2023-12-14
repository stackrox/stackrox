import React, { ReactElement } from 'react';
import { Alert, Bullseye, Button, PageSection, Spinner } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import useRestQuery from 'hooks/useRestQuery';
import { fetchClusterInitBundles } from 'services/ClustersService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { clustersInitBundlesPath } from 'routePaths';

import InitBundlesHeader from './InitBundlesHeader';
import InitBundlesTable from './InitBundlesTable';

export type InitBundlesPageProps = {
    hasWriteAccessForInitBundles: boolean;
};

function InitBundlesPage({ hasWriteAccessForInitBundles }: InitBundlesPageProps): ReactElement {
    const alignRightElement = hasWriteAccessForInitBundles ? (
        <Button
            variant="primary"
            component={LinkShim}
            href={`${clustersInitBundlesPath}?action=create`}
        >
            Create bundle
        </Button>
    ) : null;

    const {
        data: dataForFetch,
        loading: isFetching,
        error: errorForFetch,
    } = useRestQuery(fetchClusterInitBundles);

    /* eslint-disable no-nested-ternary */
    return (
        <>
            <InitBundlesHeader alignRightElement={alignRightElement} />
            <PageSection component="div">
                {isFetching ? (
                    <Bullseye>
                        <Spinner isSVG />
                    </Bullseye>
                ) : errorForFetch ? (
                    <Alert
                        variant="warning"
                        title="Unable to fetch cluster init bundles"
                        component="div"
                        isInline
                    >
                        {getAxiosErrorMessage(errorForFetch)}
                    </Alert>
                ) : (
                    <InitBundlesTable initBundles={dataForFetch?.response?.items ?? []} />
                )}
            </PageSection>
        </>
    );
    /* eslint-enable no-nested-ternary */
}

export default InitBundlesPage;
