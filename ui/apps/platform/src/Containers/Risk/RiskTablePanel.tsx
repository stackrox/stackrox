import { Label, Pagination, Toolbar, ToolbarContent, ToolbarItem } from '@patternfly/react-core';
import type { LabelProps } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import ColumnManagementButton from 'Components/ColumnManagementButton';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import type { UseURLPaginationResult } from 'hooks/useURLPagination';
import type { UseURLSortResult } from 'hooks/useURLSort';
import {
    generateVisibilityForColumns,
    getHiddenColumnCount,
    useManagedColumns,
} from 'hooks/useManagedColumns';
import type {
    RiskActivityClassification,
    RiskExposureClassification,
    RiskPostureClassification,
    RiskPriorityClassification,
} from 'types/risk.proto';
import type { ApiSortOption, SearchFilter } from 'types/search';
import { getDateTime } from 'utils/dateUtils';
import { getTableUIState } from 'utils/getTableUIState';

import { DeploymentNameColumn } from './DeploymentNameColumn';
import useDeploymentsCount from './useDeploymentsCount';
import useDeploymentsWithProcessInfo from './useDeploymentsWithProcessInfo';

export const sortFields = [
    'Deployment',
    'Created',
    'Cluster',
    'Namespace',
    'Risk Priority Classification',
    'Risk Activity Classification',
    'Risk Exposure Classification',
    'Risk Posture Classification',
    'Deployment Risk Score',
    'Proposed Deployment Risk Score',
    'Deployment Risk Priority',
];
export const defaultSortOption = {
    field: 'Risk Priority Classification',
    direction: 'asc',
} as const;

export const tableId = 'RiskTablePanel';

export const defaultColumns = {
    name: {
        title: 'Name',
        isShownByDefault: true,
        isUntoggleAble: true,
    },
    created: {
        title: 'Created',
        isShownByDefault: true,
    },
    cluster: {
        title: 'Cluster',
        isShownByDefault: true,
    },
    namespace: {
        title: 'Namespace',
        isShownByDefault: true,
    },
    priority: {
        title: 'Priority',
        isShownByDefault: true,
    },
    oldScore: {
        title: 'Old score',
        isShownByDefault: true,
    },
    newScore: {
        title: 'New score',
        isShownByDefault: false,
    },
    riskClass: {
        title: 'Risk class',
        isShownByDefault: true,
    },
    activity: {
        title: 'Activity',
        isShownByDefault: false,
    },
    exposure: {
        title: 'Exposure',
        isShownByDefault: false,
    },
    posture: {
        title: 'Posture',
        isShownByDefault: false,
    },
} as const;

const riskClassLabels: Record<RiskPriorityClassification, string> = {
    RISK_PRIORITY_UNKNOWN: 'Unknown',
    RISK_PRIORITY_RESPOND_NOW: 'Respond now',
    RISK_PRIORITY_INVESTIGATE_NOW: 'Investigate now',
    RISK_PRIORITY_REMEDIATE_NOW: 'Remediate now',
    RISK_PRIORITY_REMEDIATE: 'Remediate',
    RISK_PRIORITY_IMPROVE_POSTURE: 'Improve posture',
};

const riskClassLabelProps: Record<RiskPriorityClassification, LabelProps> = {
    RISK_PRIORITY_RESPOND_NOW: { status: 'danger' },
    RISK_PRIORITY_INVESTIGATE_NOW: { status: 'warning' },
    RISK_PRIORITY_REMEDIATE_NOW: { color: 'orangered' },
    RISK_PRIORITY_REMEDIATE: { color: 'orange' },
    RISK_PRIORITY_IMPROVE_POSTURE: { color: 'blue' },
    RISK_PRIORITY_UNKNOWN: { color: 'grey' },
};

const activityLabels: Record<RiskActivityClassification, string> = {
    RISK_ACTIVITY_UNKNOWN: 'Unknown',
    RISK_ACTIVITY_RESPOND_NOW: 'Respond now',
    RISK_ACTIVITY_INVESTIGATE_NOW: 'Investigate now',
    RISK_ACTIVITY_NONE_OBSERVED: 'None observed',
};

const activityLabelProps: Record<RiskActivityClassification, LabelProps> = {
    RISK_ACTIVITY_RESPOND_NOW: { status: 'danger' },
    RISK_ACTIVITY_INVESTIGATE_NOW: { status: 'warning' },
    RISK_ACTIVITY_NONE_OBSERVED: { color: 'grey' },
    RISK_ACTIVITY_UNKNOWN: { color: 'grey' },
};

const exposureLabels: Record<RiskExposureClassification, string> = {
    RISK_EXPOSURE_UNKNOWN: 'Unknown',
    RISK_EXPOSURE_REMEDIATE_NOW: 'Remediate now',
    RISK_EXPOSURE_REMEDIATE: 'Remediate',
    RISK_EXPOSURE_NONE_DETECTED: 'None detected',
};

const exposureLabelProps: Record<RiskExposureClassification, LabelProps> = {
    RISK_EXPOSURE_REMEDIATE_NOW: { color: 'orangered' },
    RISK_EXPOSURE_REMEDIATE: { color: 'orange' },
    RISK_EXPOSURE_NONE_DETECTED: { color: 'grey' },
    RISK_EXPOSURE_UNKNOWN: { color: 'grey' },
};

const postureLabels: Record<RiskPostureClassification, string> = {
    RISK_POSTURE_UNKNOWN: 'Unknown',
    RISK_POSTURE_CRITICAL: 'Critical',
    RISK_POSTURE_HIGH: 'High',
    RISK_POSTURE_MEDIUM: 'Medium',
    RISK_POSTURE_LOW: 'Low',
    RISK_POSTURE_NONE_DETECTED: 'None detected',
};

const postureLabelProps: Record<RiskPostureClassification, LabelProps> = {
    RISK_POSTURE_CRITICAL: { status: 'danger' },
    RISK_POSTURE_HIGH: { color: 'orangered' },
    RISK_POSTURE_MEDIUM: { color: 'orange' },
    RISK_POSTURE_LOW: { color: 'blue' },
    RISK_POSTURE_NONE_DETECTED: { color: 'grey' },
    RISK_POSTURE_UNKNOWN: { color: 'grey' },
};

type RiskTablePanelProps = {
    sortOption: ApiSortOption;
    getSortParams: UseURLSortResult['getSortParams'];
    searchFilter: SearchFilter;
    onSearchFilterChange: (newSearchFilter: SearchFilter) => void;
    pagination: UseURLPaginationResult;
};

function RiskTablePanel({
    sortOption,
    getSortParams,
    searchFilter,
    onSearchFilterChange,
    pagination,
}: RiskTablePanelProps) {
    const { page, perPage, setPage, setPerPage } = pagination;

    const { data, error, isLoading } = useDeploymentsWithProcessInfo({
        searchFilter,
        sortOption,
        page,
        perPage,
    });

    const { data: deploymentCount = 0 } = useDeploymentsCount({
        searchFilter,
    });

    const tableState = getTableUIState({ isLoading, data, error, searchFilter });
    const managedColumnState = useManagedColumns(tableId, defaultColumns);
    const getVisibilityClass = generateVisibilityForColumns(managedColumnState.columns);
    const colSpan =
        Object.values(defaultColumns).length - getHiddenColumnCount(managedColumnState.columns);

    return (
        <div>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem align={{ default: 'alignEnd' }}>
                        <ColumnManagementButton
                            columnConfig={managedColumnState.columns}
                            onApplyColumns={managedColumnState.setVisibility}
                        />
                    </ToolbarItem>
                    <ToolbarItem variant="pagination">
                        <Pagination
                            itemCount={deploymentCount}
                            page={page}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            perPage={perPage}
                            onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <Table variant="compact">
                <Thead noWrap>
                    <Tr>
                        <Th
                            className={getVisibilityClass('name')}
                            width={25}
                            sort={getSortParams('Deployment')}
                        >
                            Name
                        </Th>
                        <Th
                            className={getVisibilityClass('created')}
                            width={25}
                            sort={getSortParams('Created')}
                        >
                            Created
                        </Th>
                        <Th
                            className={getVisibilityClass('cluster')}
                            sort={getSortParams('Cluster')}
                        >
                            Cluster
                        </Th>
                        <Th
                            className={getVisibilityClass('namespace')}
                            sort={getSortParams('Namespace')}
                        >
                            Namespace
                        </Th>
                        <Th
                            className={getVisibilityClass('priority')}
                            width={10}
                            sort={getSortParams('Deployment Risk Priority')}
                        >
                            Priority
                        </Th>
                        <Th
                            className={getVisibilityClass('oldScore')}
                            width={10}
                            sort={getSortParams('Deployment Risk Score')}
                        >
                            Old score
                        </Th>
                        <Th
                            className={getVisibilityClass('newScore')}
                            width={10}
                            sort={getSortParams('Proposed Deployment Risk Score')}
                        >
                            New score
                        </Th>
                        <Th
                            className={getVisibilityClass('riskClass')}
                            width={10}
                            sort={getSortParams('Risk Priority Classification')}
                        >
                            Risk class
                        </Th>
                        <Th
                            className={getVisibilityClass('activity')}
                            width={10}
                            sort={getSortParams('Risk Activity Classification')}
                        >
                            Activity
                        </Th>
                        <Th
                            className={getVisibilityClass('exposure')}
                            width={10}
                            sort={getSortParams('Risk Exposure Classification')}
                        >
                            Exposure
                        </Th>
                        <Th
                            className={getVisibilityClass('posture')}
                            width={10}
                            sort={getSortParams('Risk Posture Classification')}
                        >
                            Posture
                        </Th>
                    </Tr>
                </Thead>
                <TbodyUnified
                    tableState={tableState}
                    colSpan={colSpan}
                    emptyProps={{ message: 'No results found' }}
                    filteredEmptyProps={{ onClearFilters: () => onSearchFilterChange({}) }}
                    renderer={({ data }) =>
                        data.map((deploymentWithProcessInfo) => {
                            const { deployment } = deploymentWithProcessInfo;

                            const riskClass =
                                deployment.riskPriorityClassification ?? 'RISK_PRIORITY_UNKNOWN';
                            const activity =
                                deployment.riskActivityClassification ?? 'RISK_ACTIVITY_UNKNOWN';
                            const exposure =
                                deployment.riskExposureClassification ?? 'RISK_EXPOSURE_UNKNOWN';
                            const posture =
                                deployment.riskPostureClassification ?? 'RISK_POSTURE_UNKNOWN';
                            const priorityAsInt = parseInt(deployment.priority, 10);
                            const priorityDisplay =
                                Number.isNaN(priorityAsInt) || priorityAsInt < 1
                                    ? 'N/A'
                                    : priorityAsInt;

                            return (
                                <Tbody key={deployment.id}>
                                    <Tr>
                                        <Td className={getVisibilityClass('name')} dataLabel="Name">
                                            <DeploymentNameColumn
                                                original={deploymentWithProcessInfo}
                                            />
                                        </Td>
                                        <Td
                                            className={getVisibilityClass('created')}
                                            dataLabel="Created"
                                        >
                                            {getDateTime(deployment.created)}
                                        </Td>
                                        <Td
                                            className={getVisibilityClass('cluster')}
                                            dataLabel="Cluster"
                                        >
                                            {deployment.cluster}
                                        </Td>
                                        <Td
                                            className={getVisibilityClass('namespace')}
                                            dataLabel="Namespace"
                                        >
                                            {deployment.namespace}
                                        </Td>
                                        <Td
                                            className={getVisibilityClass('priority')}
                                            dataLabel="Priority"
                                        >
                                            {priorityDisplay}
                                        </Td>
                                        <Td
                                            className={getVisibilityClass('oldScore')}
                                            dataLabel="Old score"
                                        >
                                            {deployment.riskScore.toFixed(1)}
                                        </Td>
                                        <Td
                                            className={getVisibilityClass('newScore')}
                                            dataLabel="New score"
                                        >
                                            {deployment.proposedRiskScore.toFixed(1)}
                                        </Td>
                                        <Td
                                            className={getVisibilityClass('riskClass')}
                                            dataLabel="Risk class"
                                        >
                                            <Label {...riskClassLabelProps[riskClass]}>
                                                {riskClassLabels[riskClass]}
                                            </Label>
                                        </Td>
                                        <Td
                                            className={getVisibilityClass('activity')}
                                            dataLabel="Activity"
                                        >
                                            <Label {...activityLabelProps[activity]}>
                                                {activityLabels[activity]}
                                            </Label>
                                        </Td>
                                        <Td
                                            className={getVisibilityClass('exposure')}
                                            dataLabel="Exposure"
                                        >
                                            <Label {...exposureLabelProps[exposure]}>
                                                {exposureLabels[exposure]}
                                            </Label>
                                        </Td>
                                        <Td
                                            className={getVisibilityClass('posture')}
                                            dataLabel="Posture"
                                        >
                                            <Label {...postureLabelProps[posture]}>
                                                {postureLabels[posture]}
                                            </Label>
                                        </Td>
                                    </Tr>
                                </Tbody>
                            );
                        })
                    }
                />
            </Table>
        </div>
    );
}

export default RiskTablePanel;
