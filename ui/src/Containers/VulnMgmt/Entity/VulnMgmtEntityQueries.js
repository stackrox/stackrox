import gql from 'graphql-tag';

import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import queryService from 'modules/queryService';

const defaultQuery = gql`
    query getDeployment($id: ID!) {
        deployment(id: $id) {
            id
            annotations {
                key
                value
            }
            cluster {
                id
                name
            }
            hostNetwork: id
            imagePullSecrets
            inactive
            labels {
                key
                value
            }
            name
            namespace
            namespaceId
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
            serviceAccount
            serviceAccountID
            failingPolicyCount
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

function getQuery(entityListType) {
    if (!entityListType) return defaultQuery;
    const { listFieldName, fragmentName, fragment } = queryService.getFragmentInfo(
        entityTypes.DEPLOYMENT,
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

export default { getQuery };
