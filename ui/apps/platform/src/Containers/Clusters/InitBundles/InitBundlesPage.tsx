import React, { ReactElement, useState } from 'react';
import { Alert, Bullseye, Button, PageSection, Spinner } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import useAnalytics, { CREATE_INIT_BUNDLE_CLICKED } from 'hooks/useAnalytics';
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
    const { analyticsTrack } = useAnalytics();
    const [initBundleToRevoke, setInitBundleToRevoke] = useState<ClusterInitBundle | null>(null);
    const headerActions = hasWriteAccessForInitBundles ? (
        <Button
            variant="primary"
            component={LinkShim}
            href={`${clustersInitBundlesPath}?action=create`}
            onClick={() => {
                analyticsTrack({
                    event: CREATE_INIT_BUNDLE_CLICKED,
                    properties: { source: 'Cluster Init Bundles' },
                });
            }}
        >
            Create bundle
        </Button>
    ) : null;

    const {
        data: dataForFetch,
        error: errorForFetch,
        isLoading: isFetching,
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
                        <Spinner />
                    </Bullseye>
                ) : errorForFetch ? (
                    <Alert
                        variant="warning"
                        title="Unable to fetch cluster init bundles"
                        component="p"
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
