import React, { ReactElement } from 'react';

import { ResourceName } from 'types/roleResources';

// First draft of description fields for possible future request from backend.
const resourceDescriptions: Record<ResourceName, string> = {
    APIToken: 'Read: View API tokens. Write: Add or revoke API tokens.',
    Alert: 'Read: View policy violations. Write: Resolve or edit policy violations.',
    AllComments:
        'Read: N/A Write: Delete comments from other users. All users can edit and delete their own comments by default. To add and remove comments or tags, you need a role with write access for the resource you are modifying.',
    AuthPlugin:
        'Read: View authentication plugins. Write: Modify authentication plugins (local administrator only).',
    AuthProvider:
        'Read: View configuration for authentication services. Write: Modify configuration for authentication services.',
    BackupPlugins:
        'Read: View backup integrations and configurations. Write: Modify backup integrations and configurations.',
    CVE: 'Internal use only',
    Cluster: 'Read: View secured clusters. Write: Add, modify, or delete secured clusters.',
    Compliance: 'Read: View compliance standards and results. Write: N/A',
    ComplianceRunSchedule:
        'Read: View scheduled compliance runs. Write: Add, modify, or delete scheduled compliance runs.',
    ComplianceRuns:
        'Read: View recent compliance runs and their completion status. Write: Trigger compliance runs.',
    Config: 'Read: View options for data retention, security notices, and other related configurations. Write: Modify options for data retention, security notices, and other related configurations.',
    DebugLogs:
        "Read: View the current logging verbosity level in Red Hat Advanced Cluster Security for Kubernetes components. Download diagnostic bundle. Note: diagnostic bundle contains information about all clusters and namespaces regardless of user's access scope. Don't give this permission to users with limited access scope. Write: Modify the logging verbosity level.",
    Deployment: 'Read: View deployments (workloads) in secured clusters. Write: N/A',
    Detection: 'Read: Check build-time policies against images or deployment YAMLs. Write: N/A',
    Group: 'Read: View the RBAC rules that match user metadata to Red Hat Advanced Cluster Security for Kubernetes roles. Write: Add, modify, or delete RBAC rules.',
    Image: 'Read: View images, their components, and their vulnerabilities. Write: N/A',
    ImageComponent: 'Internal use only',
    ImageIntegration:
        'Read: List image registry integrations. Write: Add, edit, or delete image registry integrations.',
    Indicator: 'Read: View process activity in deployments. Write: N/A',
    K8sRole:
        'Read: View roles for Kubernetes role-based access control in secured clusters. Write: N/A',
    K8sRoleBinding:
        'Read: View role bindings for Kubernetes role-based access control in secured clusters. Write: N/A',
    K8sSubject:
        'Read: View users and groups for Kubernetes role-based access control in secured clusters. Write: N/A',
    Licenses:
        'Read: View the status of the license for the StackRox Kubernetes Security Platform. Write: Upload a new license key.',
    Namespace: 'Read: View Kubernetes namespaces in secured clusters. Write: N/A',
    NetworkBaseline: 'Read: View network baseline results. Write: Modify network baselines.',
    NetworkGraph:
        'Read: View active and allowed network connections in secured clusters. Write: N/A',
    NetworkGraphConfig:
        'Read: View network graph configuration. Write: Modify network graph configuration.',
    NetworkPolicy:
        'Read: View network policies in secured clusters and simulate changes. Write: Apply network policy changes in secured clusters.',
    Node: 'Read: View Kubernetes nodes in secured clusters. Write: N/A',
    Notifier:
        'Read: View integrations for notification systems like email, Jira, or webhooks. Write: Add, modify, or delete integrations for notification systems.',
    Policy: 'Read: View system policies. Write: Add, modify, or delete system policies.',
    ProbeUpload:
        'Read: Read manifests for the uploaded probe files. Write: Upload support packages to Central.',
    ProcessWhitelist:
        'Read: View process baselines. Write: Add or remove processes from baselines.',
    Risk: 'Read: View Risk results. Write: N/A',
    Role: 'Read: View Red Hat Advanced Cluster Security for Kubernetes RBAC roles and permission sets. Write: Add, modify, or delete roles and permission sets.',
    ScannerBundle: 'Read: Download the scanner bundle. Write: N/A',
    ScannerDefinitions:
        'Read: List image scanner integrations. Write: Add, modify, or delete image scanner integrations.',
    Secret: 'Read: View metadata about secrets in secured clusters. Write: N/A',
    SensorUpgradeConfig:
        'Read: Check the status of automatic upgrades. Write: Disable or enable automatic upgrades for secured clusters.',
    ServiceAccount: 'Read: List Kubernetes service accounts in secured clusters. Write: N/A',
    ServiceIdentity:
        'Read: View metadata about Red Hat Advanced Cluster Security for Kubernetes service-to-service authentication. Write: Revoke or reissue service-to-service authentication credentials.',
    User: 'Read: View users that have accessed the Red Hat Advanced Cluster Security for Kubernetes instance, including the metadata that the authentication provider provides about them. Write: N/A',
    VulnerabilityManagementRequests:
        'Read: View all pending deferral or false positive requests for vulnerabilities. Write: Request a deferral on a vulnerability, mark it as a false positive or move a pending or previously approved request (made by the same user) back to observed.',
    VulnerabilityManagementApprovals:
        'Read: View all pending deferral or false positive requests for vulnerabilities. Write: Approve or deny any pending deferral or false positive requests and move any previously approved requests back to observed.',
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
