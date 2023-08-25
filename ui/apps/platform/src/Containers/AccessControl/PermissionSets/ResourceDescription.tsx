import React, { ReactElement } from 'react';

import { ResourceName } from 'types/roleResources';

// Description for permission resource types. 'Read:' and 'Write:' tokens have
// special meaning and mark parts related to respective operations.
const resourceDescriptions: Record<ResourceName, string> = {
    Access: 'Read: View configuration for authentication and authorization, such as authentication services, roles, groups, and users. Write: Modify configuration for authentication and authorization.',
    Administration:
        'Read: View platform configuration (e.g., network graph, sensor, debugging configs). Write: Modify platform configuration, delete comments from other users.',
    Alert: 'Read: View policy violations. Write: Resolve or edit policy violations.',
    CVE: 'Internal use only',
    Cluster: 'Read: View secured clusters. Write: Add, modify, or delete secured clusters.',
    Compliance:
        'Read: View compliance standards, results, and runs. Write: Add, modify, or delete scheduled compliance runs.',
    Deployment: 'Read: View deployments (workloads) in secured clusters. Write: N/A',
    DeploymentExtension:
        'Read: View network, process listening on ports, and process baseline extensions, risk score of deployments. Write: Modify the process, process listening on ports, and network baseline extensions of deployments.',
    Detection: 'Read: Check build-time policies against images or deployment YAMLs. Write: N/A',
    Image: 'Read: View images, their components, and their vulnerabilities. Write: N/A',
    Integration:
        'Read: View integrations and their configuration. This includes backup, registry, image signature and notification systems, API tokens. Write: Add, modify, delete integrations and their configuration, API tokens.',
    K8sRole:
        'Read: View roles for Kubernetes role-based access control in secured clusters. Write: N/A',
    K8sRoleBinding:
        'Read: View role bindings for Kubernetes role-based access control in secured clusters. Write: N/A',
    K8sSubject:
        'Read: View users and groups for Kubernetes role-based access control in secured clusters. Write: N/A',
    Namespace: 'Read: View Kubernetes namespaces in secured clusters. Write: N/A',
    NetworkGraph:
        'Read: View active and allowed network connections in secured clusters. Write: N/A',
    NetworkPolicy:
        'Read: View network policies in secured clusters and simulate changes. Write: Apply network policy changes in secured clusters.',
    Node: 'Read: View Kubernetes nodes in secured clusters. Write: N/A',
    Secret: 'Read: View metadata about secrets in secured clusters. Write: N/A',
    ServiceAccount: 'Read: List Kubernetes service accounts in secured clusters. Write: N/A',
    VulnerabilityManagementApprovals:
        'Read: View all pending deferral or false positive requests for vulnerabilities. Write: Approve or deny any pending deferral or false positive requests and move any previously approved requests back to observed.',
    VulnerabilityManagementRequests:
        'Read: View all pending deferral or false positive requests for vulnerabilities. Write: Request a deferral on a vulnerability, mark it as a false positive or move a pending or previously approved request (made by the same user) back to observed.',
    WatchedImage:
        'Read: View undeployed watched images monitored. Write: Configure watched images.',
    WorkflowAdministration:
        'Read: View all resource collections. Write: Add, modify, or delete resource collections.',
};

export type ResourceDescriptionProps = {
    resource: string;
};

export function ResourceDescription({ resource }: ResourceDescriptionProps): ReactElement {
    // The description becomes the prop for possible future request from backend.
    const description = resourceDescriptions[resource] ?? '';
    const readIndex = description.indexOf('Read: ');
    const writeIndex = description.indexOf(' Write: ');

    if (readIndex === 0 && writeIndex !== -1) {
        return (
            <>
                <p>
                    <strong>Read</strong>: {description.slice(6, writeIndex)}
                </p>
                <p>
                    {/* eslint-disable-next-line @typescript-eslint/restrict-plus-operands */}
                    <strong>Write</strong>: {description.slice(writeIndex + 8)}
                </p>
            </>
        );
    }

    return <p>{description}</p>;
}

export default ResourceDescription;
