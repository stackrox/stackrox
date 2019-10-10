import React, { useState } from 'react';
import { MemoryRouter } from 'react-router-dom';

import Widget from './Widget';

export default {
    title: 'Widget',
    component: Widget
};

export const withTitleComponents = () => (
    <MemoryRouter>
        <Widget titleComponents="Title Components go here...">
            <div className="p-4">Child Components go here...</div>
        </Widget>
    </MemoryRouter>
);

export const withHeader = () => (
    <MemoryRouter>
        <Widget header="Header goes here...">
            <div className="p-4">Child Components go here...</div>
        </Widget>
    </MemoryRouter>
);

export const withHeaderComponents = () => (
    <MemoryRouter>
        <Widget header="Header goes here..." headerComponents="Header Components go here...">
            <div className="p-4">Child Components go here...</div>
        </Widget>
    </MemoryRouter>
);

export const withPagerControls = () => {
    // eslint-disable-next-line
    const [currentPage, setCurrentPage] = useState(0);
    const totalPages = 5;
    function onPageChange(page) {
        setCurrentPage(page);
    }
    return (
        <MemoryRouter>
            <Widget header="Header goes here..." pages={totalPages} onPageChange={onPageChange}>
                <div className="p-4">Current Page: {currentPage + 1}</div>
            </Widget>
        </MemoryRouter>
    );
};
