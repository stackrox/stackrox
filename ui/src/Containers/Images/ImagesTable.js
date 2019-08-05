import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';
import Table, {
    wrapClassName,
    defaultHeaderClassName,
    defaultColumnClassName
} from 'Components/TableV2';

import NoResultsMessage from 'Components/NoResultsMessage';
import { sortValue, sortDate } from 'sorters/sorters';

const columns = [
    {
        accessor: 'name',
        Header: 'Image',
        searchField: 'Image',
        headerClassName: `w-1/2 ${defaultHeaderClassName}`,
        className: `w-1/2 word-break-all ${wrapClassName} ${defaultColumnClassName}`,
        // eslint-disable-next-line react/prop-types
        Cell: ({ value }) => <span>{value}</span>
    },
    {
        accessor: 'created',
        Header: 'Created at',
        searchField: 'Image Created Time',
        headerClassName: `w-24 ${defaultHeaderClassName}`,
        className: `w-24 ${wrapClassName} ${defaultColumnClassName}`,
        Cell: ({ original }) =>
            original.created ? dateFns.format(original.created, dateTimeFormat) : '—',
        sortMethod: sortDate
    },
    {
        accessor: 'components',
        Header: 'Components',
        searchField: 'Component Count',
        headerClassName: `w-24 ${defaultHeaderClassName}`,
        className: `w-24 ${wrapClassName} ${defaultColumnClassName}`,
        Cell: ({ original }) => (original.components !== undefined ? original.components : '—'),
        sortMethod: sortValue
    },
    {
        accessor: 'cves',
        Header: 'CVEs',
        searchField: 'CVE Count',
        headerClassName: `w-12 ${defaultHeaderClassName}`,
        className: `w-12 ${wrapClassName} ${defaultColumnClassName}`,
        Cell: ({ original }) => (original.cves !== undefined ? original.cves : '—'),
        sortMethod: sortValue
    },
    {
        accessor: 'fixableCves',
        Header: 'Fixable CVEs',
        searchField: 'Fixable CVE Count',
        headerClassName: `w-16 ${defaultHeaderClassName}`,
        className: `w-16 ${wrapClassName} ${defaultColumnClassName}`,
        Cell: ({ original }) => (original.fixableCves !== undefined ? original.fixableCves : '—'),
        sortMethod: sortValue
    }
];

function getSortOptionFromState(state) {
    let sortOption;
    if (state.sorted.length && state.sorted[0].id) {
        const column = columns.find(col => col.accessor === state.sorted[0].id);
        sortOption = {
            field: column.searchField,
            reversed: state.sorted[0].desc
        };
    } else {
        sortOption = {
            field: columns[0].searchField,
            reversed: false
        };
    }
    return sortOption;
}

function ImagesTable({ currentImages, selectedImageId, setSelectedImageId, setSortOption }) {
    if (!currentImages.length)
        return <NoResultsMessage message="No results found. Please refine your search." />;

    // When a row is clicked we want to select the list image's id. This will cause the page to load that Image as the
    // selectedImage.
    function setSelectedImage(selectedRow) {
        setSelectedImageId(selectedRow.id);
    }

    // Use the table's 'onFetchData' prop to set our sort option.
    function setSortOptionOnFetch(state) {
        setSortOption(getSortOptionFromState(state));
    }

    // Render the Table.
    return (
        <Table
            rows={currentImages}
            columns={columns}
            onRowClick={setSelectedImage}
            selectedRowId={selectedImageId}
            noDataText="No results found. Please refine your search."
            onFetchData={setSortOptionOnFetch}
        />
    );
}

ImagesTable.propTypes = {
    currentImages: PropTypes.arrayOf(PropTypes.object).isRequired,
    selectedImageId: PropTypes.string,
    setSelectedImageId: PropTypes.func.isRequired,
    setSortOption: PropTypes.func.isRequired
};

ImagesTable.defaultProps = {
    selectedImageId: ''
};

export default withRouter(ImagesTable);
