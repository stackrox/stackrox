import { useRef } from 'react';
import findIndex from 'lodash/findIndex';
import intersection from 'lodash/intersection';
import isEqual from 'lodash/isEqual';

import useURLParameter, { HistoryAction, QueryValue } from 'hooks/useURLParameter';
import { SortAggregate, SortOption, ThProps, isSortOption } from 'types/table';
import { ApiSortOption, ApiSortOptionSingle } from 'types/search';
import { isNonEmptyArray, NonEmptyArray } from 'utils/type.utils';

export type FieldOption = { field: string; aggregateBy?: SortAggregate };

export type GetSortParams = (
    columnName: string,
    fieldOptions?: SortAggregate | NonEmptyArray<FieldOption>
) => ThProps['sort'] | undefined;

function sortOptionToApiSortOption({
    field,
    direction,
    aggregateBy,
}: SortOption): ApiSortOptionSingle {
    const sortOption = {
        field,
        reversed: direction === 'desc',
    };
    if (aggregateBy) {
        const { aggregateFunc, distinct } = aggregateBy;
        return {
            ...sortOption,
            aggregateBy: {
                aggregateFunc,
                distinct: distinct === 'true',
            },
        };
    }
    return sortOption;
}

function getValidSortOption(activeSortOption: SortOption | SortOption[]): ApiSortOption {
    return Array.isArray(activeSortOption)
        ? activeSortOption.map(sortOptionToApiSortOption)
        : sortOptionToApiSortOption(activeSortOption);
}

function getActiveSortOption(
    sortOption: QueryValue,
    defaultSortOption: SortOption | NonEmptyArray<SortOption>
): SortOption | NonEmptyArray<SortOption> {
    if (Array.isArray(sortOption)) {
        const validOptions = sortOption.filter(isSortOption);
        return isNonEmptyArray(validOptions) ? validOptions : defaultSortOption;
    }

    return isSortOption(sortOption) ? sortOption : defaultSortOption;
}

export type UseURLSortProps = {
    sortFields: (string | string[])[];
    defaultSortOption: SortOption | NonEmptyArray<SortOption>;
    onSort?: (newSortOption: SortOption | SortOption[]) => void;
};

export type UseURLSortResult = {
    sortOption: ApiSortOption;
    setSortOption: (
        newSortOption: SortOption | SortOption[],
        historyAction?: HistoryAction
    ) => void;
    getSortParams: GetSortParams;
};

/**
 * A hook that manages the sort option for a table in the URL.
 *
 * @param options.sortFields
 *      An array of (string | string[]) that represent the sort fields that will
 *      sent over the API for a given table column. A `string` value represents a
 *      single field, while a `string[]` value represents multiple fields that may
 *      have a subset of values sent over the API.
 * @param options.defaultSortOption
 *      The default sort option to use when the sort option is not present in the URL.
 * @param options.onSort
 *      A callback function that is called when the sort option is changed.
 * @returns UseURLSortResult.sortOption
 *      The current sort option. This may be a single
 *      sort option or an array of sort options.
 * @returns UseURLSortResult.setSortOption
 *     A function that can be used to directly set the sort option.
 * @returns UseURLSortResult.getSortParams
 *     A function that can be used to get the sort parameters for a given column in a PatternFly table.
 *     This function has two main signatures for the singular and multi sort use cases:
 *
 *     1. getSortParams(columnName: string, fieldOptions?: SortAggregate)
 *     - This signature is used when the column has a single sort field. The `columnName` parameter
 *     is used to find the correct column index in the PatternFly table, and is also used as the default
 *     field name when sending the sort option to the API. The `fieldOptions` parameter is optional and
 *     is used to specify the aggregate function to use when sorting the column.
 *
 *     2. getSortParams(columnName: string, fieldOptions: { field: string; aggregateBy?: SortAggregate }[])
 *     - This signature is used when the column has multiple sort fields. In this form, the fields sent to
 *     the API will be the `field` values in the `fieldOptions` array, each of which can optionally
 *     include an aggregate. The correct column index in the PatternFly table is found by finding any array
 *     provided in the `sortFields` option that contains one of the `field` values in the `fieldOptions` array.
 *
 * @example
 * // A use case where a table only has single sort columns
 * const { sortOption, setSortOption, getSortParams } = useURLSort({
 *    sortFields: ['CVE', 'Top CVSS', 'First Discovered'],
 *    defaultSortOption: { field: 'CVE', direction: 'asc' },
 * });
 *
 * return (
 *   <Thead>
 *     <Th sort={getSortParams('CVE')}>CVE</Th>
 *     <Th sort={getSortParams('Top CVSS')}>Top CVSS</Th>
 *     <Th sort={getSortParams('First Discovered')}>First discovered</Th>
 *   </Thead>
 * );
 *
 * @example
 * // A use case where a table has both single and multi sort columns
 * const { sortOption, setSortOption, getSortParams } = useURLSort({
 *   sortFields: ['CVE', 'Top CVSS', 'First Discovered', ['Critical Severity Count', 'Important Severity Count']],
 *  defaultSortOption: { field: 'CVE', direction: 'asc' },
 * });
 *
 * return (
 *   <Thead>
 *     <Th sort={getSortParams('CVE')}>CVE</Th>
 *     <Th sort={getSortParams('Top CVSS')}>Top CVSS</Th>
 *     <Th sort={getSortParams('First Discovered')}>First discovered</Th>
 *    <Th sort={getSortParams('Images By Severity', [
 *       { field: 'Critical Severity Count' },
 *       { field: 'Important Severity Count' },
 *    ])}>Images by severity</Th>
 *   </Thead>
 * );
 *
 *
 */
function useURLSort({ sortFields, defaultSortOption, onSort }: UseURLSortProps): UseURLSortResult {
    const [sortOption, setSortOption] = useURLParameter('sortOption', defaultSortOption);

    // get the parsed sort option values from the URL, if available
    // otherwise, use the default sort option values
    const activeSortOption = getActiveSortOption(sortOption, defaultSortOption);

    const internalSortResultOption = useRef<ApiSortOption>(getValidSortOption(activeSortOption));

    function getSortParams(
        columnName: string,
        fieldOptions?: SortAggregate | NonEmptyArray<FieldOption>
    ): ThProps['sort'] {
        // Convert the caller provided sort fields of type (string | string[]) to an
        // array of string[].
        const declaredSortFields: string[][] = sortFields.map((field) =>
            Array.isArray(field) ? field : [field]
        );

        // Find the index of the field passed to `getSortParams` in the sortFields provided
        // to the hook.
        const targetSortFields: string[] = Array.isArray(fieldOptions)
            ? fieldOptions.map(({ field }) => field)
            : [columnName];

        const index = findIndex(
            declaredSortFields,
            (field) => intersection(field, targetSortFields).length > 0
        );

        // Find the index of the active sort field in the declared sort fields.
        const activeSortOptionFields: string[] = Array.isArray(activeSortOption)
            ? activeSortOption.map(({ field }) => field)
            : [activeSortOption.field];

        const activeSortIndex = findIndex(
            declaredSortFields,
            (field) => intersection(field, activeSortOptionFields).length > 0
        );

        // We can't support multiple sort fields with different directions at this time, so
        // we'll just use the first sort field's direction.
        const direction = Array.isArray(activeSortOption)
            ? activeSortOption[0].direction
            : activeSortOption.direction;

        return {
            sortBy: {
                index: activeSortIndex,
                direction,
                defaultDirection: 'desc',
            },
            onSort: (_event, _index, direction) => {
                const newSortOption = Array.isArray(fieldOptions)
                    ? fieldOptions.map((o) => ({ ...o, direction }))
                    : { field: columnName, aggregateBy: fieldOptions, direction };

                if (onSort) {
                    onSort(newSortOption);
                }
                setSortOption(newSortOption);
            },
            columnIndex: index,
        };
    }

    if (!isEqual(internalSortResultOption.current, getValidSortOption(activeSortOption))) {
        internalSortResultOption.current = getValidSortOption(activeSortOption);
    }

    return {
        sortOption: internalSortResultOption.current,
        setSortOption,
        getSortParams,
    };
}

export default useURLSort;
