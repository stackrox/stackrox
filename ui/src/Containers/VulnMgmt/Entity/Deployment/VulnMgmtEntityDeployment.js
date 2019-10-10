import React from 'react';
import isGQLLoading from 'utils/gqlLoading';
import { useQuery } from 'react-apollo';
import useCases from 'constants/useCaseTypes';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import queryService from 'modules/queryService';
import entityTypes from 'constants/entityTypes';
import gql from 'graphql-tag';
import Loader from 'Components/Loader';
import VulnMgmtDeploymentOverview from './VulnMgmtDeploymentOverview';
import EntityList from '../../List/VulnMgmtList';

const VulmMgmtDeployment = ({ entityId, entityListType, search, entityContext }) => {
    // TODO: templatize this so this doesn't have to be repeated for every entity type component
    const overviewQuery = gql`
    query getDeployment($id: ID!, $query: String) {
        deployment(id: $id) {
            id
            annotations {
                key
                value
            }
            ${entityContext[entityTypes.CLUSTER] ? '' : 'cluster { id name}'}
            hostNetwork: id
            imagePullSecrets
            inactive
            labels {
                key
                value
            }
            name
            ${entityContext[entityTypes.NAMESPACE] ? '' : 'namespace namespaceId'}
            ports {
                containerPort
                exposedPort
                exposure
                exposureInfos {
                    externalHostnames
                    externalIps
                    level
                    nodePort
                    serviceClusterIp
                    serviceId
                    serviceName
                    servicePort
                }
                name
                protocol
            }
            priority
            replicas
            ${entityContext[entityTypes.SERVICE_ACCOUNT] ? '' : 'serviceAccount serviceAccountID'}
            failingPolicyCount(query: $query)
            tolerations {
                key
                operator
                taintEffect
                value
            }
            type
            created
            secretCount
            imageCount
        }
    }
    `;

    function getQuery() {
        if (!entityListType) return overviewQuery;
        const { listFieldName, fragmentName, fragment } = queryService.getFragmentInfo(
            entityTypes.CVE,
            entityListType,
            useCases.VULN_MANAGEMENT
        );

        return gql`
        query getDeployment${entityListType}($id: ID!, $query: String) {
            deployment(id: $id) {
                id
                ${listFieldName}(query: $query) { ...${fragmentName} }
            }
        }
        ${fragment}
    `;
    }

    const variables = {
        variables: {
            cacheBuster: new Date().getUTCMilliseconds(),
            id: entityId,
            query: queryService.objectToWhereClause(search)
        }
    };
    const query = getQuery();

    const { loading, data } = useQuery(query, variables);
    if (isGQLLoading(loading, data)) return <Loader transparent />;
    const { deployment } = data;
    return entityListType ? (
        <EntityList
            entityListType={entityListType}
            data={deployment}
            search={search}
            entityContext={{ ...entityContext, [entityTypes.DEPLOYMENT]: entityId }}
        />
    ) : (
        <VulnMgmtDeploymentOverview data={deployment} entityContext={entityContext} />
    );
};

VulmMgmtDeployment.propTypes = entityComponentPropTypes;
VulmMgmtDeployment.defaultProps = entityComponentDefaultProps;

export default VulmMgmtDeployment;
