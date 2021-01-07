import React, { ReactElement } from 'react';

export type TableHeadProps = {
    headerGroups: {
        getHeaderGroupProps: () => {
            key: string;
        };
        headers: {
            getHeaderProps: () => {
                colSpan: number;
                key: string;
            };
            render: (string) => ReactElement;
        }[];
    }[];
};

function TableHead({ headerGroups }: TableHeadProps): ReactElement {
    return (
        <thead>
            {headerGroups.map((headerGroup) => {
                const { key: headerGroupKey } = headerGroup.getHeaderGroupProps();
                return (
                    <tr key={headerGroupKey}>
                        {headerGroup.headers.map((column) => {
                            const { colSpan, key: headerKey } = column.getHeaderProps();
                            return (
                                <th
                                    colSpan={colSpan}
                                    key={headerKey}
                                    className="text-left p-2 sticky top-0 bg-base-100 border-b border-base-300 z-1"
                                >
                                    {column.render('Header')}
                                </th>
                            );
                        })}
                    </tr>
                );
            })}
        </thead>
    );
}

export default TableHead;
