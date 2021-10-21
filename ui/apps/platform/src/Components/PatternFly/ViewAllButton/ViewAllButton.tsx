import React, { ReactElement } from 'react';
import { HashLink } from 'react-router-hash-link';

import './ViewAllButton.css';

const ViewAllButton = ({ url }: { url: string }): ReactElement => {
    return (
        <HashLink to={url} className="view-all-button pf-c-button pf-m-tertiary pf-m-small">
            View All
        </HashLink>
    );
};

export default ViewAllButton;
