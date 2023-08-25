import { ReactElement } from 'react';

export type { ThProps } from '@patternfly/react-table';

export type SortDirection = 'asc' | 'desc';
export type AggregateFunc = 'max' | 'count' | 'min';

export type SortAggregate = {
    aggregateFunc: AggregateFunc;
    distinct?: 'true' | 'false';
};

export type SortOption = {
    field: string;
    aggregateBy?: SortAggregate;
    direction: SortDirection;
};

export type TableColumn = {
    Header: string;
    accessor: string;
    Cell?: ({ original, value }) => ReactElement | string;
    sortField?: string;
};
