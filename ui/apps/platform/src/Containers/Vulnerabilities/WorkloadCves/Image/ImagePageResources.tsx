import type { CSSProperties } from 'react';
import {
    Bullseye,
    Content,
    Divider,
    ExpandableSection,
    PageSection,
    Pagination,
    Spinner,
} from '@patternfly/react-core';
import { gql, useQuery } from '@apollo/client';
import type { DocumentNode } from '@apollo/client';
import type { Pagination as PaginationParam } from 'services/types';

import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import { overrideManagedColumns, useManagedColumns } from 'hooks/useManagedColumns';
import type { ColumnConfigOverrides } from 'hooks/useManagedColumns';
import type { UseURLPaginationResult } from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import useFeatureFlags from 'hooks/useFeatureFlags';

import { getPaginationParams, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import DeploymentResourceTable, {
    deploymentResourcesFragment,
    deploymentResourcesTableId,
    defaultColumns as deploymentResourcesDefaultColumns,
} from './DeploymentResourceTable';
import type { DeploymentResources } from './DeploymentResourceTable';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';

export type ImagePageResourcesProps = {
    imageId: string;
    pagination: UseURLPaginationResult;
    deploymentResourceColumnOverrides: ColumnConfigOverrides<
        keyof typeof deploymentResourcesDefaultColumns
    >;
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

const imageV2ResourcesQuery = gql`
    ${deploymentResourcesFragment}
    query getImageResources($id: ID!, $query: String, $pagination: Pagination) {
        imageV2(id: $id) {
            id
            digest
            ...DeploymentResources
        }
    }
`;

export const getImageResourcesQuery = (isNewImageDataModelEnabled: boolean): DocumentNode =>
    isNewImageDataModelEnabled ? imageV2ResourcesQuery : imageResourcesQuery;

function ImagePageResources({
    imageId,
    pagination,
    deploymentResourceColumnOverrides,
}: ImagePageResourcesProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isNewImageDataModelEnabled = isFeatureFlagEnabled('ROX_FLATTEN_IMAGE_DATA');
    const { baseSearchFilter } = useWorkloadCveViewContext();
    const { page, perPage, setPage, setPerPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: ['Deployment', 'Cluster', 'Namespace', 'Created'],
        defaultSortOption: { field: 'Deployment', direction: 'asc' },
        onSort: () => setPage(1),
    });

    const deploymentTableToggle = useSelectToggle(true);

    const { data, previousData, loading, error } = useQuery<
        {
            image: DeploymentResources | null; // Legacy image data model, will be null when ROX_FLATTEN_IMAGE_DATA is enabled
            imageV2: DeploymentResources | null; // New image data model, will be null when ROX_FLATTEN_IMAGE_DATA is disabled
        },
        { id: string; query: string; pagination: PaginationParam }
    >(getImageResourcesQuery(isNewImageDataModelEnabled), {
        variables: {
            id: imageId,
            query: getRequestQueryStringForSearchFilter(baseSearchFilter),
            pagination: getPaginationParams({ page, perPage, sortOption }),
        },
    });

    const imageResourcesData =
        (data && (isNewImageDataModelEnabled ? data.imageV2 : data.image)) ??
        (previousData && (isNewImageDataModelEnabled ? previousData.imageV2 : previousData.image));
    const deploymentCount = imageResourcesData?.deploymentCount ?? 0;

    const deploymentResourceColumnState = useManagedColumns(
        deploymentResourcesTableId,
        deploymentResourcesDefaultColumns
    );

    const deploymentResourceColumnConfig = overrideManagedColumns(
        deploymentResourceColumnState.columns,
        deploymentResourceColumnOverrides
    );

    return (
        <>
            <PageSection
                hasBodyWrapper={false}
                component="div"
                className="pf-v6-u-py-md pf-v6-u-px-xl"
            >
                <Content component="p">Navigate to resources associated with this image</Content>
            </PageSection>
            <Divider component="div" />
            <PageSection
                hasBodyWrapper={false}
                className="pf-v6-u-display-flex pf-v6-u-flex-direction-column pf-v6-u-flex-grow-1"
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
                                    'var(--pf-t--global--spacer--xs)',
                            } as CSSProperties
                        }
                    >
                        <div className="pf-v6-u-background-color-100 pf-v6-u-pt-sm">
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
                                columnVisibilityState={deploymentResourceColumnConfig}
                            />
                        </div>
                    </ExpandableSection>
                )}
            </PageSection>
        </>
    );
}

export default ImagePageResources;
