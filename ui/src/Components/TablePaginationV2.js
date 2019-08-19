import React, { useEffect } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';
import { withRouter } from 'react-router-dom';

const TablePaginationV2 = ({ pageCount, page, setPage, searchOptions }) => {
    function onChange(e) {
        let { value } = e.target;
        value = Number(value);
        if (value > 0 && value <= pageCount) {
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

    useEffect(resetPage, [searchOptions]);

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
                        max={pageCount}
                        disabled={pageCount === 1}
                        onChange={onChange}
                        data-test-id="page-number-input"
                    />
                    of {pageCount}
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
                    disabled={page === pageCount - 1}
                    data-test-id="next-page-button"
                >
                    <Icon.ChevronRight className="h-6 w-6" />
                </button>
            </div>
        </div>
    );
};

TablePaginationV2.propTypes = {
    pageCount: PropTypes.number.isRequired,
    page: PropTypes.number.isRequired,
    setPage: PropTypes.func.isRequired,
    searchOptions: PropTypes.arrayOf(PropTypes.shape({}))
};

TablePaginationV2.defaultProps = {
    searchOptions: []
};

const mapStateToProps = createStructuredSelector({
    searchOptions: selectors.getAllSearchOptions
});

export default withRouter(connect(mapStateToProps)(TablePaginationV2));
