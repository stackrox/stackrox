import { ReactElement } from 'react';
import * as yup from 'yup';

export type { ThProps } from '@patternfly/react-table';

export const SortDirectionSchema = yup.string().oneOf(['asc', 'desc']).defined();
export const AggregateFuncSchema = yup.string().oneOf(['max', 'count', 'min']).defined();
export const SortAggregateSchema = yup.object({
    aggregateFunc: AggregateFuncSchema.required(),
    distinct: yup.string().oneOf(['true', 'false']).optional(),
});
export const SortOptionSchema = yup.object({
    field: yup.string().required(),
    // The .default(undefined) is necessary to allow for the aggregateBy field to be
    // omitted and pass validation.
    aggregateBy: SortAggregateSchema.optional().default(undefined),
    direction: SortDirectionSchema.required(),
});

export type SortDirection = yup.InferType<typeof SortDirectionSchema>;
export type AggregateFunc = yup.InferType<typeof AggregateFuncSchema>;
export type SortAggregate = yup.InferType<typeof SortAggregateSchema>;
export type SortOption = yup.InferType<typeof SortOptionSchema>;

export function isSortOption(val: unknown): val is SortOption {
    try {
        SortOptionSchema.validateSync(val);
        return true;
    } catch {
        return false;
    }
}

export type TableColumn = {
    Header: string;
    accessor: string;
    Cell?: ({ original, value }) => ReactElement | string;
    sortField?: string;
};
