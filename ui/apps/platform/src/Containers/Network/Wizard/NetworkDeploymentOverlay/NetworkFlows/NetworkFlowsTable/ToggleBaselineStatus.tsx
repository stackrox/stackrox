import React, { ReactElement } from 'react';

import { networkFlowStatus } from 'constants/networkGraph';
import { CondensedButton, CondensedAlertButton } from '@stackrox/ui-components';
import { Row } from './tableTypes';

export type ToggleBaselineStatusProps = {
    row: Row;
};

function ToggleBaselineStatus({ row }: ToggleBaselineStatusProps): ReactElement {
    function onClick(): void {
        // TODO: remove this console log and add a way to use the API call
        // for marking as anomalous or adding to baseline
        // eslint-disable-next-line no-console
        console.log(row.original);
    }

    if (row.original.status === networkFlowStatus.ANOMALOUS) {
        return (
            <CondensedButton type="button" onClick={onClick}>
                Add to baseline
            </CondensedButton>
        );
    }
    return (
        <CondensedAlertButton type="button" onClick={onClick}>
            Mark as anomalous
        </CondensedAlertButton>
    );
}

export default ToggleBaselineStatus;
