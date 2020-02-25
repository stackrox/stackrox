import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { withRouter } from 'react-router-dom';

import { pageSize as defaultPageSize } from './Table';

const TablePagination = ({ dataLength, setPage, page, pageSize }) => {
    function getTotalPages() {
        return Math.ceil(dataLength / pageSize);
    }

    function onChangePage(e) {
        let { value } = e.target;
        value = Number(value);
        if (value >= 0) {
            value -= 1;
            setPage(value);
        }
    }

    function previousPage() {
        setPage(page - 1);
    }

    function nextPage() {
        setPage(page + 1);
    }

    const totalPages = getTotalPages();
    const curPage = totalPages === 0 ? 0 : `${page + 1}`;

    // @TODO: move the resetting of the page number to 0 when the search parameters change for the entity list being displayed
    //   up to a higher level of state,
    //   because this causes the sidepanel to close when it is open to a sub-list, and the page is refreshed
    //   --when the TablePagination component in the back resets the URL to the top-level lists URL
    // useEffect(resetPage, [searchOptions]);

    return (
        <div
            data-test-id="pagination-header"
            className="flex items-center justify-end text-base-500 font-500"
        >
            <div className="flex items-center pl-5">
                <div className="mr-4 font-600">
                    Page
                    <input
                        type="number"
                        className="text-center bg-base-100 text-base-900 border-2 border-base-300 px-1 py-1 mx-2 focus:border-primary-100 outline-none"
                        value={curPage}
                        min={1}
                        max={totalPages}
                        disabled={totalPages === 1}
                        onChange={onChangePage}
                        data-test-id="page-number-input"
                    />
                    of {totalPages}
                </div>
                <button
                    type="button"
                    className="flex items-center rounded-full hover:bg-primary-200 hover:text-primary-600 mr-1 p-1"
                    onClick={previousPage}
                    disabled={page <= 0}
                    data-test-id="prev-page-button"
                >
                    <Icon.ChevronLeft className="h-6 w-6" />
                </button>
                <button
                    type="button"
                    className="flex items-center rounded-full text-base-600 hover:bg-primary-200 hover:text-primary-600 p-1"
                    onClick={nextPage}
                    disabled={page === totalPages - 1 || totalPages === 0}
                    data-test-id="next-page-button"
                >
                    <Icon.ChevronRight className="h-6 w-6" />
                </button>
            </div>
        </div>
    );
};

TablePagination.propTypes = {
    page: PropTypes.number.isRequired,
    dataLength: PropTypes.number.isRequired,
    setPage: PropTypes.func.isRequired,
    pageSize: PropTypes.number
};

TablePagination.defaultProps = {
    pageSize: defaultPageSize
};

export default withRouter(TablePagination);
