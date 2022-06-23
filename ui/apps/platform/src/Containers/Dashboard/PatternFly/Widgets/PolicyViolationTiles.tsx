import React from 'react';
import { AlertCountBySeverity } from 'services/AlertsService';
import { SearchFilter } from 'types/search';

export type PolicyViolationTilesProps = {
    searchFilter: SearchFilter;
    counts?: AlertCountBySeverity[];
};

function PolicyViolationTiles({ searchFilter, counts }: PolicyViolationTilesProps) {
    return <>{JSON.stringify(counts)}</>;
}

export default PolicyViolationTiles;
