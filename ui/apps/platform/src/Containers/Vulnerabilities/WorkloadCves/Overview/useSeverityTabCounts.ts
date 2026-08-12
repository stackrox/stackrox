import { gql, useQuery } from '@apollo/client';

import { getRegexScopedQueryString } from '../../utils/searchUtils';
import { severityTabValues } from '../../types';
import type { QuerySearchFilter, SeverityTab } from '../../types';

const severityTabCountsQuery = gql`
    query getSeverityTabCounts(
        $criticalQuery: String
        $importantQuery: String
        $moderateQuery: String
        $lowQuery: String
        $unknownQuery: String
    ) {
        critical: imageCVECount(query: $criticalQuery)
        important: imageCVECount(query: $importantQuery)
        moderate: imageCVECount(query: $moderateQuery)
        low: imageCVECount(query: $lowQuery)
        unknown: imageCVECount(query: $unknownQuery)
    }
`;

const severityConfig: { tab: SeverityTab; queryValue: string; varKey: string; resKey: string }[] = [
    {
        tab: 'Critical',
        queryValue: 'CRITICAL_VULNERABILITY_SEVERITY',
        varKey: 'criticalQuery',
        resKey: 'critical',
    },
    {
        tab: 'Important',
        queryValue: 'IMPORTANT_VULNERABILITY_SEVERITY',
        varKey: 'importantQuery',
        resKey: 'important',
    },
    {
        tab: 'Moderate',
        queryValue: 'MODERATE_VULNERABILITY_SEVERITY',
        varKey: 'moderateQuery',
        resKey: 'moderate',
    },
    {
        tab: 'Low',
        queryValue: 'LOW_VULNERABILITY_SEVERITY',
        varKey: 'lowQuery',
        resKey: 'low',
    },
    {
        tab: 'Unknown',
        queryValue: 'UNKNOWN_VULNERABILITY_SEVERITY',
        varKey: 'unknownQuery',
        resKey: 'unknown',
    },
];

export function useSeverityTabCounts(
    baseSearchFilter: QuerySearchFilter,
    enabled: boolean
): Partial<Record<SeverityTab, number>> {
    const variables: Record<string, string> = {};
    severityConfig.forEach(({ queryValue, varKey }) => {
        const filterWithSeverity = {
            ...baseSearchFilter,
            Severity: [queryValue],
        };
        variables[varKey] = getRegexScopedQueryString(filterWithSeverity);
    });

    const { data } = useQuery<Record<string, number>>(severityTabCountsQuery, {
        variables,
        skip: !enabled,
    });

    if (!data) {
        return {};
    }

    const counts: Partial<Record<SeverityTab, number>> = {};
    severityTabValues.forEach((tab) => {
        const config = severityConfig.find((c) => c.tab === tab);
        if (config) {
            counts[tab] = data[config.resKey] ?? 0;
        }
    });
    return counts;
}
