import React, { ReactElement, ReactNode } from 'react';

export type TableProps = {
    children: ReactNode;
};

function Table({ children }: TableProps): ReactElement {
    return <table className="w-full relative">{children}</table>;
}

export default Table;
