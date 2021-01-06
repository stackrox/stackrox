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
};
