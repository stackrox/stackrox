import React, { useState } from 'react';
import {
    Alert,
    AlertVariant,
    Bullseye,
    Divider,
    Spinner,
    Stack,
    StackItem,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';

import { getNumFlows } from '../utils/flowUtils';

import FlowsTable from '../common/FlowsTable';
import FlowsTableHeaderText from '../common/FlowsTableHeaderText';
import { Flow } from '../types/flow.type';

type DeploymentBaselinesSimulatedProps = {
    deploymentId: string;
};

// eslint-disable-next-line @typescript-eslint/no-unused-vars
function DeploymentBaselinesSimulated({ deploymentId }: DeploymentBaselinesSimulatedProps) {
    // component state
    const networkSimulatedBaselines: Flow[] = [];
    const isLoading = false;
    const error = '';

    const initialExpandedRows = networkSimulatedBaselines
        .filter((row) => row.children && !!row.children.length)
        .map((row) => row.id); // Default to all expanded
    const [expandedRows, setExpandedRows] = useState<string[]>(initialExpandedRows);
    const [selectedRows, setSelectedRows] = useState<string[]>([]);

    // derived data
    const numBaselines = getNumFlows(networkSimulatedBaselines);

    if (isLoading) {
        return (
            <Bullseye>
                <Spinner isSVG size="lg" />
            </Bullseye>
        );
    }

    return (
        <div className="pf-u-h-100">
            {error && (
                <Alert
                    isInline
                    variant={AlertVariant.danger}
                    title={error}
                    className="pf-u-mb-sm"
                />
            )}
            <Stack hasGutter className="pf-u-p-md">
                <StackItem>
                    <Toolbar>
                        <ToolbarContent>
                            <ToolbarItem>
                                <FlowsTableHeaderText
                                    type="baseline simulated"
                                    numFlows={numBaselines}
                                />
                            </ToolbarItem>
                        </ToolbarContent>
                    </Toolbar>
                </StackItem>
                <Divider component="hr" />
                <StackItem>
                    <FlowsTable
                        label="Deployment simulated baselines"
                        flows={networkSimulatedBaselines}
                        numFlows={numBaselines}
                        expandedRows={expandedRows}
                        setExpandedRows={setExpandedRows}
                        selectedRows={selectedRows}
                        setSelectedRows={setSelectedRows}
                        isEditable={false}
                    />
                </StackItem>
            </Stack>
        </div>
    );
}

export default DeploymentBaselinesSimulated;
