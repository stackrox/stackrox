import { useState } from 'react';
import useDeepCompareEffect from 'use-deep-compare-effect';
import { Bullseye } from '@patternfly/react-core';
import { ExclamationTriangleIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/EmptyStateTemplate/EmptyStateTemplate';
import TableHeader from 'Components/TableHeader';
import { PanelBody, PanelHead, PanelHeadEnd, PanelNew } from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
import { DEFAULT_PAGE_SIZE } from 'Components/Table';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import useURLPagination from 'hooks/useURLPagination';
import {
    fetchDeploymentsCount,
    fetchDeploymentsWithProcessInfo,
} from 'services/DeploymentsService';
import type { ListDeploymentWithProcessInfo } from 'services/DeploymentsService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { ORCHESTRATOR_COMPONENTS_KEY } from 'utils/orchestratorComponents';
import RiskTable from './RiskTable';

const sortFields = ['Deployment', 'Created', 'Cluster', 'Namespace', 'Deployment Risk Priority'];
const defaultSortOption = { field: 'Deployment Risk Priority', direction: 'asc' } as const;

type RiskTablePanelProps = {
    selectedDeploymentId: string | undefined;
    isViewFiltered: boolean;
    setIsViewFiltered: (isViewFiltered: boolean) => void;
};

function RiskTablePanel({
    selectedDeploymentId = undefined,
    isViewFiltered,
    setIsViewFiltered,
}: RiskTablePanelProps) {
    const { searchFilter } = useURLSearch();
    const { sortOption, setSortOption } = useURLSort({ sortFields, defaultSortOption });
    const { page, perPage, setPage } = useURLPagination(DEFAULT_PAGE_SIZE);

    const [currentDeployments, setCurrentDeployments] = useState<ListDeploymentWithProcessInfo[]>(
        []
    );
    const [errorMessageDeployments, setErrorMessageDeployments] = useState('');
    const [deploymentCount, setDeploymentsCount] = useState(0);

    const shouldHideOrchestratorComponents =
        localStorage.getItem(ORCHESTRATOR_COMPONENTS_KEY) !== 'true';

    useDeepCompareEffect(() => {
        const effectiveSearchFilter = {
            ...searchFilter,
            ...(shouldHideOrchestratorComponents ? { 'Orchestrator Component': 'false' } : {}),
        };
        const { request } = fetchDeploymentsWithProcessInfo(
            effectiveSearchFilter,
            sortOption,
            page,
            perPage
        );

        request.then(setCurrentDeployments).catch((error) => {
            setCurrentDeployments([]);
            setErrorMessageDeployments(getAxiosErrorMessage(error));
        });

        /*
         * Although count does not depend on change to sort option or page offset,
         * request in case of change to count of deployments in Kubernetes environment.
         */
        fetchDeploymentsCount(effectiveSearchFilter)
            .then(setDeploymentsCount)
            .catch(() => {
                setDeploymentsCount(0);
            });

        const hasSearchFilters = Object.keys(searchFilter).length > 0;
        setIsViewFiltered(hasSearchFilters);
    }, [searchFilter, sortOption, page, shouldHideOrchestratorComponents]);

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
                            iconClassName="pf-v6-u-warning-color-100"
                        >
                            {errorMessageDeployments}
                        </EmptyStateTemplate>
                    </Bullseye>
                ) : (
                    <RiskTable
                        currentDeployments={currentDeployments}
                        selectedDeploymentId={selectedDeploymentId}
                        setSortOption={(sortOption) => {
                            setSortOption(sortOption);
                            setPage(1);
                        }}
                    />
                )}
            </PanelBody>
        </PanelNew>
    );
}

export default RiskTablePanel;
