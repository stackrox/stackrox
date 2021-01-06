import React, { ReactElement } from 'react';

import { networkFlowStatus } from 'constants/networkGraph';
import { FlattenedNetworkBaseline } from 'Containers/Network/networkTypes';

import { CondensedButton, CondensedAlertButton } from '@stackrox/ui-components';

import { Row } from './tableTypes';

export type ToggleBaselineStatusProps = {
    row: Row;
    toggleBaselineStatuses: (networkBaselines: FlattenedNetworkBaseline[]) => void;
};

function ToggleBaselineStatus({
    row,
    toggleBaselineStatuses,
}: ToggleBaselineStatusProps): ReactElement {
    function onClickHandler(): void {
        toggleBaselineStatuses([row.original]);
    }

    if (row.original.status === networkFlowStatus.ANOMALOUS) {
        return (
            <CondensedButton type="button" onClick={onClickHandler}>
                Add to baseline
            </CondensedButton>
        );
    }
    return (
        <CondensedAlertButton type="button" onClick={onClickHandler}>
            Mark as anomalous
        </CondensedAlertButton>
    );
}

export default ToggleBaselineStatus;
