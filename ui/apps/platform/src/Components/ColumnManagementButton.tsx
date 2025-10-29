import { useState } from 'react';

import { ColumnManagementModal } from '@patternfly/react-component-groups';
import { Button } from '@patternfly/react-core';
import { CogIcon } from '@patternfly/react-icons';

import type { ColumnConfig } from 'hooks/useManagedColumns';

export type ColumnManagementButtonProps<ColumnKey extends string> = {
    columnConfig: Record<ColumnKey, ColumnConfig>;
    onApplyColumns: (columns: Record<string, boolean>) => void;
};

function ColumnManagementButton<ColumnKey extends string>({
    columnConfig,
    onApplyColumns,
}: ColumnManagementButtonProps<ColumnKey>) {
    const [isOpen, setOpen] = useState(false);
    const enabledColumnCount = Object.values<ColumnConfig>(columnConfig).filter(
        ({ isShown, isUntoggleAble }) => isShown && !isUntoggleAble
    ).length;

    return (
        <>
            <ColumnManagementModal
                appliedColumns={Object.values<ColumnConfig>(columnConfig).filter(
                    (c) => !c.isUntoggleAble
                )}
                applyColumns={(newColumns) => {
                    const nextState = Object.fromEntries(
                        newColumns.map(({ key, isShown }) => [key, isShown ?? false])
                    );
                    onApplyColumns(nextState);
                }}
                isOpen={isOpen}
                onClose={() => setOpen(false)}
            />
            <Button
                icon={<CogIcon />}
                onClick={() => setOpen(true)}
                variant="secondary"
                countOptions={{
                    isRead: true,
                    count: enabledColumnCount,
                }}
            >
                Columns
            </Button>
        </>
    );
}

export default ColumnManagementButton;
