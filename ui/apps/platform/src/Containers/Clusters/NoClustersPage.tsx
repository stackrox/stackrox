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

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import LinkShim from 'Components/PatternFly/LinkShim';
import { getProductBranding } from 'constants/productBranding';
import useMetadata from 'hooks/useMetadata';
import { fetchClusterInitBundles } from 'services/ClustersService';
import { getVersionedDocs } from 'utils/versioning';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { clustersBasePath, clustersInitBundlesPath } from 'routePaths';

/*
 * Comments about data flow:
 *
 * 1. It is important that /main/clusters NoClustersPage **Create bundle**
 *    goes to /main/clusters/init-bundles InitBundlesWizard in the same tab,
 *    so when **Download** goes back, NoClustersPage makes a new GET /v1/init-bundles request
 *    and therefore renders the link instead of the button.
 *
 * 2. It is important that /main/clusters NoClustersPage **Review installation instructions**
 *    opens /main/clusters/secure-a-cluster SecureCluster in new tab,
 *    so polling loop in original tab will cause conditional rendering of table
 *    whenever there is a secured cluster.
 *    That is, if it opens the page in the same tab,
 *    then it suggests the need for a back button outside of a wizard.
 */

function NoClustersPage(): ReactElement {
    // Use promise instead of useRestQuery hook because of role-based access control.
    const [errorMessage, setErrorMessage] = useState('');
    const [initBundlesCount, setInitBundlesCount] = useState(0);
    const [isLoading, setIsLoading] = useState(false);

    const { version } = useMetadata();

    const { basePageTitle } = getProductBranding();
    const textForSuccessAlert = `You have successfully deployed a ${basePageTitle} platform. Now you can configure the clusters you want to secure.`;

    useEffect(() => {
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
    }, []);

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
                                        {version && (
                                            <>
                                                <ExternalLink>
                                                    <a
                                                        href={getVersionedDocs(
                                                            version,
                                                            'installing/installing_ocp/install-secured-cluster-ocp.html'
                                                        )}
                                                        target="_blank"
                                                        rel="noopener noreferrer"
                                                    >
                                                        Installing secured cluster services on Red
                                                        Hat OpenShift
                                                    </a>
                                                </ExternalLink>
                                                <ExternalLink>
                                                    <a
                                                        href={getVersionedDocs(
                                                            version,
                                                            'installing/installing_other/install-secured-cluster-other.html'
                                                        )}
                                                        target="_blank"
                                                        rel="noopener noreferrer"
                                                    >
                                                        Installing secured cluster services on other
                                                        platforms
                                                    </a>
                                                </ExternalLink>
                                            </>
                                        )}
                                    </FlexItem>
                                )}
                            </Flex>
                        </EmptyStateBody>
                        {initBundlesCount === 0 && (
                            <Button
                                variant="primary"
                                isLarge
                                component={LinkShim}
                                href={`${clustersInitBundlesPath}?action=create`}
                            >
                                Create bundle
                            </Button>
                        )}
                        <div className="pf-u-mt-xl">
                            <Link to={`${clustersBasePath}/new`}>Legacy installation method</Link>
                        </div>
                    </EmptyState>
                )}
            </PageSection>
        </>
    );
    /* eslint-enable no-nested-ternary */
}

export default NoClustersPage;
