import { useCallback } from 'react';

import {
    fetchComplianceAggregatedResults,
    fetchComplianceStandards,
} from 'services/ComplianceService';
import entityTypes from 'constants/entityTypes';
import useRestQuery from 'hooks/useRestQuery';

export type ComplianceStandard = {
    id: string;
    name: string;
};

export type AggregationResult = {
    controls: {
        results: {
            aggregationKeys: {
                id: string;
                scope: string;
            }[];
            numFailing: number;
            numPassing: number;
            numSkipped: number;
            unit: string;
        }[];
    };
    complianceStandards: ComplianceStandard[];
};

async function fetchComplianceLevelsByStandard(where: string): Promise<AggregationResult> {
    const [aggregated, standards] = await Promise.all([
        fetchComplianceAggregatedResults({
            groupBy: [entityTypes.STANDARD],
            unit: entityTypes.CONTROL,
            where,
        }),
        fetchComplianceStandards(),
    ]);

    return {
        controls: {
            results: aggregated.results ?? [],
        },
        complianceStandards: standards.map(({ id, name }) => ({ id, name })),
    };
}

export default function useComplianceLevelsByStandard(where: string) {
    const restQuery = useCallback(() => fetchComplianceLevelsByStandard(where), [where]);
    return useRestQuery(restQuery);
}
