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
import type { Pagination as PaginationParam } from 'services/types';

import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import { overrideManagedColumns, useManagedColumns } from 'hooks/useManagedColumns';
import type { ColumnConfigOverrides } from 'hooks/useManagedColumns';
import type { UseURLPaginationResult } from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';

import { getPaginationParams, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import DeploymentResourceTable, {
    deploymentResourcesV2Fragment,
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

const imageV2ResourcesQuery = gql`
    ${deploymentResourcesV2Fragment}
    query getImageResources($id: ID!, $query: String, $pagination: Pagination) {
        imageV2(id: $id) {
            id
            digest
            ...DeploymentResourcesV2
        }
    }
`;

function ImagePageResources({
    imageId,
    pagination,
    deploymentResourceColumnOverrides,
}: ImagePageResourcesProps) {
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
            imageV2: DeploymentResources | null;
        },
        { id: string; query: string; pagination: PaginationParam }
    >(imageV2ResourcesQuery, {
        variables: {
            id: imageId,
            query: getRequestQueryStringForSearchFilter(baseSearchFilter),
            pagination: getPaginationParams({ page, perPage, sortOption }),
        },
    });

    const imageResourcesData = data?.imageV2 ?? previousData?.imageV2;
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
            <PageSection component="div">
                <Content component="p">Navigate to resources associated with this image</Content>
            </PageSection>
            <Divider component="div" />
            <PageSection component="div">
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
                    >
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
                    </ExpandableSection>
                )}
            </PageSection>
        </>
    );
}

export default ImagePageResources;
