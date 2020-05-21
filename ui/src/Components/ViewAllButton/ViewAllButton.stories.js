import React from 'react';
import { MemoryRouter } from 'react-router-dom';

import ViewAllButton from './ViewAllButton';

export default {
    title: 'ViewAllButton',
    component: ViewAllButton,
};

export const withUrl = () => {
    const url = '/main/vuln_management/images';

    return (
        <MemoryRouter>
            <div className="flex">
                <ViewAllButton url={url} />
            </div>
        </MemoryRouter>
    );
};
