import { ReactElement } from 'react';
import { FlattenedNetworkBaseline, BaselineStatus } from 'Containers/Network/networkTypes';

export type Row = {
    id: string;
    original: FlattenedNetworkBaseline;
    values: {
        status: BaselineStatus;
    };
    groupByVal?: BaselineStatus;
    groupByID?: string;
    isGrouped?: boolean;
    subRows?: Row[];
    leafRows?: Row[];
};

export type Cell = {
    getCellProps: () => {
        key: string;
    };
    row: {
        isGrouped: boolean;
    };
    column: {
        id: string;
    };
    render: (string) => ReactElement;
};
