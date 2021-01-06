import React, { ReactElement, ReactNode } from 'react';

export type TableBodyProps = {
    children: ReactNode;
};

function TableBody({ children }: TableBodyProps): ReactElement {
    return <tbody>{children}</tbody>;
}

export default TableBody;
