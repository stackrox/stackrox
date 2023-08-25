import React, { useState, useEffect } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { withRouter } from 'react-router-dom';
import debounce from 'lodash/debounce';
import clamp from 'lodash/clamp';

import { DEFAULT_PAGE_SIZE as defaultPageSize } from './Table';

// time to wait while user is typing, before firing off a new backend pagination call
//   (see https://stackoverflow.com/a/44755058 for background)
const TYPING_DELAY = 800;

const TablePagination = ({ dataLength, setPage, page, pageSize }) => {
    function getTotalPages() {
        return Math.ceil(dataLength / pageSize);
    }

    const totalPages = getTotalPages();

    // At first glance, this looks like unnecessary duplicating,
    //   but it's solving the problem of giving the user time to actually type a number
    //   before firing off an event to query the backend with new pagination
    //
    // 1. create local state to allow the text input to change immediately
    //    @TODO: abstract the math out of this component, backend pagination is 0-based
    const [localPage, setLocalPage] = useState(page + 1);

    // 2. debounce the setPage callback to delay the setPage call when typing
    const delayedSetPage = debounce((newPage) => setPage(newPage), TYPING_DELAY);

    useEffect(() => {
        setLocalPage(page + 1);
    }, [page]);

    function onChangePage(e) {
        const { value } = e.target;
        if (value === '' || value === '0') {
            setLocalPage(''); // allow user to delete existing page number, and then sit and think about which page they want
            return;
        }

        const newPageValue = clamp(Number(value), 1, totalPages); // @TODO: abstract the math out of this component
        setLocalPage(newPageValue);

        const adjustedPage = newPageValue - 1;
        delayedSetPage(adjustedPage);
    }

    function previousPage() {
        const newPage = clamp(localPage - 1, 1, totalPages);
        setLocalPage(newPage);
        const adjustedValue = newPage - 1; // @TODO: abstract the math out of this component
        setPage(adjustedValue);
    }

    function nextPage() {
        const newPage = clamp(localPage + 1, 1, totalPages);
        setLocalPage(newPage);
        const adjustedValue = newPage - 1; // @TODO: abstract the math out of this component
        setPage(adjustedValue);
    }

    // @TODO: move the resetting of the page number to 0 when the search parameters change for the entity list being displayed
    //   up to a higher level of state,
    //   because this causes the sidepanel to close when it is open to a sub-list, and the page is refreshed
    //   --when the TablePagination component in the back resets the URL to the top-level lists URL
    // useEffect(resetPage, [searchOptions]);

    return (
        <div data-testid="pagination-header" className="flex items-center justify-end">
            <div className="flex items-center pl-5">
                <div className="mr-4 min-w-24">
                    Page
                    <input
                        type="number"
                        className="text-center bg-base-100 text-base-600 border-2 border-base-300 px-1 py-1 mx-2 focus:border-primary-100 outline-none"
                        value={localPage}
                        min={1}
                        max={totalPages}
                        disabled={totalPages < 2}
                        onChange={onChangePage}
                        data-testid="page-number-input"
                        aria-label="Page Number"
                    />
                    of {totalPages === 0 ? 1 : totalPages}
                </div>
                <button
                    type="button"
                    className="flex items-center rounded-full text-base-600 hover:bg-primary-200 hover:text-primary-600 mr-1 p-1"
                    onClick={previousPage}
                    disabled={page <= 0}
                    aria-label="Go to previous page"
                >
                    <Icon.ChevronLeft className="h-6 w-6" />
                </button>
                <button
                    type="button"
                    className="flex items-center rounded-full text-base-600 hover:bg-primary-200 hover:text-primary-600 p-1"
                    onClick={nextPage}
                    disabled={page >= totalPages - 1}
                    aria-label="Go to next page"
                >
                    <Icon.ChevronRight className="h-6 w-6" />
                </button>
            </div>
        </div>
    );
};

TablePagination.propTypes = {
    page: PropTypes.number.isRequired,
    dataLength: PropTypes.number,
    setPage: PropTypes.func.isRequired,
    pageSize: PropTypes.number,
};

TablePagination.defaultProps = {
    pageSize: defaultPageSize,
    dataLength: 0,
};

export default withRouter(TablePagination);
