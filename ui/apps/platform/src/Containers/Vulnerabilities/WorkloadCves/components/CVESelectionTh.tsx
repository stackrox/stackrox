import React from 'react';
import { pluralize } from '@patternfly/react-core';
import { Th } from '@patternfly/react-table';

import useMap from 'hooks/useMap';

export type CVESelectionThProps = {
    selectedCves: ReturnType<
        typeof useMap<string, { cve: string; summary: string; numAffectedImages: number }>
    >;
};

function CVESelectionTh({ selectedCves }: CVESelectionThProps) {
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
