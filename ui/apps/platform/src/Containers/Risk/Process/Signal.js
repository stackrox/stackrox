import React from 'react';
import PropTypes from 'prop-types';
import dateFns from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';

import Table, {
    defaultHeaderClassName,
    defaultColumnClassName,
    wrapClassName,
} from 'Components/Table';
import NoResultsMessage from 'Components/NoResultsMessage';

const columns = [
    {
        Header: 'Time',
        id: 'time',
        accessor: (d) => dateFns.format(d.signal.time, dateTimeFormat),
        headerClassName: `${defaultHeaderClassName} w-1/4 pointer-events-none`,
        className: `${wrapClassName} ${defaultColumnClassName} w-1/4 cursor-auto`,
    },
    {
        Header: 'Pod ID',
        accessor: 'podId',
        headerClassName: `${defaultHeaderClassName} w-1/3 pointer-events-none`,
        className: `${wrapClassName} ${defaultColumnClassName} w-1/3 cursor-auto`,
    },
    {
        Header: 'UID',
        id: 'uid',
        accessor: (d) => d.signal.uid,
        headerClassName: `${defaultHeaderClassName} w-1/6 pointer-events-none`,
        className: `${wrapClassName} ${defaultColumnClassName} w-1/6 cursor-auto`,
    },
];

function Signal({ signals }) {
    const rows = signals;
    if (!rows.length) {
        return <NoResultsMessage message="No results found. Please refine your search." />;
    }

    return (
        <div className="border-b border-base-300">
            <Table
                rows={signals}
                columns={columns}
                noDataText="No results found. Please refine your search."
                page={0}
            />
        </div>
    );
}

Signal.propTypes = {
    signals: PropTypes.arrayOf(PropTypes.object),
};

Signal.defaultProps = {
    signals: [],
};

export default Signal;
