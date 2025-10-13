import React from 'react';
import PropTypes from 'prop-types';
import clamp from 'lodash/clamp';
import { ChevronLeft } from 'react-feather';

const PrevPaginationButton = ({ className, currentPage, totalSize, pageSize, onChange }) => {
    const totalPages = Math.ceil(totalSize / pageSize);
    function onPreviousPage() {
        const newPage = clamp(currentPage - 1, 1, totalPages);
        onChange(newPage);
    }
    return (
        <button
            type="button"
            className={className}
            onClick={onPreviousPage}
            disabled={currentPage <= 1}
            aria-label="Go to previous page"
        >
            <ChevronLeft className="h-6 w-6" />
        </button>
    );
};

PrevPaginationButton.propTypes = {
    className: PropTypes.string,
    currentPage: PropTypes.number.isRequired,
    totalSize: PropTypes.number.isRequired,
    onChange: PropTypes.func.isRequired,
    pageSize: PropTypes.number,
};

PrevPaginationButton.defaultProps = {
    className: '',
    pageSize: 10,
};

export default PrevPaginationButton;
