import { Bullseye } from '@patternfly/react-core';
import { ExclamationTriangleIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/EmptyStateTemplate/EmptyStateTemplate';
import TableHeader from 'Components/TableHeader';
import { PanelBody, PanelHead, PanelHeadEnd, PanelNew } from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
import type { UseURLPaginationResult } from 'hooks/useURLPagination';
import type { ApiSortOption, SearchFilter } from 'types/search';
import type { SortOption } from 'types/table';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import useDeploymentsCount from './useDeploymentsCount';
import useDeploymentsWithProcessInfo from './useDeploymentsWithProcessInfo';
import RiskTable from './RiskTable';

export const sortFields = [
    'Deployment',
    'Created',
    'Cluster',
    'Namespace',
    'Deployment Risk Priority',
];
export const defaultSortOption = { field: 'Deployment Risk Priority', direction: 'asc' } as const;

type RiskTablePanelProps = {
    selectedDeploymentId: string | undefined;
    isViewFiltered: boolean;
    sortOption: ApiSortOption;
    onSortOptionChange: (newSortOption: SortOption | SortOption[]) => void;
    searchFilter: SearchFilter;
    pagination: UseURLPaginationResult;
};

function RiskTablePanel({
    selectedDeploymentId = undefined,
    isViewFiltered,
    sortOption,
    onSortOptionChange,
    searchFilter,
    pagination,
}: RiskTablePanelProps) {
    const { page, perPage, setPage } = pagination;

    const { data, error } = useDeploymentsWithProcessInfo({
        searchFilter,
        sortOption,
        page,
        perPage,
    });
    const currentDeployments = data ?? [];

    const { data: deploymentCount = 0 } = useDeploymentsCount({
        searchFilter,
    });

    const errorMessageDeployments = error ? getAxiosErrorMessage(error) : '';

    return (
        <PanelNew testid="panel">
            <PanelHead>
                <TableHeader
                    length={deploymentCount}
                    type="deployment"
                    isViewFiltered={isViewFiltered}
                />
                <PanelHeadEnd>
                    <TablePagination
                        page={page - 1}
                        dataLength={deploymentCount}
                        pageSize={perPage}
                        setPage={(newPage) => setPage(newPage + 1)}
                    />
                </PanelHeadEnd>
            </PanelHead>
            <PanelBody>
                {errorMessageDeployments ? (
                    <Bullseye>
                        <EmptyStateTemplate
                            title="Unable to load deployments"
                            headingLevel="h2"
                            icon={ExclamationTriangleIcon}
                            iconClassName="pf-v5-u-warning-color-100"
                        >
                            {errorMessageDeployments}
                        </EmptyStateTemplate>
                    </Bullseye>
                ) : (
                    <RiskTable
                        currentDeployments={currentDeployments}
                        selectedDeploymentId={selectedDeploymentId}
                        setSortOption={onSortOptionChange}
                    />
                )}
            </PanelBody>
        </PanelNew>
    );
}

export default RiskTablePanel;
