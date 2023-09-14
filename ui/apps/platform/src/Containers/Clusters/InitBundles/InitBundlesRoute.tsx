import React, { ReactElement, useEffect, useState } from 'react';
import { useLocation, useParams } from 'react-router-dom';
import { Alert, Bullseye, PageSection, Spinner, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import { ClusterInitBundle, fetchClusterInitBundles } from 'services/ClustersService';
import { getQueryObject } from 'utils/queryStringUtils';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import InitBundleForm from './InitBundleForm';
import InitBundleView from './InitBundleView';
import InitBundlesTable from './InitBundlesTable';

function getAction(search: string): 'create' | 'edit' | '' {
    const queryObject = getQueryObject(search);

    switch (queryObject.action) {
        case 'create':
        case 'edit':
            return queryObject.action;
        default:
            return '';
    }
}

function InitBundlesRoute(): ReactElement {
    const { search } = useLocation();
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

    const action = getAction(search);
    const h1 = id || action === 'create' ? 'Init bundle' : 'Init bundles';
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
                ) : action === 'edit' && initBundle ? (
                    <InitBundleForm initBundle={initBundle} />
                ) : initBundle ? (
                    <InitBundleView initBundle={initBundle} />
                ) : action === 'create' && !initBundle ? (
                    <InitBundleForm initBundle={null} />
                ) : (
                    <InitBundlesTable initBundles={initBundles} />
                )}
            </PageSection>
        </>
    );
    /* eslint-enable no-nested-ternary */
}

export default InitBundlesRoute;
