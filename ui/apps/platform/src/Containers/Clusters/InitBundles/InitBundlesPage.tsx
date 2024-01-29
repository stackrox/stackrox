import React, { ReactElement, useState } from 'react';
import { Alert, Bullseye, Button, PageSection, Spinner } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import useRestQuery from 'hooks/useRestQuery';
import { ClusterInitBundle, fetchClusterInitBundles } from 'services/ClustersService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { clustersInitBundlesPath } from 'routePaths';

import InitBundlesHeader, { titleInitBundles } from './InitBundlesHeader';
import InitBundlesTable from './InitBundlesTable';
import RevokeBundleModal from './RevokeBundleModal';

export type InitBundlesPageProps = {
    hasWriteAccessForInitBundles: boolean;
};

function InitBundlesPage({ hasWriteAccessForInitBundles }: InitBundlesPageProps): ReactElement {
    const [initBundleToRevoke, setInitBundleToRevoke] = useState<ClusterInitBundle | null>(null);
    const headerActions = hasWriteAccessForInitBundles ? (
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
        error: errorForFetch,
        loading: isFetching,
        refetch,
    } = useRestQuery(fetchClusterInitBundles);

    function onCloseModal(wasRevoked: boolean) {
        setInitBundleToRevoke(null);
        if (wasRevoked) {
            refetch();
        }
    }

    /* eslint-disable no-nested-ternary */
    return (
        <>
            <InitBundlesHeader headerActions={headerActions} title={titleInitBundles} />
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
                    <>
                        <InitBundlesTable
                            hasWriteAccessForInitBundles={hasWriteAccessForInitBundles}
                            initBundles={dataForFetch?.response?.items ?? []}
                            setInitBundleToRevoke={setInitBundleToRevoke}
                        />
                        {initBundleToRevoke && (
                            <RevokeBundleModal
                                initBundle={initBundleToRevoke}
                                onCloseModal={onCloseModal}
                            />
                        )}
                    </>
                )}
            </PageSection>
        </>
    );
    /* eslint-enable no-nested-ternary */
}

export default InitBundlesPage;
