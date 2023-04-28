import React from 'react';
import PropTypes from 'prop-types';
import clamp from 'lodash/clamp';
import { ChevronRight } from 'react-feather';

const NextPaginationButton = ({ className, currentPage, totalSize, pageSize, onChange }) => {
    const totalPages = Math.ceil(totalSize / pageSize);
    function onNextPage() {
        const newPage = clamp(currentPage + 1, 1, totalPages);
        onChange(newPage);
    }
    return (
        <button
            type="button"
            className={className}
            onClick={onNextPage}
            disabled={currentPage === totalPages || totalPages === 1}
            aria-label="Go to next page"
        >
            <ChevronRight className="h-6 w-6" />
        </button>
    );
};

NextPaginationButton.propTypes = {
    className: PropTypes.string,
    currentPage: PropTypes.number.isRequired,
    totalSize: PropTypes.number.isRequired,
    onChange: PropTypes.func.isRequired,
    pageSize: PropTypes.number,
};

NextPaginationButton.defaultProps = {
    className: '',
    pageSize: 10,
};

export default NextPaginationButton;
