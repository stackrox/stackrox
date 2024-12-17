import React, { ReactElement, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Alert, Bullseye, Button, PageSection, Spinner } from '@patternfly/react-core';

import useRestQuery from 'hooks/useRestQuery';
import { fetchClusterInitBundles } from 'services/ClustersService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import InitBundleDescription from './InitBundleDescription';
import InitBundlesHeader from './InitBundlesHeader';
import RevokeBundleModal from './RevokeBundleModal';

export type InitBundlePageProps = {
    hasWriteAccessForInitBundles: boolean;
    id: string;
};

function InitBundlePage({ hasWriteAccessForInitBundles, id }: InitBundlePageProps): ReactElement {
    const navigate = useNavigate();
    const [isRevoking, setIsRevoking] = useState(false);

    const {
        data: dataForFetch,
        isLoading: isFetching,
        error: errorForFetch,
    } = useRestQuery(fetchClusterInitBundles);

    const initBundle = dataForFetch?.response?.items.find(
        (initBundleArg) => initBundleArg.id === id
    );

    function onClickRevoke() {
        setIsRevoking(true);
    }

    function onCloseModal(wasRevoked: boolean) {
        setIsRevoking(false);
        if (wasRevoked) {
            navigate(-1); // to table
        }
    }

    const headerActions =
        hasWriteAccessForInitBundles && initBundle ? (
            <Button
                variant="danger"
                isDisabled={isRevoking}
                isLoading={isRevoking}
                onClick={onClickRevoke}
            >
                Revoke bundle
            </Button>
        ) : null;

    /* eslint-disable no-nested-ternary */
    return (
        <>
            <InitBundlesHeader headerActions={headerActions} title="Cluster init bundle" />
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
                ) : initBundle ? (
                    <>
                        <InitBundleDescription initBundle={initBundle} />
                        {isRevoking && (
                            <RevokeBundleModal
                                initBundle={initBundle}
                                onCloseModal={onCloseModal}
                            />
                        )}
                    </>
                ) : (
                    <Alert
                        variant="warning"
                        title="Unable to find cluster init bundle"
                        component="p"
                        isInline
                    />
                )}
            </PageSection>
        </>
    );
    /* eslint-enable no-nested-ternary */
}

export default InitBundlePage;
