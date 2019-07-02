import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { POLICIES as QUERY } from 'queries/policy';

import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import LifecycleStageLabel from 'Components/LifecycleStageLabel';
import List from './List';

const tableColumns = [
    {
        Header: 'Id',
        headerClassName: 'hidden',
        className: 'hidden',
        accessor: 'id'
    },
    {
        Header: `Policy`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'name'
    },
    {
        Header: `Description`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'description'
    },
    {
        Header: `Lifecycle Stage`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        Cell: ({ original }) => {
            const { lifecycleStages } = original;
            return lifecycleStages.map(lifecycleStage => (
                <LifecycleStageLabel
                    key={lifecycleStage}
                    className="mr-2"
                    lifecycleStage={lifecycleStage}
                />
            ));
        },
        accessor: 'lifecycleStages'
    }
];

const createTableRows = data => data.policies;

const Policies = ({ className, selectedRowId, onRowClick }) => (
    <List
        className={className}
        query={QUERY}
        entityType={entityTypes.POLICY}
        tableColumns={tableColumns}
        createTableRows={createTableRows}
        onRowClick={onRowClick}
        selectedRowId={selectedRowId}
        idAttribute="id"
    />
);

Policies.propTypes = {
    className: PropTypes.string,
    selectedRowId: PropTypes.string,
    onRowClick: PropTypes.func.isRequired
};

Policies.defaultProps = {
    className: '',
    selectedRowId: null
};

export default Policies;
