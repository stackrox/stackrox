import React, { ReactElement, ReactNode } from 'react';

export type TableBodyProps = {
    children: ReactNode;
};

export function TableBody({ children }: TableBodyProps): ReactElement {
    return <tbody>{children}</tbody>;
}
