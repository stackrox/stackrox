import React, { ReactElement } from 'react';
import { HashLink } from 'react-router-hash-link';

const ViewAllButton = ({ url }: { url: string }): ReactElement => {
    return (
        <HashLink to={url} className="btn-sm btn-base whitespace-nowrap no-underline">
            View All
        </HashLink>
    );
};

export default ViewAllButton;
