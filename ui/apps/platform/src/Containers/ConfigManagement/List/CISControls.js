import React, { useContext } from 'react';
import capitalize from 'lodash/capitalize';

import NotApplicableIconText from 'Components/PatternFly/IconText/NotApplicableIconText';
import PolicyStatusIconText from 'Components/PatternFly/IconText/PolicyStatusIconText';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import searchContext from 'Containers/searchContext';
import COMPLIANCE_STATES from 'constants/complianceStates';
import { entityListPropTypes, entityListDefaultprops } from 'constants/entityPageProps';
import entityTypes from 'constants/entityTypes';
import { CLIENT_SIDE_SEARCH_OPTIONS as SEARCH_OPTIONS } from 'constants/searchOptions';
import { standardLabels } from 'messages/standards';
import { LIST_STANDARD_NO_NODES as QUERY } from 'queries/standard';
import { sortVersion, sortStatus } from 'sorters/sorters';
import queryService from 'utils/queryService';
import ListFrontendPaginated from './ListFrontendPaginated';

const tableColumns = [
    {
        Header: 'Id',
        headerClassName: 'hidden',
        className: 'hidden',
        accessor: 'id',
    },
    {
        Header: `Standard`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        accessor: 'standard',
    },
    {
        Header: `Control`,
        headerClassName: `w-1/2 ${defaultHeaderClassName}`,
        className: `w-1/2 ${defaultColumnClassName}`,
        accessor: 'control',
        sortMethod: sortVersion,
    },
    {
        Header: `Control Status`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName} capitalize`,
        Cell: ({ original, pdf }) => {
            if (original.status === COMPLIANCE_STATES['N/A']) {
                return <NotApplicableIconText isTextOnly={pdf} />;
            }
            return <PolicyStatusIconText isPass={original.status === 'Pass'} isTextOnly={pdf} />;
        },
        accessor: 'status',
        sortMethod: sortStatus,
    },
];

const filterByComplianceState = (rows, state) => {
    if (!state || !rows) {
        return rows;
    }
    const complianceState = capitalize(state);
    const filteredRows = rows.filter((row) => {
        if (complianceState === COMPLIANCE_STATES.PASS) {
            return row.status === COMPLIANCE_STATES.PASS;
        }
        if (complianceState === COMPLIANCE_STATES.FAIL) {
            return row.status === COMPLIANCE_STATES.FAIL;
        }
        return row.status === COMPLIANCE_STATES['N/A'];
    });
    return filteredRows;
};

const createTableRows = (data) => {
    if (!data || !data.results || !data.results.results.length) {
        return [];
    }

    let standardKeyIndex = 0;
    let controlKeyIndex = 0;
    data.results.results[0].aggregationKeys.forEach(({ scope }, idx) => {
        if (scope === entityTypes.STANDARD) {
            standardKeyIndex = idx;
        }
        if (scope === entityTypes.CONTROL) {
            controlKeyIndex = idx;
        }
    });
    const controls = {};
    data.results.results.forEach(({ keys, numFailing, numPassing }) => {
        if (!keys[controlKeyIndex]) {
            return;
        }
        const controlId = keys[controlKeyIndex].id;
        if (controls[controlId]) {
            const { status } = controls[controlId];
            if (status === COMPLIANCE_STATES.FAIL || numFailing) {
                controls[controlId].status = COMPLIANCE_STATES.FAIL;
            }
        } else {
            let status = '';
            if (!numPassing) {
                status = COMPLIANCE_STATES.FAIL;
            }
            if (!numFailing) {
                status = COMPLIANCE_STATES.PASS;
            }
            if (!numPassing && !numFailing) {
                status = COMPLIANCE_STATES['N/A'];
            }
            controls[controlId] = {
                id: controlId,
                standard: standardLabels[keys[standardKeyIndex].id],
                control: `${keys[controlKeyIndex].name} - ${keys[controlKeyIndex].description}`,
                status,
            };
        }
    });
    return Object.values(controls);
};

const CISControls = ({ className, selectedRowId, onRowClick, query, data }) => {
    const searchParam = useContext(searchContext);
    const autoFocusSearchInput = !selectedRowId;

    const { [SEARCH_OPTIONS.COMPLIANCE.STATE]: complianceState, ...restQuery } =
        queryService.getQueryBasedOnSearchContext(query, searchParam);
    const queryObject = { ...restQuery };
    if (!queryObject.Standard) {
        queryObject.Standard = 'CIS';
    }
    const queryText = queryService.objectToWhereClause(queryObject);
    const variables = {
        where: queryText,
        groupBy: [entityTypes.STANDARD, entityTypes.CONTROL],
    };

    function createTableRowsFilteredByComplianceState(items) {
        const tableRows = createTableRows(items);
        const filteredTableRows = filterByComplianceState(tableRows, complianceState);
        return filteredTableRows;
    }
    return (
        <ListFrontendPaginated
            className={className}
            query={QUERY}
            variables={variables}
            headerText="CIS Controls"
            noDataText="No control results available. Please run a scan."
            entityType={entityTypes.CONTROL}
            tableColumns={tableColumns}
            createTableRows={createTableRowsFilteredByComplianceState}
            onRowClick={onRowClick}
            selectedRowId={selectedRowId}
            idAttribute="id"
            defaultSorted={[
                {
                    id: 'status',
                    desc: false,
                },
                {
                    id: 'standard',
                    desc: false,
                },
                {
                    id: 'control',
                    desc: false,
                },
            ]}
            defaultSearchOptions={[SEARCH_OPTIONS.COMPLIANCE.STATE]}
            data={filterByComplianceState(data, complianceState)}
            autoFocusSearchInput={autoFocusSearchInput}
        />
    );
};

CISControls.propTypes = entityListPropTypes;
CISControls.defaultProps = entityListDefaultprops;

export default CISControls;
