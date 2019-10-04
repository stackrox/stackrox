import gql from 'graphql-tag';

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

function getEntityQuery() {
    return defaultQuery;
}

export default { getEntityQuery };
