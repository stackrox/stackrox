import React, { useState, useCallback, useEffect } from 'react';
import PropTypes from 'prop-types';
import debounce from 'lodash/debounce';
import clamp from 'lodash/clamp';

// time to wait while user is typing, before firing off a new backend pagination call
//   (see https://stackoverflow.com/a/44755058 for background)
const TYPING_DELAY = 800;

const PaginationInput = ({ totalSize, onChange, currentPage, pageSize }) => {
    const [localPage, setLocalPage] = useState(currentPage);
    const delayedSetPage = useCallback(
        () => debounce((newPage) => onChange(newPage), TYPING_DELAY),
        [onChange]
    );

    const totalPages = Math.ceil(totalSize / pageSize);

    useEffect(() => {
        setLocalPage(currentPage);
    }, [currentPage]);

    function onChangePage(e) {
        const { value } = e.target;
        if (value === '') {
            setLocalPage(''); // allow user to delete existing page number, and then sit and think about which page they want
            return;
        }
        const newPageValue = clamp(Number(value), 1, totalPages);
        setLocalPage(newPageValue);
        delayedSetPage(newPageValue);
    }

    return (
        <div className="select-none">
            Page
            <input
                type="number"
                className="text-center bg-base-100 text-base-600 border-2 border-base-300 px-1 py-1 mx-2 focus:border-primary-400 outline-none"
                value={localPage}
                min={1}
                max={totalPages}
                disabled={totalPages < 2}
                onChange={onChangePage}
                data-testid="pagination-input"
                aria-label="Page Number"
            />
            of {totalPages}
        </div>
    );
};

PaginationInput.propTypes = {
    currentPage: PropTypes.number.isRequired,
    totalSize: PropTypes.number.isRequired,
    onChange: PropTypes.func.isRequired,
    pageSize: PropTypes.number,
};

PaginationInput.defaultProps = {
    pageSize: 10,
};

export default PaginationInput;
