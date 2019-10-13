import gql from 'graphql-tag';

import entityTypes from 'constants/entityTypes';
import { DEPLOYMENTS_QUERY } from 'queries/deployment';
import { IMAGES } from 'queries/image';
import { NAMESPACES_QUERY } from 'queries/namespace';
import { POLICIES } from 'queries/policy';

const CLUSTERS_QUERY = gql`
    query clusters($query: String) {
        results: clusters(query: $query) {
            id
            name
            # cves
            status {
                lastContact
                orchestratorMetadata {
                    version
                }
            }
            # created
            namespaceCount
            deploymentCount
            policyCount
            policyStatus {
                failingPolicies {
                    id
                }
            }
            # latestViolation
            # risk
        }
    }
`;

// TODO: delete this query
const CVES_QUERY = gql`
    query {
        vulnerabilities {
            cve
            cvss
            summary
            fixedByVersion
            isFixable
            lastScanned
            components {
                name
                version
            }
            images {
                id
                name {
                    fullName
                    registry
                    remote
                    tag
                }
            }
            deployments {
                id
                name
            }
        }
    }
`;
const LIST_QUERIES = {
    [entityTypes.CLUSTER]: CLUSTERS_QUERY,
    [entityTypes.CVE]: CVES_QUERY,
    [entityTypes.DEPLOYMENT]: DEPLOYMENTS_QUERY,
    [entityTypes.IMAGE]: IMAGES,
    [entityTypes.NAMESPACE]: NAMESPACES_QUERY,
    [entityTypes.POLICY]: POLICIES
};

function getListQuery(listType) {
    return LIST_QUERIES[listType];
}

export default {
    getListQuery
};
