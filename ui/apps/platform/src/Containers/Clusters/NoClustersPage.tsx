import React, { ReactElement, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import {
    Alert,
    Bullseye,
    Button,
    EmptyState,
    EmptyStateBody,
    EmptyStateIcon,
    Flex,
    FlexItem,
    PageSection,
    Spinner,
    Title,
    Text,
} from '@patternfly/react-core';
import { CloudSecurityIcon } from '@patternfly/react-icons';

import LinkShim from 'Components/PatternFly/LinkShim';
import { getProductBranding } from 'constants/productBranding';
// import useAuthStatus from 'hooks/useAuthStatus'; // TODO after 4.4 release
import { fetchClusterInitBundles } from 'services/ClustersService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { clustersBasePath, clustersInitBundlesPath } from 'routePaths';

import SecureClusterModal from './InitBundles/SecureClusterModal';

/*
 * Comments about data flow:
 *
 * 1. It is important that /main/clusters NoClustersPage **Create bundle**
 *    goes to /main/clusters/init-bundles InitBundlesWizard in the same tab,
 *    so when **Download** goes back, NoClustersPage makes a new GET /v1/init-bundles request
 *    and therefore renders the link instead of the button.
 *
 * 2. It is important that /main/clusters NoClustersPage **Review installation methods**
 *    opens the modal SecureCluster in the same tab,
 *    so polling loop in original tab will cause conditional rendering of table
 *    whenever there is a secured cluster.
 */

export type NoClustersPageProps = {
    isModalOpen: boolean;
    setIsModalOpen: (isOpen: boolean) => void;
};

function NoClustersPage({ isModalOpen, setIsModalOpen }): ReactElement {
    /*
    // TODO after 4.4 release
    const { currentUser } = useAuthStatus();
    const hasAdminRole = Boolean(currentUser?.userInfo?.roles.some(({ name }) => name === 'Admin')); // optional chaining just in case of the unexpected
    */

    // Use promise instead of useRestQuery hook because of role-based access control.
    const [errorMessage, setErrorMessage] = useState('');
    const [initBundlesCount, setInitBundlesCount] = useState(0);
    const [isLoading, setIsLoading] = useState(false);

    const { basePageTitle } = getProductBranding();
    const textForSuccessAlert = `You have successfully deployed a ${basePageTitle} platform. Now you can configure the clusters you want to secure.`;

    useEffect(() => {
        // TODO after 4.4 release: if (hasAdminRole) {
        setIsLoading(true);
        fetchClusterInitBundles()
            .then(({ response }) => {
                setErrorMessage('');
                setInitBundlesCount(response.items.length);
            })
            .catch((error) => {
                setErrorMessage(getAxiosErrorMessage(error));
            })
            .finally(() => {
                setIsLoading(false);
            });
        // TODO after 4.4 releaes: }
    }, []); // TODO after 4.4 release [hasAdminRole]

    // TODO after 4.4 release add hasAdminRole to conditional rendering.
    /* eslint-disable no-nested-ternary */
    return (
        <>
            <PageSection variant="light" component="div" padding={{ default: 'noPadding' }}>
                <Alert isInline variant="success" title="You are ready to go!">
                    {textForSuccessAlert}
                </Alert>
            </PageSection>
            <PageSection variant="light">
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
                ) : (
                    <EmptyState variant="large">
                        <EmptyStateIcon icon={CloudSecurityIcon} />
                        <Title headingLevel="h1">Secure clusters with a reusable init bundle</Title>
                        <EmptyStateBody>
                            <Flex
                                direction={{ default: 'column' }}
                                spaceItems={{ default: 'spaceItemsLg' }}
                            >
                                <FlexItem>
                                    <Text component="p">
                                        Follow the instructions to install secured cluster services.
                                    </Text>
                                    <Text component="p">
                                        Upon successful installation, secured clusters are listed
                                        here.
                                    </Text>
                                </FlexItem>
                                {initBundlesCount !== 0 && (
                                    <FlexItem>
                                        <Text component="p">
                                            You have successfully created cluster init bundles.
                                        </Text>
                                    </FlexItem>
                                )}
                            </Flex>
                        </EmptyStateBody>
                        {initBundlesCount === 0 ? (
                            <Button
                                variant="primary"
                                isLarge
                                component={LinkShim}
                                href={`${clustersInitBundlesPath}?action=create`}
                            >
                                Create bundle
                            </Button>
                        ) : (
                            <Button
                                variant="primary"
                                isLarge
                                onClick={() => {
                                    setIsModalOpen(true);
                                }}
                            >
                                Review installation methods
                            </Button>
                        )}
                        <div className="pf-u-mt-xl">
                            <Link to={`${clustersBasePath}/new`}>Legacy installation method</Link>
                        </div>
                        <SecureClusterModal
                            isModalOpen={isModalOpen}
                            setIsModalOpen={setIsModalOpen}
                        />
                    </EmptyState>
                )}
            </PageSection>
        </>
    );
    /* eslint-enable no-nested-ternary */
}

export default NoClustersPage;
