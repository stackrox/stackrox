import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';

const ViewAllButton = ({ url }: { url: string }): ReactElement => {
    return (
        <Link to={url} className="btn-sm btn-base whitespace-nowrap no-underline">
            View all
        </Link>
    );
};

export default ViewAllButton;
