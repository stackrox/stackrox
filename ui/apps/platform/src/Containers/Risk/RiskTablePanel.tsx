import { useCallback, useContext, useState } from 'react';
import { useNavigate } from 'react-router-dom-v5-compat';
import useDeepCompareEffect from 'use-deep-compare-effect';
import { Bullseye } from '@patternfly/react-core';
import { ExclamationTriangleIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/EmptyStateTemplate/EmptyStateTemplate';
import TableHeader from 'Components/TableHeader';
import { PanelBody, PanelHead, PanelHeadEnd, PanelNew } from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
import { DEFAULT_PAGE_SIZE } from 'Components/Table';
import { pagingParams, sortParams } from 'constants/searchParams';
import workflowStateContext from 'Containers/workflowStateContext';
import useURLSearch from 'hooks/useURLSearch';
import {
    fetchDeploymentsCount,
    fetchDeploymentsWithProcessInfo,
} from 'services/DeploymentsService';
import type { ListDeploymentWithProcessInfo } from 'services/DeploymentsService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { convertSortToGraphQLFormat, convertSortToRestFormat } from 'utils/searchUtils';
import { ORCHESTRATOR_COMPONENTS_KEY } from 'utils/orchestratorComponents';
import RiskTable from './RiskTable';

const DEFAULT_RISK_SORT = [{ id: 'Deployment Risk Priority', desc: false }] as const;

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
    const navigate = useNavigate();
    const workflowState = useContext(workflowStateContext);
    const sortOption = workflowState.sort[sortParams.page] || DEFAULT_RISK_SORT;
    const currentPage = workflowState.paging[pagingParams.page];

    const { searchFilter } = useURLSearch();

    const [currentDeployments, setCurrentDeployments] = useState<ListDeploymentWithProcessInfo[]>(
        []
    );
    const [errorMessageDeployments, setErrorMessageDeployments] = useState('');
    const [deploymentCount, setDeploymentsCount] = useState(0);

    function setPage(newPage) {
        navigate(workflowState.setPage(newPage).toUrl());
    }
    const setSortOption = useCallback(
        (newSortOption) => {
            const convertedSortOption = convertSortToGraphQLFormat(newSortOption);

            const newUrl = workflowState.setSort(convertedSortOption).setPage(0).toUrl();

            navigate(newUrl);
        },
        [navigate, workflowState]
    );

    const restSort = convertSortToRestFormat(
        sortOption.length > 0 ? sortOption : DEFAULT_RISK_SORT
    );

    const shouldHideOrchestratorComponents =
        localStorage.getItem(ORCHESTRATOR_COMPONENTS_KEY) !== 'true';

    useDeepCompareEffect(() => {
        const effectiveSearchFilter = {
            ...searchFilter,
            ...(shouldHideOrchestratorComponents ? { 'Orchestrator Component': 'false' } : {}),
        };
        const { request } = fetchDeploymentsWithProcessInfo(
            effectiveSearchFilter,
            restSort,
            currentPage + 1, // Convert 0-based to 1-based page
            DEFAULT_PAGE_SIZE
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
    }, [searchFilter, restSort, currentPage, shouldHideOrchestratorComponents]);

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
                        page={currentPage}
                        dataLength={deploymentCount}
                        pageSize={DEFAULT_PAGE_SIZE}
                        setPage={setPage}
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
                        setSortOption={setSortOption}
                    />
                )}
            </PanelBody>
        </PanelNew>
    );
}

export default RiskTablePanel;
