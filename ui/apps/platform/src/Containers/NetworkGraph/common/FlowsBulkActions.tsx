import React, { ReactElement } from 'react';

import MenuDropdown from 'Components/PatternFly/MenuDropdown';
import { DropdownItem } from '@patternfly/react-core';

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
        <MenuDropdown toggleText="Bulk actions" isDisabled={selectedRows.length === 0}>
            <DropdownItem key="mark_as_anomalous" onClick={markSelectedAsAnomalousHandler}>
                Mark as anomalous
            </DropdownItem>
            {type !== 'baseline' && (
                <DropdownItem key="add_to_baseline" onClick={addSelectedToBaselineHandler}>
                    Add to baseline
                </DropdownItem>
            )}
        </MenuDropdown>
    );
}

export default FlowsBulkActions;
