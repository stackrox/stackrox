import React from 'react';
import { workflowEntityPropTypes, workflowEntityDefaultProps } from 'constants/entityPageProps';
import useCases from 'constants/useCaseTypes';
import queryService from 'modules/queryService';
import entityTypes from 'constants/entityTypes';
import gql from 'graphql-tag';
import WorkflowEntityPage from 'Containers/Workflow/WorkflowEntityPage';
import VulnMgmtCveOverview from './VulnMgmtCveOverview';
import VulnMgmtList from '../../List/VulnMgmtList';

const VulmMgmtCve = ({ entityId, entityListType, search, entityContext, sort, page }) => {
    const overviewQuery = gql`
        query getCve($id: ID!) {
            result: vulnerability(id: $id) {
                id: cve
                cve
                envImpact
                cvss
                scoreVersion
                link # for View on NVD website
                vectors {
                    __typename
                    ... on CVSSV2 {
                        impactScore
                        exploitabilityScore
                        vector
                    }
                    ... on CVSSV3 {
                        impactScore
                        exploitabilityScore
                        vector
                    }
                }
                publishedOn
                lastModified
                summary
                fixedByVersion
                isFixable
                lastScanned
                componentCount
                imageCount
                deploymentCount
            }
        }
    `;

    function getListQuery(listFieldName, fragmentName, fragment) {
        return gql`
        query getCve${entityListType}($id: ID!, $query: String) {
            result: vulnerability(id: $id) {
                cve
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
            entityType={entityTypes.CVE}
            entityListType={entityListType}
            useCase={useCases.VULN_MANAGEMENT}
            ListComponent={VulnMgmtList}
            OverviewComponent={VulnMgmtCveOverview}
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

VulmMgmtCve.propTypes = workflowEntityPropTypes;
VulmMgmtCve.defaultProps = workflowEntityDefaultProps;

export default VulmMgmtCve;
