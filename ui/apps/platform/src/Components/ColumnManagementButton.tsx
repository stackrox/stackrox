import React, { useState } from 'react';

import { ColumnManagementModal } from '@patternfly/react-component-groups';
import { Button } from '@patternfly/react-core';

import { ColumnConfig, ManagedColumns } from 'hooks/useManagedColumns';

export type ColumnManagementButtonProps<ColumnKey extends string> = {
    managedColumnState: ManagedColumns<ColumnKey>;
};

function ColumnManagementButton<ColumnKey extends string>({
    managedColumnState,
}: ColumnManagementButtonProps<ColumnKey>) {
    const [isOpen, setOpen] = useState(false);
    const { columns, setColumns } = managedColumnState;
    const enabledColumnCount = Object.values<ColumnConfig>(columns).filter(
        ({ isShown }) => isShown
    ).length;

    return (
        <>
            <ColumnManagementModal
                appliedColumns={Object.values(columns)}
                applyColumns={(newColumns) => {
                    const nextState = Object.fromEntries(
                        newColumns.map(({ key, isShown }) => [key, isShown ?? false])
                    );
                    setColumns(nextState);
                }}
                isOpen={isOpen}
                onClose={() => setOpen(false)}
            />
            <Button
                onClick={() => setOpen(true)}
                variant="secondary"
                countOptions={{
                    isRead: true,
                    count: enabledColumnCount,
                }}
            >
                Manage columns
            </Button>
        </>
    );
}

export default ColumnManagementButton;
