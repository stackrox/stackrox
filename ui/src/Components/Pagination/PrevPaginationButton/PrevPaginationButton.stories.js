/* eslint-disable react-hooks/rules-of-hooks */
import React, { useState } from 'react';

import PaginationInput from '../PaginationInput';
import PrevPaginationButton from './PrevPaginationButton';

export default {
    title: 'PrevPaginationButton',
    component: PrevPaginationButton
};

export const basicUsage = () => {
    const [currentPage, setPage] = useState(5);
    const totalSize = 50;
    const pageSize = 10;

    return (
        <div className="flex flex-1 items-center">
            <PaginationInput
                currentPage={currentPage}
                totalSize={50}
                pageSize={10}
                onChange={setPage}
            />
            <PrevPaginationButton
                className="border-2 border-primary-300 hover:bg-primary-200 ml-3 p-2"
                totalSize={totalSize}
                currentPage={currentPage}
                pageSize={pageSize}
                onChange={setPage}
            />
        </div>
    );
};
