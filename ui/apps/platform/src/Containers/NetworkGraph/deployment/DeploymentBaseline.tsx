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

import download from 'utils/download';
import { Deployment } from 'types/deployment.proto';
import { NetworkPolicyModification } from 'types/networkPolicy.proto';
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
import useFetchBaselineNetworkPolicy from '../api/useFetchBaselineNetworkPolicy';

type DeploymentBaselinesProps = {
    deployment: Deployment;
    deploymentId: string;
    onNodeSelect: (id: string) => void;
};

function DeploymentBaselines({ deployment, deploymentId, onNodeSelect }: DeploymentBaselinesProps) {
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
    const {
        isLoading: isLoadingNetworkPolicy,
        error: networkPolicyError,
        fetchBaselineNetworkPolicy,
    } = useFetchBaselineNetworkPolicy({
        deploymentId,
        includePorts: !isExcludingPortsAndProtocols,
    });

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
    const allUniquePorts = getAllUniquePorts(networkBaselines);
    const errorMessage = networkPolicyError || fetchError || modifyError || toggleError;

    const onSelectFlow = (entityId: string) => {
        onNodeSelect(entityId);
    };

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

    function downloadBaselineNetworkPolicy(baselineModification: NetworkPolicyModification) {
        const currentDateString = new Date().toISOString();
        download(
            `${deployment.name}-network-policy-${currentDateString}.yaml`,
            baselineModification.applyYaml,
            'yaml'
        );
    }

    function downloadBaselineNetworkPolicyHandler() {
        fetchBaselineNetworkPolicy(downloadBaselineNetworkPolicy);
    }

    if (isLoading || isModifying || isToggling) {
        return (
            <Bullseye>
                <Spinner isSVG size="lg" />
            </Bullseye>
        );
    }

    return (
        <div className="pf-u-h-100 pf-u-p-md">
            {errorMessage && (
                <Alert
                    isInline
                    variant={AlertVariant.danger}
                    title={errorMessage}
                    className="pf-u-mb-sm"
                />
            )}
            <Stack>
                <StackItem className="pf-u-pb-md">
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
                                        Trigger violations for traffic not in the baseline. This
                                        offers soak time for finalizing your network policies, where
                                        ACS can alert you on unexpected traffic, while that traffic
                                        is not blocked by a network policy.
                                    </div>
                                }
                            >
                                <HelpIcon className="pf-u-color-200" />
                            </Tooltip>
                        </FlexItem>
                    </Flex>
                </StackItem>
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
                <Divider component="hr" className="pf-u-py-md" />
                <StackItem className="pf-u-pb-md">
                    <Toolbar className="pf-u-p-0">
                        <ToolbarContent className="pf-u-px-0">
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
                        onSelectFlow={onSelectFlow}
                    />
                </StackItem>
                <StackItem className="pf-u-pt-md">
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
                            <Button
                                variant="primary"
                                onClick={downloadBaselineNetworkPolicyHandler}
                                isLoading={isLoadingNetworkPolicy}
                            >
                                Download baseline as network policy
                            </Button>
                        </FlexItem>
                    </Flex>
                </StackItem>
            </Stack>
        </div>
    );
}

export default DeploymentBaselines;
