import React, { useState } from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import { gql, useQuery } from '@apollo/client';
import { Message } from '@stackrox/ui-components';

import Loader from 'Components/Loader';
import fixableVulnTypeContext from 'Containers/VulnMgmt/fixableVulnTypeContext';
import { getCveTableColumns, defaultCveSort } from 'Containers/VulnMgmt/List/Cves/VulnMgmtListCves';
import {
    CLUSTER_CVE_LIST_FRAGMENT,
    NODE_CVE_LIST_FRAGMENT,
    IMAGE_CVE_LIST_FRAGMENT,
    VULN_CVE_LIST_FRAGMENT,
} from 'Containers/VulnMgmt/VulnMgmt.fragments';
import { LIST_PAGE_SIZE } from 'constants/workflowPages.constants';
import entityTypes from 'constants/entityTypes';
import { resourceLabels } from 'messages/common';
import queryService from 'utils/queryService';
import useFeatureFlags from 'hooks/useFeatureFlags';

import FixableCveExportButton from '../VulnMgmtComponents/FixableCveExportButton';
import TableWidget from './TableWidget';
import { getScopeQuery } from './VulnMgmtPolicyQueryUtil';

const queryFieldNames = {
    [entityTypes.CLUSTER]: 'cluster',
    [entityTypes.NODE]: 'node',
    [entityTypes.NAMESPACE]: 'namespace',
    [entityTypes.DEPLOYMENT]: 'deployment',
    [entityTypes.COMPONENT]: 'component',
    [entityTypes.NODE_COMPONENT]: 'nodeComponent',
    [entityTypes.IMAGE_COMPONENT]: 'imageComponent',
};

const TableWidgetFixableCves = ({
    workflowState,
    entityContext,
    entityType,
    name,
    id,
    vulnType = entityTypes.CVE,
}) => {
    const [fixableCvesPage, setFixableCvesPage] = useState(0);
    const [cveSort, setCveSort] = useState(defaultCveSort);

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const showVMUpdates = isFeatureFlagEnabled('ROX_FRONTEND_VM_UPDATES');

    const displayedEntityType = resourceLabels[entityType];
    const displayedVulnType = resourceLabels[vulnType];

    const queryFieldName = queryFieldNames[entityType];
    let queryVulnCounterFieldName = showVMUpdates ? 'imageVulnerabilityCounter' : 'vulnCounter';
    let queryVulnsFieldName = showVMUpdates ? 'imageVulnerabilities' : 'vulns';
    let queryCVEFieldsName = showVMUpdates ? 'imageCVEFields' : 'cveFields';
    let queryFragment = showVMUpdates ? IMAGE_CVE_LIST_FRAGMENT : VULN_CVE_LIST_FRAGMENT;

    if (vulnType === entityTypes.CLUSTER_CVE) {
        queryVulnCounterFieldName = 'clusterVulnerabilityCounter';
        queryVulnsFieldName = 'clusterVulnerabilities';
        queryCVEFieldsName = 'clusterCVEFields';
        queryFragment = CLUSTER_CVE_LIST_FRAGMENT;
    } else if (vulnType === entityTypes.NODE_CVE) {
        queryVulnCounterFieldName = 'nodeVulnerabilityCounter';
        queryVulnsFieldName = 'nodeVulnerabilities';
        queryCVEFieldsName = 'nodeCVEFields';
        queryFragment = NODE_CVE_LIST_FRAGMENT;
    } else if (entityType === entityTypes.IMAGE_COMPONENT || vulnType === entityTypes.IMAGE_CVE) {
        // TODO: after the split of CVE types is released, make this the default
        queryVulnCounterFieldName = 'imageVulnerabilityCounter';
        queryVulnsFieldName = 'imageVulnerabilities';
        queryCVEFieldsName = 'imageCVEFields';
        queryFragment = IMAGE_CVE_LIST_FRAGMENT;
    }

    // `id` field is not needed in result,
    //   but is needed to keep apollo-client from throwing an error with certain entities,
    //   because apollo-client lib is "sub-optimal", https://github.com/apollographql/react-apollo/issues/1656
    const fixableCvesQuery = gql`
        query getFixableCvesForEntity(
            $id: ID!
            ${
                entityType !== entityTypes.NODE_COMPONENT && vulnType !== entityTypes.NODE_CVE
                    ? '$query: String'
                    : ''
            }
            $scopeQuery: String
            $vulnQuery: String
            $vulnPagination: Pagination
        ) {
            result: ${queryFieldName}(id: $id) {
                ${entityType !== entityTypes.NAMESPACE ? 'id' : ''}
                vulnCounter: ${queryVulnCounterFieldName} {
                    all {
                        fixable
                    }
                }
                vulnerabilities:  ${queryVulnsFieldName}(query: $vulnQuery, scopeQuery: $scopeQuery, pagination: $vulnPagination) {
                    ... ${queryCVEFieldsName}
                }
            }
        }
        ${queryFragment}
    `;
    const queryOptions = {
        variables: {
            id,
            scopeQuery: getScopeQuery(entityContext),
            vulnQuery: queryService.objectToWhereClause({ Fixable: true }),
            vulnPagination: queryService.getPagination(cveSort, fixableCvesPage, LIST_PAGE_SIZE),
        },
    };
    const {
        loading: cvesLoading,
        data: fixableCvesData,
        error: cvesError,
    } = useQuery(fixableCvesQuery, queryOptions);

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
        <fixableVulnTypeContext.Provider value={vulnType}>
            {cvesLoading && (
                <div className="p-6">
                    <Loader transparent />
                </div>
            )}
            {cvesError && (
                <Message type="error">
                    {cvesError.message || 'Error retrieving fixable CVEs'}
                </Message>
            )}
            {!cvesLoading && !cvesError && (
                <TableWidget
                    header={`${fixableCount} fixable ${pluralize(
                        displayedVulnType,
                        fixableCount
                    )} found across this ${displayedEntityType}`}
                    headerActions={cveActions}
                    rows={fixableCves}
                    entityType={vulnType}
                    noDataText={`No fixable CVEs available in this ${displayedEntityType}`}
                    className="bg-base-100"
                    columns={getCveTableColumns(workflowState)}
                    idAttribute="id"
                    pageSize={LIST_PAGE_SIZE}
                    parentPageState={fixableCveState}
                    currentSort={cveSort}
                    defaultSorted={[]}
                    sortHandler={onSortedChange}
                />
            )}
        </fixableVulnTypeContext.Provider>
    );
};

TableWidgetFixableCves.propType = {
    workflowState: PropTypes.shape({}).isRequired,
    entityContext: PropTypes.shape({}).isRequired,
    entityType: PropTypes.string.isRequired,
    vulnType: PropTypes.string,
    name: PropTypes.string.isRequired,
    id: PropTypes.string.isRequired,
};

export default TableWidgetFixableCves;
