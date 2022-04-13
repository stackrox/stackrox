import { ReactElement } from 'react';

export type { ThProps } from '@patternfly/react-table';

export type SortDirection = 'asc' | 'desc';

export type SortOption = {
    field: string;
    direction: SortDirection;
};

export type TableColumn = {
    Header: string;
    accessor: string;
    Cell?: ({ original, value }) => ReactElement | string;
    sortField?: string;
};
