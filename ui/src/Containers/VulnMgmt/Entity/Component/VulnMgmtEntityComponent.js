import React from 'react';
import gql from 'graphql-tag';
import WorkflowEntityPage from 'Containers/Workflow/WorkflowEntityPage';
import queryService from 'modules/queryService';
import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import { CVE_LIST_FRAGMENT_FOR_IMAGE } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import EntityList from '../../List/VulnMgmtList';
import VulnMgmtComponentOverview from './VulnMgmtComponentOverview';

const VulnMgmtComponent = ({ entityId, entityListType, search, entityContext, sort, page }) => {
    const overviewQuery = gql`
        query getComponent($id: ID!) {
            result: imageComponent(id: $id) {
                id
                name
                version
                priority
                vulns {
                    ...cveListFields
                }
            }
        }
        ${CVE_LIST_FRAGMENT_FOR_IMAGE}
    `;

    function getListQuery(listFieldName, fragmentName, fragment) {
        return gql`
        query getComponentSubEntity${entityListType}($id: ID!, $query: String) {
            result: deployment(id: $id) {
                id
                ${listFieldName}(query: $query) { ...${fragmentName} }
            }
        }
        ${fragment}
    `;
    }

    const queryOptions = {
        variables: {
            id: entityId,
            query: queryService.objectToWhereClause(search)
        }
    };

    return (
        <WorkflowEntityPage
            entityId={entityId}
            entityType={entityTypes.COMPONENT}
            entityListType={entityListType}
            useCase={useCases.VULN_MANAGEMENT}
            ListComponent={EntityList}
            OverviewComponent={VulnMgmtComponentOverview}
            overviewQuery={overviewQuery}
            getListQuery={getListQuery}
            search={search}
            sort={sort}
            page={page}
            queryOptions={queryOptions}
            entityContext={entityContext}
        />
    );
};

export default VulnMgmtComponent;
