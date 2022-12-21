import { useMemo } from 'react';
import { gql, useQuery } from '@apollo/client';
import { getPolicyCategories } from 'services/PoliciesService';
import { SearchFilter } from 'types/search';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { policySeverities, PolicySeverity } from 'types/policy.proto';
import { AlertGroup } from 'services/AlertsService';
import useRestQuery from './useRestQuery';

// TODO Cancel is a noop
const categoryQuery = () => ({ request: getPolicyCategories(), cancel: () => {} });

function prepareGqlCountQuery(categories: string[], searchFilter: SearchFilter) {
    const queryParts: string[] = [];

    categories.forEach((cat) => {
        const gqlSafeCategory = cat.replace(/\W/g, '_');
        Object.values(policySeverities).forEach((severity) => {
            const query = getRequestQueryStringForSearchFilter({
                Category: cat,
                Severity: severity,
                ...searchFilter,
            });

            queryParts.push(`${gqlSafeCategory}_${severity}: violationCount(query: "${query}")`);
        });
    });

    const countQuery = gql`
        query violationCountsByPolicyCategory {
            violationCount
            ${queryParts.join('\n')}
        }
    `;

    return countQuery;
}

function prepareAlertGroupData(
    responseData: ViolationsByCategoryResponse | undefined
): AlertGroup[] | undefined {
    const groups: Record<string, AlertGroup['counts']> = {};
    if (!responseData) {
        return [];
    }
    Object.entries(responseData).forEach(([name, count]) => {
        const severity = policySeverities.find((sev) => name.includes(sev));
        if (severity) {
            const group = name.replace(`_${severity}`, '');

            if (!groups[group]) {
                groups[group] = [];
            }
            if (count > 0) {
                groups[group].push({ severity, count: `${count}` });
            }
        }
    });
    // TODO Need to map formatted category names back to display name
    return Object.entries(groups)
        .filter(([, counts]) => counts.length > 0)
        .map(([group, counts]) => ({ group, counts }));
}

export type ViolationsByCategoryResponse = Record<`${string}_${PolicySeverity}`, number>;

export default function useViolationsByCategory(searchFilter: SearchFilter) {
    const {
        data: categories,
        loading: isCategoriesLoading,
        error: categoriesError,
    } = useRestQuery(categoryQuery);
    const countQuery = useMemo(
        () => prepareGqlCountQuery(categories ?? [], searchFilter),
        [categories, searchFilter]
    );
    const {
        data: counts,
        loading: isCountsLoading,
        error: countsError,
    } = useQuery<ViolationsByCategoryResponse>(countQuery);

    const alertGroupData = prepareAlertGroupData(counts);

    return {
        data: alertGroupData,
        loading: isCategoriesLoading || isCountsLoading,
        error: categoriesError || countsError,
    };
}
