import React from 'react';
import { pluralize } from '@patternfly/react-core';
import { Th } from '@patternfly/react-table';

import useMap from 'hooks/useMap';

export type CVESelectionThProps<T extends { cve: string }> = {
    selectedCves: ReturnType<typeof useMap<string, T>>;
};

function CVESelectionTh<T extends { cve: string }>({ selectedCves }: CVESelectionThProps<T>) {
    return (
        <Th
            title={
                selectedCves.size > 0
                    ? `Clear ${pluralize(selectedCves.size, 'selected CVE')}`
                    : undefined
            }
            select={{
                isSelected: selectedCves.size !== 0,
                isDisabled: selectedCves.size === 0,
                onSelect: selectedCves.clear,
            }}
        />
    );
}

export default CVESelectionTh;
