import React, { ReactElement } from 'react';
import { DropdownItem } from '@patternfly/react-core';

import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';

type FlowsBulkActionsProps = {
    type: 'baseline' | 'active' | 'extraneous';
    selectedRows: string[];
    onClearSelectedRows: () => void;
};

function FlowsBulkActions({
    type,
    selectedRows,
    onClearSelectedRows,
}: FlowsBulkActionsProps): ReactElement {
    // setter functions
    const markSelectedAsAnomalous = () => {
        // @TODO: Mark as anomalous
        onClearSelectedRows();
    };
    const addSelectedToBaseline = () => {
        // @TODO: Add to baseline
        onClearSelectedRows();
    };

    return (
        <BulkActionsDropdown isDisabled={selectedRows.length === 0}>
            {type !== 'baseline' && (
                <DropdownItem
                    key="mark_as_anomalous"
                    component="button"
                    onClick={markSelectedAsAnomalous}
                >
                    Mark as anomalous
                </DropdownItem>
            )}
            <DropdownItem key="add_to_baseline" component="button" onClick={addSelectedToBaseline}>
                Add to baseline
            </DropdownItem>
        </BulkActionsDropdown>
    );
}

export default FlowsBulkActions;
