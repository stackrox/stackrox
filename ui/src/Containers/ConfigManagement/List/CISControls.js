import React from 'react';
import entityTypes from 'constants/entityTypes';
import { standardLabels } from 'messages/standards';
import { LIST_STANDARD_NO_NODES as QUERY } from 'queries/standard';
import queryService from 'modules/queryService';
import { sortVersion } from 'sorters/sorters';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import { CLIENT_SIDE_SEARCH_OPTIONS as SEARCH_OPTIONS } from 'constants/searchOptions';

import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import LabelChip from 'Components/LabelChip';
import List from './List';

const COMPLIANCE_STATES = {
    Pass: 'Pass',
    Fail: 'Fail'
};

const tableColumns = [
    {
        Header: 'Id',
        headerClassName: 'hidden',
        className: 'hidden',
        accessor: 'id'
    },
    {
        Header: `Standard`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'standard'
    },
    {
        Header: `Control`,
        headerClassName: `w-1/2 ${defaultHeaderClassName}`,
        className: `w-1/2 ${defaultColumnClassName}`,
        accessor: 'control',
        sortMethod: sortVersion
    },
    {
        Header: `Control Status`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        // eslint-disable-next-line
    Cell: ({ original }) => {
            return !original.passing ? <LabelChip text="Fail" type="alert" /> : 'Pass';
        },
        accessor: 'passing'
    }
];

const filterByComplianceState = (rows, state) => {
    if (!state || !rows) return rows;
    return rows.filter(row => (state === COMPLIANCE_STATES.Pass ? row.passing : !row.passing));
};

const createTableRows = data => {
    if (!data || !data.results || !data.results.results.length) return [];

    let standardKeyIndex = 0;
    let controlKeyIndex = 0;
    let nodeKeyIndex = 0;
    data.results.results[0].aggregationKeys.forEach(({ scope }, idx) => {
        if (scope === entityTypes.STANDARD) standardKeyIndex = idx;
        if (scope === entityTypes.CONTROL) controlKeyIndex = idx;
        if (scope === entityTypes.NODE) nodeKeyIndex = idx;
    });
    const controls = {};
    data.results.results.forEach(({ keys, numFailing, numPassing }) => {
        if (!keys[controlKeyIndex]) return;
        const controlId = keys[controlKeyIndex].id;
        if (controls[controlId]) {
            controls[controlId].nodes.push(keys[nodeKeyIndex].name);
            if (numFailing || (!numPassing && !numFailing)) {
                controls[controlId].passing = false;
            }
        } else {
            controls[controlId] = {
                id: controlId,
                standard: standardLabels[keys[standardKeyIndex].id],
                control: `${keys[controlKeyIndex].name} - ${keys[controlKeyIndex].description}`,
                passing: !numFailing,
                nodes: [keys[nodeKeyIndex].name]
            };
        }
    });
    return Object.values(controls);
};

const CISControls = ({ className, selectedRowId, onRowClick, query, data }) => {
    const queryText = queryService.objectToWhereClause({ Standard: 'CIS', ...query });
    const variables = {
        where: queryText,
        groupBy: [entityTypes.STANDARD, entityTypes.CONTROL, entityTypes.NODE]
    };

    const complianceState = query ? query[SEARCH_OPTIONS.COMPLIANCE.STATE] : null;

    function createTableRowsFilteredByComplianceState(items) {
        const tableRows = createTableRows(items);
        const filteredTableRows = filterByComplianceState(tableRows, complianceState);
        return filteredTableRows;
    }

    return (
        <List
            className={className}
            query={QUERY}
            variables={variables}
            headerText="CIS Controls"
            entityType={entityTypes.CONTROL}
            tableColumns={tableColumns}
            createTableRows={createTableRowsFilteredByComplianceState}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="id"
            defaultSearchOptions={[SEARCH_OPTIONS.COMPLIANCE.STATE]}
            data={filterByComplianceState(data, complianceState)}
        />
    );
};

CISControls.propTypes = entityListPropTypes;
CISControls.defaultProps = entityListDefaultprops;

export default CISControls;
