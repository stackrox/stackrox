import React, { CSSProperties } from 'react';
import {
    Bullseye,
    Divider,
    ExpandableSection,
    PageSection,
    Pagination,
    Spinner,
    Text,
} from '@patternfly/react-core';
import { gql, useQuery } from '@apollo/client';
import { Pagination as PaginationParam } from 'services/types';

import useURLSort from 'hooks/useURLSort';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';

import { getPaginationParams, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import ImageResourceTable, { ImageResources, imageResourcesFragment } from './ImageResourceTable';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';

export type DeploymentPageResourcesProps = {
    deploymentId: string;
    pagination: UseURLPaginationResult;
};

const deploymentResourcesQuery = gql`
    ${imageResourcesFragment}
    query getDeploymentResources($id: ID!, $query: String, $pagination: Pagination) {
        deployment(id: $id) {
            id
            ...ImageResources
        }
    }
`;

function DeploymentPageResources({ deploymentId, pagination }: DeploymentPageResourcesProps) {
    const { baseSearchFilter } = useWorkloadCveViewContext();
    const { page, perPage, setPage, setPerPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: ['Image'],
        defaultSortOption: { field: 'Image', direction: 'asc' },
        onSort: () => setPage(1),
    });

    const imageTableToggle = useSelectToggle(true);

    const { data, previousData, loading, error } = useQuery<
        { deployment: ImageResources | null },
        { id: string; query: string; pagination: PaginationParam }
    >(deploymentResourcesQuery, {
        variables: {
            id: deploymentId,
            query: getRequestQueryStringForSearchFilter(baseSearchFilter),
            pagination: getPaginationParams({ page, perPage, sortOption }),
        },
    });

    const deploymentResourcesData = data?.deployment ?? previousData?.deployment;
    const imageCount = deploymentResourcesData?.imageCount ?? 0;

    return (
        <>
            <PageSection component="div" variant="light" className="pf-v5-u-py-md pf-v5-u-px-xl">
                <Text>Navigate to resources associated with this deployment</Text>
            </PageSection>
            <Divider component="div" />
            <PageSection
                className="pf-v5-u-display-flex pf-v5-u-flex-direction-column pf-v5-u-flex-grow-1"
                component="div"
            >
                {error && (
                    <TableErrorComponent
                        error={error}
                        message="Adjust your filters and try again"
                    />
                )}
                {loading && !deploymentResourcesData && (
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                )}
                {deploymentResourcesData && (
                    <ExpandableSection
                        toggleText={`Images (${imageCount})`}
                        onToggle={() => imageTableToggle.onToggle(!imageTableToggle.isOpen)}
                        isExpanded={imageTableToggle.isOpen}
                        style={
                            {
                                '--pf-v5-c-expandable-section__content--MarginTop':
                                    'var(--pf-v5-global--spacer--xs)',
                            } as CSSProperties
                        }
                    >
                        <div className="pf-v5-u-background-color-100 pf-v5-u-pt-sm">
                            <Pagination
                                itemCount={imageCount}
                                page={page}
                                perPage={perPage}
                                onSetPage={(_, newPage) => setPage(newPage)}
                                onPerPageSelect={(_, newPerPage) => {
                                    setPerPage(newPerPage);
                                }}
                            />
                            <ImageResourceTable
                                data={deploymentResourcesData}
                                getSortParams={getSortParams}
                            />
                        </div>
                    </ExpandableSection>
                )}
            </PageSection>
        </>
    );
}

export default DeploymentPageResources;
