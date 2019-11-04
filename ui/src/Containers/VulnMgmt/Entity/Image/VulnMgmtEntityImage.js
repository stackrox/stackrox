import React from 'react';
import { workflowEntityPropTypes, workflowEntityDefaultProps } from 'constants/entityPageProps';
import useCases from 'constants/useCaseTypes';
import queryService from 'modules/queryService';
import entityTypes from 'constants/entityTypes';
import gql from 'graphql-tag';
import WorkflowEntityPage from 'Containers/Workflow/WorkflowEntityPage';
import VulnMgmtImageOverview from './VulnMgmtImageOverview';
import EntityList from '../../List/VulnMgmtList';

const VulnMgmtImage = ({ entityId, entityListType, search, entityContext, sort, page }) => {
    const overviewQuery = gql`
        query getImage($id: ID!${entityListType ? ', $query: String' : ''}) {
            result: image(sha: $id) {
                id
                lastUpdated
                ${entityContext[entityTypes.DEPLOYMENT] ? '' : 'deploymentCount'}
                metadata {
                    layerShas
                    v1 {
                        created
                        layers {
                            instruction
                            created
                            value
                        }
                    }
                    v2 {
                        digest
                    }
                }
                vulnCount
                priority
                topVuln {
                    cvss
                    scoreVersion
                }
                name {
                    fullName
                    registry
                    remote
                    tag
                }
                scan {
                    scanTime
                    components {
                        id
                        priority
                        name
                        layerIndex
                        version
                        license {
                            name
                            type
                            url
                        }
                        vulnCounter {
                            all {
                                total
                                fixable
                            }
                            low {
                                total
                                fixable
                            }
                            medium {
                                total
                                fixable
                            }
                            high {
                                total
                                fixable
                            }
                            critical {
                                total
                                fixable
                            }
                        }
                        vulns {
                            cve
                            isFixable
                            lastScanned
                            impactScore
                            publishedOn
                            cvss
                            link
                            summary
                        }
                    }
                }
            }
        }
    `;

    function getListQuery(listFieldName, fragmentName, fragment) {
        return gql`
        query getImage${entityListType}($id: ID!, $query: String) {
            result: image(sha: $id) {
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
            entityType={entityTypes.IMAGE}
            entityListType={entityListType}
            useCase={useCases.VULN_MANAGEMENT}
            ListComponent={EntityList}
            OverviewComponent={VulnMgmtImageOverview}
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

VulnMgmtImage.propTypes = workflowEntityPropTypes;
VulnMgmtImage.defaultProps = workflowEntityDefaultProps;

export default VulnMgmtImage;
