import React from 'react';
import entityTypes from 'constants/entityTypes';
import { POLICIES as QUERY } from 'queries/policy';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import queryService from 'modules/queryService';
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

const Policies = ({ className, onRowClick, query, selectedRowId }) => {
    const queryText = queryService.objectToWhereClause(query);
    const variables = queryText ? { query: queryText } : null;
    return (
        <List
            className={className}
            query={QUERY}
            variables={variables}
            entityType={entityTypes.POLICY}
            tableColumns={tableColumns}
            createTableRows={createTableRows}
            selectedRowId={selectedRowId}
            onRowClick={onRowClick}
            idAttribute="id"
        />
    );
};

Policies.propTypes = entityListPropTypes;
Policies.defaultProps = entityListDefaultprops;

export default Policies;
