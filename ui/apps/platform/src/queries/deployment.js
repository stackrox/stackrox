import { gql } from '@apollo/client';

export const DEPLOYMENT_FRAGMENT = gql`
    fragment deploymentFields on Deployment {
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
        serviceAccountID
        failingPolicies(query: $query) {
            id
            name
        }
        policyStatus(query: $query)
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
`;
export const DEPLOYMENT_QUERY = gql`
    query getDeployment($id: ID!, $query: String) {
        deployment(id: $id) {
            ...deploymentFields
        }
    }
    ${DEPLOYMENT_FRAGMENT}
`;

export const DEPLOYMENT_NAME = gql`
    query getDeployment($id: ID!) {
        deployment(id: $id) {
            id
            name
        }
    }
`;

export const DEPLOYMENTS_QUERY = gql`
    query deployments($query: String, $pagination: Pagination) {
        results: deployments(query: $query, pagination: $pagination) {
            id
            name
            clusterName
            clusterId
            namespace
            namespaceId
            serviceAccount
            serviceAccountID
            secretCount
            imageCount
            policyStatus
        }
        count: deploymentCount(query: $query)
    }
`;

export const DEPLOYMENTS_WITH_IMAGE = gql`
    query getDeployments($query: String) {
        deployments(query: $query) {
            id
            name
            clusterName
            namespace
            serviceAccount
        }
    }
`;
