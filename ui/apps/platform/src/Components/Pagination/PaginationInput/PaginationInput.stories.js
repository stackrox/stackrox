/* eslint-disable react-hooks/rules-of-hooks */
import React, { useState } from 'react';

import PaginationInput from './PaginationInput';

export default {
    title: 'Pagination',
    component: PaginationInput,
};

export const basicUsage = () => {
    const [currentPage, setPage] = useState(1);
    const totalSize = 50;
    const pageSize = 10;

    return (
        <PaginationInput
            currentPage={currentPage}
            totalSize={totalSize}
            pageSize={pageSize}
            onChange={setPage}
        />
    );
};
