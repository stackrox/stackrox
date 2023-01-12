import React, { ReactElement } from 'react';
import { DropdownItem } from '@patternfly/react-core';

import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';

type FlowsBulkActionsProps = {
    type: 'baseline' | 'active' | 'extraneous';
    selectedRows: string[];
    onClearSelectedRows: () => void;
    markSelectedAsAnomalous?: () => void;
    addSelectedToBaseline?: () => void;
};

function FlowsBulkActions({
    type,
    selectedRows,
    onClearSelectedRows,
    markSelectedAsAnomalous,
    addSelectedToBaseline,
}: FlowsBulkActionsProps): ReactElement {
    // setter functions
    const markSelectedAsAnomalousHandler = () => {
        // @TODO: Mark as anomalous
        markSelectedAsAnomalous?.();
        onClearSelectedRows();
    };
    const addSelectedToBaselineHandler = () => {
        // @TODO: Add to baseline
        addSelectedToBaseline?.();
        onClearSelectedRows();
    };

    return (
        <BulkActionsDropdown isDisabled={selectedRows.length === 0}>
            <DropdownItem
                key="mark_as_anomalous"
                component="button"
                onClick={markSelectedAsAnomalousHandler}
            >
                Mark as anomalous
            </DropdownItem>
            {type !== 'baseline' && (
                <DropdownItem
                    key="add_to_baseline"
                    component="button"
                    onClick={addSelectedToBaselineHandler}
                >
                    Add to baseline
                </DropdownItem>
            )}
        </BulkActionsDropdown>
    );
}

export default FlowsBulkActions;
