import React from 'react';
import { Bullseye, Divider, PageSection, Spinner, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate/EmptyStateTemplate';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import { useDeploymentListeningEndpoints } from './hooks/useDeploymentListeningEndpoints';
import ListeningEndpointsTable from './ListeningEndpointsTable';

function ListeningEndpointsPage() {
    const { data, lastFetchError, isFetchingNextPage, isEndOfResults } =
        useDeploymentListeningEndpoints();
    const isInitialLoad =
        data.length === 0 && !lastFetchError && isFetchingNextPage && !isEndOfResults;
    const deployments = data
        .flat()
        .filter((deployment) => deployment.listeningEndpoints.length > 0);

    return (
        <>
            <PageTitle title="Listening Endpoints" />
            <PageSection variant="light">
                <Title headingLevel="h1">Listening endpoints</Title>
            </PageSection>
            <Divider component="div" />
            <PageSection isFilled>
                {lastFetchError && (
                    <div className="pf-u-background-color-100">
                        <EmptyStateTemplate
                            title="Error loading deployments with listening endpoints"
                            headingLevel="h2"
                            icon={ExclamationCircleIcon}
                            iconClassName="pf-u-danger-color-100"
                        >
                            {lastFetchError.message}
                        </EmptyStateTemplate>
                    </div>
                )}
                {isInitialLoad && (
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                )}
                {!lastFetchError && !isInitialLoad && (
                    <>
                        {deployments.length === 0 ? (
                            <Title headingLevel="h2">
                                No deployments with listening endpoints found
                            </Title>
                        ) : (
                            <ListeningEndpointsTable deployments={deployments} />
                        )}
                    </>
                )}
            </PageSection>
        </>
    );
}

export default ListeningEndpointsPage;
