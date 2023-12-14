import React, { ReactElement, useState } from 'react';
import { Alert, Bullseye, Button, PageSection, Spinner } from '@patternfly/react-core';

import useRestQuery from 'hooks/useRestQuery';
import { fetchClusterInitBundles, revokeClusterInitBundles } from 'services/ClustersService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import InitBundleDescription from './InitBundleDescription';
import InitBundlesHeader from './InitBundlesHeader';

export type InitBundlePageProps = {
    hasWriteAccessForInitBundles: boolean;
    id: string;
};

function InitBundlePage({ hasWriteAccessForInitBundles, id }: InitBundlePageProps): ReactElement {
    const [isRevoking, setIsRevoking] = useState(false);
    const [errorMessageForRevoke, setErrorMessageForRevoke] = useState('');

    const {
        data: dataForFetch,
        loading: isFetching,
        error: errorForFetch,
    } = useRestQuery(fetchClusterInitBundles);

    const initBundle = dataForFetch?.response?.items.find(
        (initBundleArg) => initBundleArg.id === id
    );

    function onClickRevoke() {
        setIsRevoking(true);
        // TODO investigate second argument and add confirmation modal.
        revokeClusterInitBundles([id], [])
            .then(() => {
                setErrorMessageForRevoke('');
            })
            .catch((error) => {
                setErrorMessageForRevoke(getAxiosErrorMessage(error));
            })
            .finally(() => {
                setIsRevoking(false);
            });
    }

    const alignRightElement =
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
            <InitBundlesHeader
                alignRightElement={alignRightElement}
                titleNotInitBundles="Cluster init bundle"
            />
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
                ) : initBundle ? (
                    <>
                        {errorMessageForRevoke && (
                            <Alert
                                variant="danger"
                                title="Unable to revoke cluster init bundle"
                                component="div"
                                isInline
                            >
                                {errorMessageForRevoke}
                            </Alert>
                        )}
                        <InitBundleDescription initBundle={initBundle} />
                    </>
                ) : (
                    <Alert
                        variant="warning"
                        title="Unable to find cluster init bundle"
                        component="div"
                        isInline
                    />
                )}
            </PageSection>
        </>
    );
    /* eslint-enable no-nested-ternary */
}

export default InitBundlePage;
