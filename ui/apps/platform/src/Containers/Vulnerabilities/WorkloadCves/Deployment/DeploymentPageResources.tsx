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

import useURLSort from 'hooks/useURLSort';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import type { UseURLPaginationResult } from 'hooks/useURLPagination';
import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';

import { getPaginationParams, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import ImageResourceTable, { imageResourcesFragment } from './ImageResourceTable';
import type { ImageResources } from './ImageResourceTable';
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
            <PageSection component="div">
                <Content component="p">
                    Navigate to resources associated with this deployment
                </Content>
            </PageSection>
            <Divider component="div" />
            <PageSection component="div">
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
                    >
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
                    </ExpandableSection>
                )}
            </PageSection>
        </>
    );
}

export default DeploymentPageResources;
