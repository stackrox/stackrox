import { ReactElement } from 'react';

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

export type TableColorStyles = {
    bgColor: string;
    borderColor: string;
    textColor: string;
};
