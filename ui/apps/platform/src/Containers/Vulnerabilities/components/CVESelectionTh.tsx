import React from 'react';
import { Th } from '@patternfly/react-table';

import useMap from 'hooks/useMap';

export type CVESelectionThProps<T extends { cve: string }> = {
    selectedCves: ReturnType<typeof useMap<string, T>>;
    className?: string;
};

function CVESelectionTh<T extends { cve: string }>({
    selectedCves,
    className,
}: CVESelectionThProps<T>) {
    return (
        <Th
            className={className}
            title={selectedCves.size > 0 ? `Clear selected CVEs` : undefined}
            select={{
                isSelected: selectedCves.size !== 0,
                isDisabled: selectedCves.size === 0,
                onSelect: selectedCves.clear,
            }}
        />
    );
}

export default CVESelectionTh;
