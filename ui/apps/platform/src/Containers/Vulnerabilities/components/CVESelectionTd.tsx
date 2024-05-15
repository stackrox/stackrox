import React from 'react';
import { Td } from '@patternfly/react-table';

import useMap from 'hooks/useMap';

export type CVESelectionTdProps<T extends { cve: string }> = {
    selectedCves: ReturnType<typeof useMap<string, T>>;
    rowIndex: number;
    item: T;
};

function CVESelectionTd<T extends { cve: string }>({
    selectedCves,
    rowIndex,
    item,
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
            }}
        />
    );
}

export default CVESelectionTd;
