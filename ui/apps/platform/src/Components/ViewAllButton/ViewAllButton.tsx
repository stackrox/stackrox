import React, { ReactElement } from 'react';
import { HashLink } from 'react-router-hash-link';

const ViewAllButton = ({ url }: { url: string }): ReactElement => {
    return (
        <HashLink to={url} className="no-underline">
            <button className="btn-sm btn-base whitespace-nowrap" type="button">
                View All
            </button>
        </HashLink>
    );
};

export default ViewAllButton;
