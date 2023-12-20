import React from 'react';
import { Td } from '@patternfly/react-table';

import useMap from 'hooks/useMap';

export type CVESelectionTdProps = {
    selectedCves: ReturnType<
        typeof useMap<string, { cve: string; summary: string; numAffectedImages: number }>
    >;
    rowIndex: number;
    cve: string;
    summary: string;
    numAffectedImages: number;
};

function CVESelectionTd({
    selectedCves,
    rowIndex,
    cve,
    summary,
    numAffectedImages,
}: CVESelectionTdProps) {
    return (
        <Td
            select={{
                rowIndex,
                onSelect: () => {
                    if (selectedCves.has(cve)) {
                        selectedCves.remove(cve);
                    } else {
                        selectedCves.set(cve, { cve, summary, numAffectedImages });
                    }
                },
                isSelected: selectedCves.has(cve),
            }}
        />
    );
}

export default CVESelectionTd;
