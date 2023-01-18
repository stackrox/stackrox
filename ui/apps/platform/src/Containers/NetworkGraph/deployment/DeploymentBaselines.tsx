import React from 'react';
import {
    Alert,
    AlertVariant,
    Bullseye,
    Button,
    Checkbox,
    Divider,
    Flex,
    FlexItem,
    Spinner,
    Stack,
    StackItem,
    Switch,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';
import { HelpIcon } from '@patternfly/react-icons';

import { AdvancedFlowsFilterType } from '../common/AdvancedFlowsFilter/types';
import { filterNetworkFlows, getAllUniquePorts, getNumFlows } from '../utils/flowUtils';

import AdvancedFlowsFilter, {
    defaultAdvancedFlowsFilters,
} from '../common/AdvancedFlowsFilter/AdvancedFlowsFilter';
import EntityNameSearchInput from '../common/EntityNameSearchInput';
import FlowsTable from '../common/FlowsTable';
import FlowsTableHeaderText from '../common/FlowsTableHeaderText';
import FlowsBulkActions from '../common/FlowsBulkActions';
import useFetchNetworkBaselines from '../api/useFetchNetworkBaselines';
import { Flow } from '../types/flow.type';
import useModifyBaselineStatuses from '../api/useModifyBaselineStatuses';
import useToggleAlertingOnBaselineViolation from '../api/useToggleAlertingOnBaselineViolation';

type DeploymentBaselinesProps = {
    deploymentId: string;
};

function DeploymentBaselines({ deploymentId }: DeploymentBaselinesProps) {
    // component state
    const [isExcludingPortsAndProtocols, setIsExcludingPortsAndProtocols] =
        React.useState<boolean>(false);

    const [entityNameFilter, setEntityNameFilter] = React.useState<string>('');
    const [advancedFilters, setAdvancedFilters] = React.useState<AdvancedFlowsFilterType>(
        defaultAdvancedFlowsFilters
    );
    const {
        isLoading,
        error: fetchError,
        data: { networkBaselines, isAlertingOnBaselineViolation },
        refetchBaselines,
    } = useFetchNetworkBaselines(deploymentId);
    const {
        isModifying,
        error: modifyError,
        modifyBaselineStatuses,
    } = useModifyBaselineStatuses(deploymentId);
    const {
        isToggling,
        error: toggleError,
        toggleAlertingOnBaselineViolation,
    } = useToggleAlertingOnBaselineViolation(deploymentId);
    const filteredNetworkBaselines = filterNetworkFlows(
        networkBaselines,
        entityNameFilter,
        advancedFilters
    );

    const initialExpandedRows = filteredNetworkBaselines
        .filter((row) => row.children && !!row.children.length)
        .map((row) => row.id); // Default to all expanded
    const [expandedRows, setExpandedRows] = React.useState<string[]>(initialExpandedRows);
    const [selectedRows, setSelectedRows] = React.useState<string[]>([]);

    // derived data
    const numBaselines = getNumFlows(filteredNetworkBaselines);
    const allUniquePorts = getAllUniquePorts(filteredNetworkBaselines);

    function addToBaseline(flow: Flow) {
        modifyBaselineStatuses([flow], 'BASELINE', refetchBaselines);
    }

    function markAsAnomalous(flow: Flow) {
        modifyBaselineStatuses([flow], 'ANOMALOUS', refetchBaselines);
    }

    function addSelectedToBaseline() {
        const selectedFlows = filteredNetworkBaselines.filter((networkBaseline) => {
            return selectedRows.includes(networkBaseline.id);
        });
        modifyBaselineStatuses(selectedFlows, 'BASELINE', refetchBaselines);
    }

    function markSelectedAsAnomalous() {
        const selectedFlows = filteredNetworkBaselines.filter((networkBaseline) => {
            return selectedRows.includes(networkBaseline.id);
        });
        modifyBaselineStatuses(selectedFlows, 'ANOMALOUS', refetchBaselines);
    }

    function toggleAlertingOnBaselineViolationHandler() {
        toggleAlertingOnBaselineViolation(!isAlertingOnBaselineViolation, refetchBaselines);
    }

    if (isLoading || isModifying || isToggling) {
        return (
            <Bullseye>
                <Spinner isSVG size="lg" />
            </Bullseye>
        );
    }

    return (
        <div className="pf-u-h-100">
            {(fetchError || modifyError || toggleError) && (
                <Alert
                    isInline
                    variant={AlertVariant.danger}
                    title={fetchError || modifyError || toggleError}
                    className="pf-u-mb-sm"
                />
            )}
            <Stack hasGutter className="pf-u-p-md">
                <StackItem>
                    <Flex alignItems={{ default: 'alignItemsCenter' }}>
                        <FlexItem>
                            <Switch
                                id="simple-switch"
                                label="Alert on baseline violation"
                                isChecked={isAlertingOnBaselineViolation}
                                onChange={toggleAlertingOnBaselineViolationHandler}
                                isDisabled={isLoading || isModifying || isToggling}
                            />
                        </FlexItem>
                        <FlexItem>
                            <Tooltip
                                content={
                                    <div>
                                        Trigger violations for network policies not in the baseline
                                    </div>
                                }
                            >
                                <HelpIcon className="pf-u-color-200" />
                            </Tooltip>
                        </FlexItem>
                    </Flex>
                </StackItem>
                <Divider component="hr" />
                <StackItem>
                    <Flex>
                        <FlexItem flex={{ default: 'flex_1' }}>
                            <EntityNameSearchInput
                                value={entityNameFilter}
                                setValue={setEntityNameFilter}
                            />
                        </FlexItem>
                        <FlexItem>
                            <AdvancedFlowsFilter
                                filters={advancedFilters}
                                setFilters={setAdvancedFilters}
                                allUniquePorts={allUniquePorts}
                            />
                        </FlexItem>
                    </Flex>
                </StackItem>
                <Divider component="hr" />
                <StackItem>
                    <Toolbar>
                        <ToolbarContent>
                            <ToolbarItem>
                                <FlowsTableHeaderText type="baseline" numFlows={numBaselines} />
                            </ToolbarItem>
                            <ToolbarItem alignment={{ default: 'alignRight' }}>
                                <FlowsBulkActions
                                    type="baseline"
                                    selectedRows={selectedRows}
                                    onClearSelectedRows={() => setSelectedRows([])}
                                    markSelectedAsAnomalous={markSelectedAsAnomalous}
                                    addSelectedToBaseline={addSelectedToBaseline}
                                />
                            </ToolbarItem>
                        </ToolbarContent>
                    </Toolbar>
                </StackItem>
                <Divider component="hr" />
                <StackItem>
                    <FlowsTable
                        label="Deployment baselines"
                        flows={filteredNetworkBaselines}
                        numFlows={numBaselines}
                        expandedRows={expandedRows}
                        setExpandedRows={setExpandedRows}
                        selectedRows={selectedRows}
                        setSelectedRows={setSelectedRows}
                        addToBaseline={addToBaseline}
                        markAsAnomalous={markAsAnomalous}
                        isEditable
                    />
                </StackItem>
                <StackItem>
                    <Flex
                        className="pf-u-pb-md"
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsMd' }}
                        alignItems={{ default: 'alignItemsCenter' }}
                        justifyContent={{ default: 'justifyContentCenter' }}
                    >
                        <FlexItem>
                            <Checkbox
                                id="exclude-ports-and-protocols-checkbox"
                                label="Exclude ports & protocols"
                                isChecked={isExcludingPortsAndProtocols}
                                onChange={setIsExcludingPortsAndProtocols}
                            />
                        </FlexItem>
                        <FlexItem>
                            <Button variant="primary">Simulate baseline as network policy</Button>
                        </FlexItem>
                    </Flex>
                </StackItem>
            </Stack>
        </div>
    );
}

export default DeploymentBaselines;
