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

import { UseURLPaginationResult } from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { deploymentsDefaultSort, defaultDeploymentSortFields } from '../sortUtils';
import TableErrorComponent from '../components/TableErrorComponent';
import DeploymentResourceTable, {
    DeploymentResources,
    deploymentResourcesFragment,
} from './DeploymentResourceTable';

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
    const { page, perPage, setPage, setPerPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: defaultDeploymentSortFields,
        defaultSortOption: deploymentsDefaultSort,
        onSort: () => setPage(1),
    });

    const deploymentTableToggle = useSelectToggle(true);

    const { data, previousData, loading, error } = useQuery<
        { image: DeploymentResources | null },
        { id: string; query: string; pagination: PaginationParam }
    >(imageResourcesQuery, {
        variables: {
            id: imageId,
            query: '',
            pagination: {
                offset: (page - 1) * perPage,
                limit: perPage,
                sortOption,
            },
        },
    });

    const imageResourcesData = data?.image ?? previousData?.image;
    const deploymentCount = imageResourcesData?.deploymentCount ?? 0;

    return (
        <>
            <PageSection component="div" variant="light" className="pf-u-py-md pf-u-px-xl">
                <Text>Navigate to resources associated with this image</Text>
            </PageSection>
            <Divider component="div" />
            <PageSection
                className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1"
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
                        <Spinner isSVG />
                    </Bullseye>
                )}
                {imageResourcesData && (
                    <ExpandableSection
                        toggleText={`Deployments (${deploymentCount})`}
                        onToggle={deploymentTableToggle.onToggle}
                        isExpanded={deploymentTableToggle.isOpen}
                        style={
                            {
                                '--pf-c-expandable-section__content--MarginTop':
                                    'var(--pf-global--spacer--xs)',
                            } as CSSProperties
                        }
                    >
                        <div className="pf-u-background-color-100 pf-u-pt-sm">
                            <Pagination
                                itemCount={deploymentCount}
                                page={page}
                                perPage={perPage}
                                onSetPage={(_, newPage) => setPage(newPage)}
                                onPerPageSelect={(_, newPerPage) => {
                                    if (deploymentCount < (page - 1) * newPerPage) {
                                        setPage(1);
                                    }
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
