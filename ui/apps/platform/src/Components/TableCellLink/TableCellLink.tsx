import React, { ReactNode } from 'react';
import { Link } from 'react-router-dom';

export type TableCellLinkProps = {
    children: ReactNode;
    pdf?: boolean;
    url: string;
};
function TableCellLink({ children, pdf, url }: TableCellLinkProps): ReactNode {
    // Prevent row click.
    function onClick(e) {
        e.stopPropagation();
    }

    // This field is necessary to exclude rendering the Link during PDF generation. It causes an error where the Link can't be rendered outside a Router
    if (pdf) {
        return children;
    }

    return (
        <Link to={url} className="h-full text-left items-center flex" onClick={onClick}>
            {children}
        </Link>
    );
}

export default TableCellLink;
