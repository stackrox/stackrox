import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { gql, useQuery } from '@apollo/client';
import { Alert } from '@patternfly/react-core';

import Loader from 'Components/Loader';
import fixableVulnTypeContext from 'Containers/VulnMgmt/fixableVulnTypeContext';
import { getCveTableColumns, defaultCveSort } from 'Containers/VulnMgmt/List/Cves/VulnMgmtListCves';
import {
    CLUSTER_CVE_LIST_FRAGMENT,
    NODE_CVE_LIST_FRAGMENT,
    IMAGE_CVE_LIST_FRAGMENT,
} from 'Containers/VulnMgmt/VulnMgmt.fragments';
import { LIST_PAGE_SIZE } from 'constants/workflowPages.constants';
import entityTypes from 'constants/entityTypes';
import queryService from 'utils/queryService';
import useFeatureFlags from 'hooks/useFeatureFlags';

import {
    entityNounOrdinaryCase,
    entityNounOrdinaryCaseSingular,
} from '../entitiesForVulnerabilityManagement';
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
    vulnType,
}) => {
    const [fixableCvesPage, setFixableCvesPage] = useState(0);
    const [cveSort, setCveSort] = useState(defaultCveSort);

    const { isFeatureFlagEnabled } = useFeatureFlags();

    const displayedEntityType = entityNounOrdinaryCaseSingular[entityType];

    const queryFieldName =
        vulnType === entityTypes.NODE_CVE
            ? queryFieldNames[entityTypes.NODE_COMPONENT] // fix for React state transition
            : queryFieldNames[entityType];
    let queryVulnCounterFieldName = 'imageVulnerabilityCounter';
    let queryVulnsFieldName = 'imageVulnerabilities';
    let queryCVEFieldsName = 'imageCVEFields';
    let queryFragment = IMAGE_CVE_LIST_FRAGMENT;
    let exportType = entityTypes.IMAGE_CVE;

    if (vulnType === entityTypes.CLUSTER_CVE) {
        queryVulnCounterFieldName = 'clusterVulnerabilityCounter';
        queryVulnsFieldName = 'clusterVulnerabilities';
        queryCVEFieldsName = 'clusterCVEFields';
        queryFragment = CLUSTER_CVE_LIST_FRAGMENT;
        exportType = entityTypes.CLUSTER_CVE;
    } else if (vulnType === entityTypes.NODE_CVE) {
        queryVulnCounterFieldName = 'nodeVulnerabilityCounter';
        queryVulnsFieldName = 'nodeVulnerabilities';
        queryCVEFieldsName = 'nodeCVEFields';
        queryFragment = NODE_CVE_LIST_FRAGMENT;
        exportType = entityTypes.NODE_CVE;
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
    const displayedVulnType = entityNounOrdinaryCase(fixableCount, vulnType);

    const cveActions = (
        <FixableCveExportButton
            disabled={!fixableCount}
            workflowState={workflowState}
            entityName={name}
            exportType={exportType}
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
                <Alert variant="warning" isInline title="Error retrieving fixable CVEs">
                    {cvesError.message}
                </Alert>
            )}
            {!cvesLoading && !cvesError && (
                <TableWidget
                    header={`${fixableCount} fixable ${displayedVulnType} found across this ${displayedEntityType}`}
                    headerActions={cveActions}
                    rows={fixableCves}
                    entityType={vulnType}
                    noDataText={`No fixable CVEs available in this ${displayedEntityType}`}
                    className="bg-base-100"
                    columns={getCveTableColumns(workflowState, isFeatureFlagEnabled)}
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
