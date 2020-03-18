import React, { useState } from 'react';

import { PaginationInput, PrevPaginationButton, NextPaginationButton } from 'Components/Pagination';

const Pagination = () => {
    const [currentPage, setPage] = useState(1);
    const totalSize = 50;
    const pageSize = 10;
    function onChange(page) {
        setPage(page);
    }
    return (
        <div className="flex flex-col h-full items-center justify-center">
            <div className="mb-4">
                <PrevPaginationButton
                    className="border-b-2 border-l-2 border-primary-300 border-t-2 px-4 py-2 rounded-l hover:bg-primary-200"
                    totalSize={totalSize}
                    currentPage={currentPage}
                    pageSize={pageSize}
                    onChange={onChange}
                />
                <NextPaginationButton
                    className="border-b-2 border-l-2 border-primary-300 border-r-2 border-t-2 px-4 py-2 rounded-r hover:bg-primary-200"
                    totalSize={totalSize}
                    currentPage={currentPage}
                    pageSize={pageSize}
                    onChange={onChange}
                />
            </div>
            <PaginationInput
                totalSize={totalSize}
                currentPage={currentPage}
                pageSize={pageSize}
                onChange={onChange}
            />
        </div>
    );
};

export default Pagination;
