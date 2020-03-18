/* eslint-disable react-hooks/rules-of-hooks */
import React, { useState } from 'react';

import PaginationInput from '../PaginationInput';
import NextPaginationButton from './NextPaginationButton';

export default {
    title: 'NextPaginationButton',
    component: NextPaginationButton
};

export const basicUsage = () => {
    const [currentPage, setPage] = useState(1);
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
            <NextPaginationButton
                className="border-2 border-primary-300 hover:bg-primary-200 ml-3 p-2"
                totalSize={totalSize}
                currentPage={currentPage}
                pageSize={pageSize}
                onChange={setPage}
            />
        </div>
    );
};
