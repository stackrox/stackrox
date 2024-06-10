import React from 'react';
import { Td } from '@patternfly/react-table';

import useMap from 'hooks/useMap';

export type CVESelectionTdProps<T extends { cve: string }> = {
    selectedCves: ReturnType<typeof useMap<string, T>>;
    rowIndex: number;
    item: T;
    isDisabled?: boolean;
};

function CVESelectionTd<T extends { cve: string }>({
    selectedCves,
    rowIndex,
    item,
    isDisabled,
}: CVESelectionTdProps<T>) {
    const { cve } = item;
    return (
        <Td
            select={{
                rowIndex,
                onSelect: () => {
                    if (selectedCves.has(cve)) {
                        selectedCves.remove(cve);
                    } else {
                        selectedCves.set(cve, item);
                    }
                },
                isSelected: selectedCves.has(cve),
                isDisabled,
            }}
        />
    );
}

export default CVESelectionTd;
