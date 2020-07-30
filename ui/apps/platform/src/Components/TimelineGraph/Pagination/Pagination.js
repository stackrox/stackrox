import React from 'react';
import PropTypes from 'prop-types';

import { PaginationInput, PrevPaginationButton, NextPaginationButton } from 'Components/Pagination';

const Pagination = ({ currentPage, totalSize, pageSize, onChange }) => {
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

Pagination.propTypes = {
    currentPage: PropTypes.number.isRequired,
    pageSize: PropTypes.number.isRequired,
    totalSize: PropTypes.number.isRequired,
    onChange: PropTypes.func.isRequired,
};

export default Pagination;
