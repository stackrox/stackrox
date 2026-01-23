import { useEffect, useState } from 'react';
import type { ReactElement } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import {
    Alert,
    Bullseye,
    Button,
    EmptyState,
    EmptyStateBody,
    EmptyStateFooter,
    EmptyStateHeader,
    EmptyStateIcon,
    Flex,
    FlexItem,
    PageSection,
    Spinner,
    Text,
} from '@patternfly/react-core';
import { CloudSecurityIcon } from '@patternfly/react-icons';

import LinkShim from 'Components/PatternFly/LinkShim';
import { getProductBranding } from 'constants/productBranding';
import useAnalytics, {
    CREATE_CLUSTER_REGISTRATION_SECRET_CLICKED,
    LEGACY_SECURE_A_CLUSTER_LINK_CLICKED,
    SECURE_A_CLUSTER_LINK_CLICKED,
    VIEW_INIT_BUNDLES_CLICKED,
} from 'hooks/useAnalytics';
// import useAuthStatus from 'hooks/useAuthStatus'; // TODO after 4.4 release
import { fetchClusterRegistrationSecrets } from 'services/ClustersService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import {
    clustersBasePath,
    clustersClusterRegistrationSecretsPath,
    clustersInitBundlesPath,
} from 'routePaths';

import SecureClusterModal from './ClusterRegistrationSecrets/SecureClusterModal';

const headingLevel = 'h1'; // Replace with h2 if refactoring restores h1 element with Clusters

/*
 * Comments about data flow:
 *
 * 1. It is important that /main/clusters NoClustersPage **Create cluster registration secret**
 *    goes to /main/clusters/cluster-registration-secrets ClusterRegistrationSecretForm in the same tab,
 *    so when **Download** goes back, NoClustersPage makes a new GET /v1/cluster-init/crs request
 *    and therefore renders the link instead of the button.
 *
 * 2. It is important that /main/clusters NoClustersPage **View installation methods**
 *    opens the modal SecureClusterModal in the same tab,
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
    const [registrationSecretsCount, setRegistrationSecretsCount] = useState(0);
    const [isLoading, setIsLoading] = useState(false);

    const { basePageTitle } = getProductBranding();

    useEffect(() => {
        // TODO after 4.4 release: if (hasAdminRole) {
        setIsLoading(true);
        fetchClusterRegistrationSecrets()
            .then(({ items }) => {
                setErrorMessage('');
                setRegistrationSecretsCount(items.length);
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
    return (
        <>
            <Alert
                variant="info"
                title="Upon successful installation, the secured clusters might take a few moments to show up."
                component="p"
                isInline
            />
            <PageSection variant="light">
                {isLoading ? (
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                ) : errorMessage ? (
                    <Alert
                        variant="warning"
                        title="Unable to fetch cluster registration secrets"
                        component="p"
                        isInline
                    >
                        {errorMessage}
                    </Alert>
                ) : (
                    <EmptyState variant="xl">
                        <EmptyStateHeader
                            titleText="Secure clusters with a registration secret"
                            icon={<EmptyStateIcon icon={CloudSecurityIcon} />}
                            headingLevel={headingLevel}
                        />
                        <EmptyStateBody>
                            <Flex
                                direction={{ default: 'column' }}
                                spaceItems={{ default: 'spaceItemsLg' }}
                            >
                                {registrationSecretsCount === 0 ? (
                                    <FlexItem>
                                        <Text component="p">
                                            {`You have successfully deployed a ${basePageTitle} platform.`}
                                        </Text>
                                        <Text component="p">
                                            Before you can secure clusters, create a registration
                                            secret.
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
                        <EmptyStateFooter>
                            {registrationSecretsCount === 0 ? (
                                <Button
                                    variant="primary"
                                    size="lg"
                                    component={LinkShim}
                                    href={`${clustersClusterRegistrationSecretsPath}?action=create`}
                                    onClick={() =>
                                        analyticsTrack({
                                            event: CREATE_CLUSTER_REGISTRATION_SECRET_CLICKED,
                                            properties: { source: 'No Clusters' },
                                        })
                                    }
                                >
                                    Create cluster registration secret
                                </Button>
                            ) : (
                                <Button
                                    variant="primary"
                                    size="lg"
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
                            <Flex direction={{ default: 'column' }} className="pf-v5-u-mt-xl">
                                <Link
                                    to={clustersInitBundlesPath}
                                    onClick={() => {
                                        analyticsTrack({
                                            event: VIEW_INIT_BUNDLES_CLICKED,
                                            properties: { source: 'No Clusters' },
                                        });
                                    }}
                                >
                                    Init bundles installation method
                                </Link>
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
                                {registrationSecretsCount !== 0 && (
                                    <Text component="p" className="pf-v5-u-w-50vw">
                                        If you misplaced your registration secret, we recommend
                                        locating the previously downloaded YAML on your device first
                                        by the name of the{' '}
                                        <Link to={clustersClusterRegistrationSecretsPath}>
                                            generated registration secret
                                        </Link>
                                        , or you may need to create a new registration secret.
                                    </Text>
                                )}
                            </Flex>
                            <SecureClusterModal
                                isModalOpen={isModalOpen}
                                setIsModalOpen={setIsModalOpen}
                            />
                        </EmptyStateFooter>
                    </EmptyState>
                )}
            </PageSection>
        </>
    );
}

export default NoClustersPage;
