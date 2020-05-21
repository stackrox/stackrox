import React, { useState } from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import gql from 'graphql-tag';
import { useQuery } from 'react-apollo';

import Loader from 'Components/Loader';
import Message from 'Components/Message';
import { getCveTableColumns, defaultCveSort } from 'Containers/VulnMgmt/List/Cves/VulnMgmtListCves';
import { VULN_CVE_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import { LIST_PAGE_SIZE } from 'constants/workflowPages.constants';
import entityTypes from 'constants/entityTypes';
import { resourceLabels } from 'messages/common';
import queryService from 'utils/queryService';

import FixableCveExportButton from '../VulnMgmtComponents/FixableCveExportButton';
import TableWidget from './TableWidget';
import { getScopeQuery } from './VulnMgmtPolicyQueryUtil';

const TableWidgetFixableCves = ({ workflowState, entityContext, entityType, name, id }) => {
    const [fixableCvesPage, setFixableCvesPage] = useState(0);
    const [cveSort, setCveSort] = useState(defaultCveSort);

    const displayedEntityType = resourceLabels[entityType];
    const idFieldName = entityType === entityTypes.IMAGE ? 'sha' : 'id';

    // `id` field is not needed in result,
    //   but is needed to keep apollo-client from throwing an error with certain entities,
    //   because apollo-client lib is "sub-optimal", https://github.com/apollographql/react-apollo/issues/1656
    const fixableCvesQuery = gql`
        query getFixableCvesForEntity(
            $id: ID!
            $query: String
            $scopeQuery: String
            $vulnQuery: String
            $vulnPagination: Pagination
        ) {
            result: ${displayedEntityType}(${idFieldName}: $id) {
                ${entityType !== entityTypes.NAMESPACE ? 'id' : ''}
                vulnCounter {
                    all {
                        fixable
                    }
                }
                vulnerabilities: vulns(query: $vulnQuery, pagination: $vulnPagination) {
                    ...cveFields
                }
            }
        }
        ${VULN_CVE_LIST_FRAGMENT}
    `;
    const queryOptions = {
        variables: {
            id,
            query: '',
            scopeQuery: getScopeQuery(entityContext),
            vulnQuery: queryService.objectToWhereClause({ Fixable: true }),
            vulnPagination: queryService.getPagination(cveSort, fixableCvesPage, LIST_PAGE_SIZE),
        },
    };
    const { loading: cvesLoading, data: fixableCvesData, error: cvesError } = useQuery(
        fixableCvesQuery,
        queryOptions
    );

    const fixableCves = fixableCvesData?.result?.vulnerabilities || [];
    const fixableCount = fixableCvesData?.result?.vulnCounter?.all?.fixable || 0;
    const fixableCveState = {
        page: fixableCvesPage,
        setPage: setFixableCvesPage,
        totalCount: fixableCount,
    };

    const cveActions = (
        <FixableCveExportButton
            disabled={!fixableCount}
            workflowState={workflowState}
            entityName={name}
        />
    );

    // @TODO: wrapping the sort state updater,
    //        to document that we may eventually have to handle multi-columns sorts here
    function onSortedChange(newSort) {
        setCveSort(newSort);
    }

    return (
        <>
            {cvesLoading && (
                <div className="p-6">
                    <Loader transparent />
                </div>
            )}
            {cvesError && (
                <Message
                    type="error"
                    message={cvesError.message || 'Error retrieving fixable CVEs'}
                />
            )}
            {!cvesLoading && !cvesError && (
                <TableWidget
                    header={`${fixableCount} fixable ${pluralize(
                        entityTypes.CVE,
                        fixableCount
                    )} found across this ${displayedEntityType}`}
                    headerActions={cveActions}
                    rows={fixableCves}
                    entityType={entityTypes.CVE}
                    noDataText={`No fixable CVEs available in this ${displayedEntityType}`}
                    className="bg-base-100"
                    columns={getCveTableColumns(workflowState)}
                    idAttribute="cve"
                    pageSize={LIST_PAGE_SIZE}
                    parentPageState={fixableCveState}
                    currentSort={cveSort}
                    defaultSorted={[]}
                    sortHandler={onSortedChange}
                />
            )}
        </>
    );
};

TableWidgetFixableCves.propType = {
    workflowState: PropTypes.shape({}).isRequired,
    entityContext: PropTypes.shape({}).isRequired,
    entityType: PropTypes.string.isRequired,
    name: PropTypes.string.isRequired,
    id: PropTypes.string.isRequired,
};

export default TableWidgetFixableCves;
