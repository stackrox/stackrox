import React, { useEffect } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';
import { withRouter } from 'react-router-dom';

import { pageSize } from './Table';

const TablePagination = ({ dataLength, setPage, page, searchOptions, location }) => {
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

    function resetPage() {
        setPage(0);
    }

    const curPage = `${page + 1}`;
    const totalPages = getTotalPages();

    useEffect(resetPage, [searchOptions, location]);

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
                    disabled={page === totalPages - 1}
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
    searchOptions: PropTypes.arrayOf(PropTypes.shape({})),
    location: ReactRouterPropTypes.location.isRequired
};

TablePagination.defaultProps = {
    searchOptions: []
};

const mapStateToProps = createStructuredSelector({
    searchOptions: selectors.getAllSearchOptions
});

export default withRouter(
    connect(
        mapStateToProps,
        null
    )(TablePagination)
);
