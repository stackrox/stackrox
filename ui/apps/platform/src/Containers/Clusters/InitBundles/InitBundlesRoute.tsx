import React, { ReactElement, useEffect, useState } from 'react';
import { useLocation, useParams } from 'react-router-dom';
import qs from 'qs';
import { Alert, Bullseye, PageSection, Spinner, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import usePermissions from 'hooks/usePermissions';
import { ClusterInitBundle, fetchClusterInitBundles } from 'services/ClustersService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import InitBundleForm from './InitBundleForm';
import InitBundleView from './InitBundleView';
import InitBundlesTable from './InitBundlesTable';

function InitBundlesRoute(): ReactElement {
    const { hasReadWriteAccess } = usePermissions();
    // Pending resolution whether resources or Admin role.
    const hasWriteAccessForInitBundles =
        hasReadWriteAccess('Administration') && hasReadWriteAccess('Integration');

    const { search } = useLocation();
    const isCreateAction =
        hasWriteAccessForInitBundles &&
        qs.parse(search, { ignoreQueryPrefix: true }).action === 'create';
    const { id } = useParams(); // see clustersInitBundlesPathWithParam in routePaths.ts

    const [isLoading, setIsLoading] = useState(false);
    const [initBundles, setInitBundles] = useState<ClusterInitBundle[]>([]);
    const [errorMessage, setErrorMessage] = useState('');

    useEffect(() => {
        setIsLoading(true);
        fetchClusterInitBundles()
            .then(({ response: { items } }) => {
                setInitBundles(items);
                setErrorMessage('');
            })
            .catch((error) => {
                setInitBundles([]);
                setErrorMessage(getAxiosErrorMessage(error));
            })
            .finally(() => {
                setIsLoading(false);
            });
    }, [setIsLoading]);

    const h1 = id || isCreateAction ? 'Init bundle' : 'Init bundles';
    const title = `Clusters - ${h1}`;
    const initBundle = initBundles.find((initBundleArg) => initBundleArg.id === id);

    /* eslint-disable no-nested-ternary */
    return (
        <>
            <PageTitle title={title} />
            <PageSection component="div" variant="light">
                <Title headingLevel="h1">{h1}</Title>
            </PageSection>
            <PageSection component="div">
                {isLoading ? (
                    <Bullseye>
                        <Spinner isSVG />
                    </Bullseye>
                ) : errorMessage ? (
                    <Alert
                        variant="warning"
                        title="Unable to fetch cluster init bundles"
                        component="div"
                        isInline
                    >
                        {errorMessage}
                    </Alert>
                ) : id && !initBundle ? (
                    <Alert
                        variant="warning"
                        title="Unable to find cluster init bundle"
                        component="div"
                        isInline
                    >
                        {id}
                    </Alert>
                ) : initBundle ? (
                    <InitBundleView initBundle={initBundle} />
                ) : isCreateAction ? (
                    <InitBundleForm />
                ) : (
                    <InitBundlesTable initBundles={initBundles} />
                )}
            </PageSection>
        </>
    );
    /* eslint-enable no-nested-ternary */
}

export default InitBundlesRoute;
