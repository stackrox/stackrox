import React from 'react';

import {
    ExpandableSection,
    ExpandableSectionToggle,
    Stack,
    StackItem,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import pluralize from 'pluralize';

import { TimeWindow } from 'constants/timeWindows';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';

import FlowsTableHeaderText from '../common/FlowsTableHeaderText';
import { FlowTable } from '../components/FlowTable';
import { useNetworkBaselineStatus } from '../hooks/useNetworkBaselineStatus';

type ExternalFlowsProps = {
    deploymentId: string;
    timeWindow: TimeWindow;
};

function ExternalFlows({ deploymentId, timeWindow }: ExternalFlowsProps) {
    const anomalous = useNetworkBaselineStatus(deploymentId, timeWindow, 'ANOMALOUS');
    const baseline = useNetworkBaselineStatus(deploymentId, timeWindow, 'BASELINE');

    const { isOpen: isAnomalousFlowsExpanded, onToggle: toggleAnomalousFlowsExpandable } =
        useSelectToggle(true);
    const { isOpen: isBaselineFlowsExpanded, onToggle: toggleBaselineFlowsExpandable } =
        useSelectToggle(true);

    const totalAnomalous = anomalous.total;
    const totalBaseline = baseline.total;

    return (
        <Stack>
            <StackItem>
                <Toolbar className="pf-v5-u-p-0">
                    <ToolbarContent className="pf-v5-u-px-0">
                        <ToolbarItem>
                            <FlowsTableHeaderText
                                type={'total'}
                                numFlows={anomalous.total + baseline.total}
                            />
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>
            </StackItem>
            <StackItem>
                <Stack hasGutter>
                    <StackItem>
                        <ExpandableSectionToggle
                            isExpanded={isAnomalousFlowsExpanded}
                            onToggle={(isExpanded) => toggleAnomalousFlowsExpandable(isExpanded)}
                            toggleId={'anomalous-expandable-toggle'}
                            contentId={'anomalous-expandable-content'}
                        >
                            {`${totalAnomalous} anomalous ${pluralize('flow', totalAnomalous)}`}
                        </ExpandableSectionToggle>
                        <ExpandableSection
                            isExpanded={isAnomalousFlowsExpanded}
                            isDetached
                            toggleId={'anomalous-expandable-toggle'}
                            contentId={'anomalous-expandable-content'}
                        >
                            <FlowTable
                                pagination={anomalous.pagination}
                                flowCount={totalAnomalous}
                                emptyStateMessage="No anomalous flows."
                                tableState={anomalous.tableState}
                            />
                        </ExpandableSection>
                    </StackItem>
                    <StackItem>
                        <ExpandableSectionToggle
                            isExpanded={isBaselineFlowsExpanded}
                            onToggle={(isExpanded) => toggleBaselineFlowsExpandable(isExpanded)}
                            toggleId={'baseline-expandable-toggle'}
                            contentId={'baseline-expandable-content'}
                        >
                            {`${totalBaseline} baseline ${pluralize('flow', totalBaseline)}`}
                        </ExpandableSectionToggle>
                        <ExpandableSection
                            isDetached
                            toggleId={'baseline-expandable-toggle'}
                            contentId={'baseline-expandable-content'}
                            isExpanded={isBaselineFlowsExpanded}
                        >
                            <FlowTable
                                pagination={baseline.pagination}
                                flowCount={totalBaseline}
                                emptyStateMessage="No baseline flows."
                                tableState={baseline.tableState}
                            />
                        </ExpandableSection>
                    </StackItem>
                </Stack>
            </StackItem>
        </Stack>
    );
}

export default ExternalFlows;
