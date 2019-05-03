import React from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';
import Table, {
    wrapClassName,
    defaultHeaderClassName,
    defaultColumnClassName
} from 'Components/Table';

import NoResultsMessage from 'Components/NoResultsMessage';
import { sortValue, sortDate } from 'sorters/sorters';

const ImagesTable = ({ rows, selectedImage, history, location, page }) => {
    function updateSelectedImage(image) {
        const urlSuffix = image && image.id ? `/${image.id}` : '';
        history.push({
            pathname: `/main/images${urlSuffix}`,
            search: location.search
        });
    }

    const columns = [
        {
            accessor: 'name',
            Header: 'Image',
            headerClassName: `w-1/2 ${defaultHeaderClassName}`,
            className: `w-1/2 word-break-all ${wrapClassName} ${defaultColumnClassName}`,
            // eslint-disable-next-line react/prop-types
            Cell: ({ value }) => <span>{value}</span>
        },
        {
            accessor: 'created',
            Header: 'Created at',
            headerClassName: `w-24 ${defaultHeaderClassName}`,
            className: `w-24 ${wrapClassName} ${defaultColumnClassName}`,
            Cell: ({ original }) =>
                original.created ? dateFns.format(original.created, dateTimeFormat) : '—',
            sortMethod: sortDate
        },
        {
            accessor: 'components',
            Header: 'Components',
            headerClassName: `w-24 ${defaultHeaderClassName}`,
            className: `w-24 ${wrapClassName} ${defaultColumnClassName}`,
            Cell: ({ original }) => (original.components !== undefined ? original.components : '—'),
            sortMethod: sortValue
        },
        {
            accessor: 'cves',
            Header: 'CVEs',
            headerClassName: `w-12 ${defaultHeaderClassName}`,
            className: `w-12 ${wrapClassName} ${defaultColumnClassName}`,
            Cell: ({ original }) => (original.cves !== undefined ? original.cves : '—'),
            sortMethod: sortValue
        },
        {
            accessor: 'fixableCves',
            Header: 'Fixable CVEs',
            headerClassName: `w-16 ${defaultHeaderClassName}`,
            className: `w-16 ${wrapClassName} ${defaultColumnClassName}`,
            Cell: ({ original }) =>
                original.fixableCves !== undefined ? original.fixableCves : '—',
            sortMethod: sortValue
        }
    ];
    const selectedId = selectedImage && selectedImage.id;
    if (!rows.length)
        return <NoResultsMessage message="No results found. Please refine your search." />;
    return (
        <Table
            rows={rows}
            columns={columns}
            onRowClick={updateSelectedImage}
            selectedRowId={selectedId}
            noDataText="No results found. Please refine your search."
            page={page}
            defaultSorted={[
                {
                    id: 'cves',
                    desc: true
                }
            ]}
        />
    );
};

ImagesTable.propTypes = {
    rows: PropTypes.arrayOf(PropTypes.object).isRequired,
    selectedImage: PropTypes.shape({}),
    history: ReactRouterPropTypes.history.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    page: PropTypes.number.isRequired
};

ImagesTable.defaultProps = {
    selectedImage: null
};

export default withRouter(ImagesTable);
