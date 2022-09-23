import React, { ReactElement } from 'react';

import { ResourceName } from 'types/roleResources';

// Description for permission resource types. 'Read:' and 'Write:' tokens have
// special meaning and mark parts related to respective operations.
const resourceDescriptions: Record<ResourceName, string> = {
    Access: 'Read: View configuration for authentication and authorization, such as authentication services, roles, groups, and users. Write: Modify configuration for authentication and authorization.',
    Administration:
        'Read: View platform configuration (e.g., network graph, sensor, debugging configs). Write: Modify platform configuration, delete comments from other users.',
    Alert: 'Read: View policy violations. Write: Resolve or edit policy violations.',
    AllComments:
        'Read: N/A Write: Delete comments from other users. All users can edit and delete their own comments by default. To add and remove comments or tags, you need a role with write access for the resource you are modifying.',
    CVE: 'Internal use only',
    Cluster: 'Read: View secured clusters. Write: Add, modify, or delete secured clusters.',
    Compliance:
        'Read: View compliance standards, results, and runs. Write: Add, modify, or delete scheduled compliance runs.',
    ComplianceRunSchedule:
        'Read: View scheduled compliance runs. Write: Add, modify, or delete scheduled compliance runs.',
    ComplianceRuns:
        'Read: View recent compliance runs and their completion status. Write: Trigger compliance runs.',
    Config: 'Read: View options for data retention, security notices, and other related configurations. Write: Modify options for data retention, security notices, and other related configurations.',
    DebugLogs:
        "Read: View the current logging verbosity level of all components, including Central, Scanner, Sensor, Collector, and Admission controller. Download the diagnostic bundle. Important: The diagnostic bundle contains sensitive information, not dependent on the user's role and access scope. The diagnostic bundle includes information about all clusters and namespaces, access control, notifier integrations, and system configuration. Do not give this permission to users with limited access scope. Write: Modify the logging verbosity level.",
    Deployment: 'Read: View deployments (workloads) in secured clusters. Write: N/A',
    DeploymentExtension:
        'Read: View network and process baseline extensions, risk score of deployments. Write: Modify the process and network baseline extensions of deployments.',
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
    NetworkGraphConfig:
        'Read: View network graph configuration. Write: Modify network graph configuration.',
    NetworkPolicy:
        'Read: View network policies in secured clusters and simulate changes. Write: Apply network policy changes in secured clusters.',
    Node: 'Read: View Kubernetes nodes in secured clusters. Write: N/A',
    Policy: 'Read: View system policies. Write: Add, modify, or delete system policies.',
    ProbeUpload:
        'Read: Read manifests for the uploaded probe files. Write: Upload support packages to Central.',
    ScannerBundle: 'Read: Download the scanner bundle. Write: N/A',
    ScannerDefinitions:
        'Read: List image scanner integrations. Write: Add, modify, or delete image scanner integrations.',
    Secret: 'Read: View metadata about secrets in secured clusters. Write: N/A',
    SensorUpgradeConfig:
        'Read: Check the status of automatic upgrades. Write: Disable or enable automatic upgrades for secured clusters.',
    ServiceAccount: 'Read: List Kubernetes service accounts in secured clusters. Write: N/A',
    ServiceIdentity:
        'Read: View metadata about service-to-service authentication. Write: Revoke or reissue service-to-service authentication credentials.',
    VulnerabilityManagementApprovals:
        'Read: View all pending deferral or false positive requests for vulnerabilities. Write: Approve or deny any pending deferral or false positive requests and move any previously approved requests back to observed.',
    VulnerabilityManagementRequests:
        'Read: View all pending deferral or false positive requests for vulnerabilities. Write: Request a deferral on a vulnerability, mark it as a false positive or move a pending or previously approved request (made by the same user) back to observed.',
    VulnerabilityReports:
        'Read: View all vulnerability report configurations. Write: Add, modify or delete vulnerability report configurations.',
    WatchedImage:
        'Read: View undeployed watched images monitored. Write: Configure watched images.',
};

export type ResourceDescriptionProps = {
    resource: string;
};

function ResourceDescription({ resource }: ResourceDescriptionProps): ReactElement {
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
