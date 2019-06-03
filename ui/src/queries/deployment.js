import gql from 'graphql-tag';

export const DEPLOYMENT_QUERY = gql`
    query getDeployment($id: ID!) {
        deployment(id: $id) {
            id
            annotations {
                key
                value
            }
            clusterId
            clusterName
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
            tolerations {
                key
                operator
                taintEffect
                value
            }
            type
            updatedAt
            version
        }
    }
`;

export const DEPLOYMENTS_QUERY = gql`
    query getDeployments {
        results: deployments {
            id
            name
            clusterName
            namespace
            alerts {
                id
            }
            serviceAccount
        }
    }
`;
