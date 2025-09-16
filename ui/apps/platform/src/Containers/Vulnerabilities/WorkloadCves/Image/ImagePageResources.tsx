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

import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';

import { getPaginationParams, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import DeploymentResourceTable, {
    DeploymentResources,
    deploymentResourcesFragment,
} from './DeploymentResourceTable';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';

export type ImagePageResourcesProps = {
    imageId: string;
    pagination: UseURLPaginationResult;
};

const imageResourcesQuery = gql`
    ${deploymentResourcesFragment}
    query getImageResources($id: ID!, $query: String, $pagination: Pagination) {
        image(id: $id) {
            id
            ...DeploymentResources
        }
    }
`;

function ImagePageResources({ imageId, pagination }: ImagePageResourcesProps) {
    const { baseSearchFilter } = useWorkloadCveViewContext();
    const { page, perPage, setPage, setPerPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: ['Deployment', 'Cluster', 'Namespace', 'Created'],
        defaultSortOption: { field: 'Deployment', direction: 'asc' },
        onSort: () => setPage(1),
    });

    const deploymentTableToggle = useSelectToggle(true);

    const { data, previousData, loading, error } = useQuery<
        { image: DeploymentResources | null },
        { id: string; query: string; pagination: PaginationParam }
    >(imageResourcesQuery, {
        variables: {
            id: imageId,
            query: getRequestQueryStringForSearchFilter(baseSearchFilter),
            pagination: getPaginationParams({ page, perPage, sortOption }),
        },
    });

    const imageResourcesData = data?.image ?? previousData?.image;
    const deploymentCount = imageResourcesData?.deploymentCount ?? 0;

    return (
        <>
            <PageSection component="div" variant="light" className="pf-v5-u-py-md pf-v5-u-px-xl">
                <Text>Navigate to resources associated with this image</Text>
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
                {loading && !imageResourcesData && (
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                )}
                {imageResourcesData && (
                    <ExpandableSection
                        toggleText={`Deployments (${deploymentCount})`}
                        onToggle={() =>
                            deploymentTableToggle.onToggle(!deploymentTableToggle.isOpen)
                        }
                        isExpanded={deploymentTableToggle.isOpen}
                        style={
                            {
                                '--pf-v5-c-expandable-section__content--MarginTop':
                                    'var(--pf-v5-global--spacer--xs)',
                            } as CSSProperties
                        }
                    >
                        <div className="pf-v5-u-background-color-100 pf-v5-u-pt-sm">
                            <Pagination
                                itemCount={deploymentCount}
                                page={page}
                                perPage={perPage}
                                onSetPage={(_, newPage) => setPage(newPage)}
                                onPerPageSelect={(_, newPerPage) => {
                                    setPerPage(newPerPage);
                                }}
                            />
                            <DeploymentResourceTable
                                data={imageResourcesData}
                                getSortParams={getSortParams}
                            />
                        </div>
                    </ExpandableSection>
                )}
            </PageSection>
        </>
    );
}

export default ImagePageResources;
