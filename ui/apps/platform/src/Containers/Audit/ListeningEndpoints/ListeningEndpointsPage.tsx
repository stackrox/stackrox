import React from 'react';
import {
    Bullseye,
    Button,
    Divider,
    PageSection,
    Pagination,
    Spinner,
    Text,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate/EmptyStateTemplate';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useURLPagination from 'hooks/useURLPagination';
import { useDeploymentListeningEndpoints } from './hooks/useDeploymentListeningEndpoints';
import ListeningEndpointsTable from './ListeningEndpointsTable';

function ListeningEndpointsPage() {
    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { data, error, loading } = useDeploymentListeningEndpoints(page, perPage);

    return (
        <>
            <PageTitle title="Listening Endpoints" />
            <PageSection variant="light">
                <Title headingLevel="h1">Listening endpoints</Title>
            </PageSection>
            <Divider component="div" />
            <PageSection isFilled className="pf-u-display-flex pf-u-flex-direction-column">
                <Toolbar>
                    <ToolbarContent>
                        <ToolbarItem variant="pagination" alignment={{ default: 'alignRight' }}>
                            <Pagination
                                toggleTemplate={({ firstIndex, lastIndex }) => (
                                    <span>
                                        <b>
                                            {firstIndex} - {lastIndex}
                                        </b>{' '}
                                        of <b>many</b>
                                    </span>
                                )}
                                page={page}
                                perPage={perPage}
                                onSetPage={(_, newPage) => setPage(newPage)}
                                onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                            />
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>
                <div className="pf-u-background-color-100">
                    {error && (
                        <Bullseye>
                            <EmptyStateTemplate
                                title="Error loading deployments with listening endpoints"
                                headingLevel="h2"
                                icon={ExclamationCircleIcon}
                                iconClassName="pf-u-danger-color-100"
                            >
                                {getAxiosErrorMessage(error.message)}
                            </EmptyStateTemplate>
                        </Bullseye>
                    )}
                    {loading && (
                        <Bullseye>
                            <Spinner aria-label="Loading listening endpoints for deployments" />
                        </Bullseye>
                    )}
                    {!error && !loading && data && (
                        <>
                            {data.length === 0 ? (
                                <Bullseye>
                                    <EmptyStateTemplate
                                        title="No deployments with listening endpoints found"
                                        headingLevel="h2"
                                    >
                                        <Text>Clear any search value and try again</Text>
                                        <Button
                                            variant="link"
                                            onClick={() => {
                                                /* TODO */
                                            }}
                                        >
                                            Clear search
                                        </Button>
                                    </EmptyStateTemplate>
                                </Bullseye>
                            ) : (
                                <ListeningEndpointsTable deployments={data} />
                            )}
                        </>
                    )}
                </div>
            </PageSection>
        </>
    );
}

export default ListeningEndpointsPage;
