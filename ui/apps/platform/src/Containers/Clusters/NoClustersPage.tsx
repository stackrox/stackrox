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
    Toolbar,
    ToolbarContent,
} from '@patternfly/react-core';
import { CloudSecurityIcon } from '@patternfly/react-icons';

import LinkShim from 'Components/PatternFly/LinkShim';
import { getProductBranding } from 'constants/productBranding';
import useAnalytics, {
    CREATE_INIT_BUNDLE_CLICKED,
    LEGACY_SECURE_A_CLUSTER_LINK_CLICKED,
    SECURE_A_CLUSTER_LINK_CLICKED,
} from 'hooks/useAnalytics';
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
    const { analyticsTrack } = useAnalytics();

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

    // Why is some EmptyState content outside of EmptyStateBody element?
    // Because  Button is inside, it has same width at the text :(

    // TODO after 4.4 release add hasAdminRole to conditional rendering.
    /* eslint-disable no-nested-ternary */
    return (
        <>
            <PageSection variant="light">
                <Toolbar inset={{ default: 'insetNone' }} className="pf-u-pb-0">
                    <ToolbarContent>
                        <Title headingLevel="h1">Clusters</Title>
                    </ToolbarContent>
                </Toolbar>
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
                    <EmptyState variant="xl">
                        <EmptyStateIcon icon={CloudSecurityIcon} />
                        <Title headingLevel="h2">Secure clusters with a reusable init bundle</Title>
                        <EmptyStateBody>
                            <Flex
                                direction={{ default: 'column' }}
                                spaceItems={{ default: 'spaceItemsLg' }}
                            >
                                {initBundlesCount === 0 ? (
                                    <FlexItem>
                                        <Text component="p">
                                            {`You have successfully deployed a ${basePageTitle} platform.`}
                                        </Text>
                                        <Text component="p">
                                            Before you can secure clusters, create an init bundle.
                                        </Text>
                                    </FlexItem>
                                ) : (
                                    <FlexItem>
                                        <Text component="p">
                                            Use your preferred method to install secured cluster
                                            services.
                                        </Text>
                                        <Text component="p">
                                            After successful installation, it might take a few
                                            moments for this page to display secured clusters.
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
                                onClick={() =>
                                    analyticsTrack({
                                        event: CREATE_INIT_BUNDLE_CLICKED,
                                        properties: { source: 'No Clusters' },
                                    })
                                }
                            >
                                Create init bundle
                            </Button>
                        ) : (
                            <Button
                                variant="primary"
                                isLarge
                                onClick={() => {
                                    setIsModalOpen(true);
                                    analyticsTrack({
                                        event: SECURE_A_CLUSTER_LINK_CLICKED,
                                        properties: { source: 'No Clusters' },
                                    });
                                }}
                            >
                                View installation methods
                            </Button>
                        )}
                        <Flex direction={{ default: 'column' }} className="pf-u-mt-xl">
                            <Link
                                to={`${clustersBasePath}/new`}
                                onClick={() => {
                                    analyticsTrack({
                                        event: LEGACY_SECURE_A_CLUSTER_LINK_CLICKED,
                                        properties: { source: 'No Clusters' },
                                    });
                                }}
                            >
                                Legacy installation method
                            </Link>
                            {initBundlesCount !== 0 && (
                                <Text component="p">
                                    If you lost the YAML file that you downloaded,{' '}
                                    <Link
                                        to={`${clustersInitBundlesPath}?action=create`}
                                        onClick={() => {
                                            analyticsTrack({
                                                event: CREATE_INIT_BUNDLE_CLICKED,
                                                properties: { source: 'No Clusters' },
                                            });
                                        }}
                                    >
                                        create another init bundle
                                    </Link>
                                </Text>
                            )}
                        </Flex>
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
