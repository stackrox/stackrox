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
import { defaultImageSortFields, imagesDefaultSort } from '../sortUtils';
import TableErrorComponent from '../components/TableErrorComponent';
import ImageResourceTable, { ImageResources, imageResourcesFragment } from './ImageResourceTable';

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
    const { page, perPage, setPage, setPerPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: defaultImageSortFields,
        defaultSortOption: imagesDefaultSort,
        onSort: () => setPage(1),
    });

    const imageTableToggle = useSelectToggle(true);

    const { data, previousData, loading, error } = useQuery<
        { deployment: ImageResources | null },
        { id: string; query: string; pagination: PaginationParam }
    >(deploymentResourcesQuery, {
        variables: {
            id: deploymentId,
            query: '',
            pagination: {
                offset: (page - 1) * perPage,
                limit: perPage,
                sortOption,
            },
        },
    });

    const deploymentResourcesData = data?.deployment ?? previousData?.deployment;
    const imageCount = deploymentResourcesData?.imageCount ?? 0;

    return (
        <>
            <PageSection component="div" variant="light" className="pf-u-py-md pf-u-px-xl">
                <Text>Navigate to resources associated with this deployment</Text>
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
                {loading && !deploymentResourcesData && (
                    <Bullseye>
                        <Spinner isSVG />
                    </Bullseye>
                )}
                {deploymentResourcesData && (
                    <ExpandableSection
                        toggleText={`Images (${imageCount})`}
                        onToggle={imageTableToggle.onToggle}
                        isExpanded={imageTableToggle.isOpen}
                        style={
                            {
                                '--pf-c-expandable-section__content--MarginTop':
                                    'var(--pf-global--spacer--xs)',
                            } as CSSProperties
                        }
                    >
                        <div className="pf-u-background-color-100 pf-u-pt-sm">
                            <Pagination
                                itemCount={imageCount}
                                page={page}
                                perPage={perPage}
                                onSetPage={(_, newPage) => setPage(newPage)}
                                onPerPageSelect={(_, newPerPage) => {
                                    if (imageCount < (page - 1) * newPerPage) {
                                        setPage(1);
                                    }
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
