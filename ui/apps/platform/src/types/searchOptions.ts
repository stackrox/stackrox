// Frontend counterpart to backend source of truth:
// https://github.com/stackrox/stackrox/blob/master/pkg/search/options.go

// Minimal conversion: original order with empty comments in place of empty lines.
export type SearchFieldLabel =
    | 'Cluster'
    | 'Cluster ID'
    | 'Cluster Label'
    | 'Cluster Scope'
    | 'Cluster Type'
    | 'Cluster Discovered Time'
    | 'Cluster Platform Type'
    | 'Cluster Kubernetes Version'
    // cluster health search fields
    | 'Cluster Status'
    | 'Sensor Status'
    | 'Collector Status'
    | 'Admission Control Status'
    | 'Scanner Status'
    | 'Last Contact'
    //
    | 'Policy ID'
    | 'Enforcement'
    | 'Policy'
    | 'Policy Category'
    | 'Policy Category ID'
    //
    | 'Lifecycle Stage'
    | 'Description'
    | 'Category'
    | 'Severity'
    | 'SEVERITY' // TODO replace UPPERCASE with Title Case
    | 'Disabled'
    //
    | 'CVE ID'
    | 'CVE'
    | 'CVE Type'
    | 'CVE Published On'
    | 'CVE Fix Available Timestamp'
    | 'CVE Created Time'
    | 'CVE Snoozed'
    | 'CVE Snooze Expiry'
    | 'CVSS'
    | 'NVD CVSS'
    | 'Impact Score'
    | 'Vulnerability State'
    | 'CVE Orphaned'
    | 'CVE Orphaned Time'
    | 'EPSS Probability'
    | 'Known Exploit' // frontend only pending backend implementation see obsolete #16887
    | 'Known Ransomware Campaign' // frontend only pending backend implementation see obsolete #16887
    | 'Advisory Name'
    | 'Advisory Link'
    //
    | 'CVE Info'
    //
    | 'Component'
    | 'Component ID'
    | 'Component Version'
    | 'Component Source'
    | 'Component Location'
    | 'Component Top CVSS'
    | 'Dockerfile Instruction Keyword'
    | 'Dockerfile Instruction Value'
    | 'First Image Occurrence Timestamp'
    | 'First System Occurrence Timestamp'
    | 'Host IPC'
    | 'Host Network'
    | 'Host PID'
    | 'Image Created Time'
    | 'Image'
    | 'Image Sha'
    | 'Image Signature Fetched Time'
    | 'Image Signature Verified By'
    | 'Image Registry'
    | 'Image Remote'
    | 'Image Scan Time'
    | 'Node Scan Time'
    | 'Image OS'
    | 'Image Tag'
    | 'Image User'
    | 'Image Command'
    | 'Image CVE Count'
    | 'Image Entrypoint'
    | 'Image Label'
    | 'Image Volumes'
    | 'Fixable'
    | 'FIXABLE' // TODO replace UPPERCASE with Title Case
    | 'Fixed By'
    | 'Cluster CVE Fixed By'
    | 'Cluster CVE Fixable'
    | 'CLUSTER CVE FIXABLE' // TODO replace UPPERCASE with Title Case
    | 'Fixable CVE Count'
    | 'Last Updated'
    | 'Image Top CVSS'
    | 'Node Top CVSS'
    | 'Image ID'
    | 'Unknown CVE Count'
    | 'Fixable Unknown CVE Count'
    | 'Critical CVE Count'
    | 'Fixable Critical CVE Count'
    | 'Important CVE Count'
    | 'Fixable Important CVE Count'
    | 'Moderate CVE Count'
    | 'Fixable Moderate CVE Count'
    | 'Low CVE Count'
    | 'Fixable Low CVE Count'
    //
    // Base Image
    | 'Base Image Id'
    | 'Base Image Repository'
    | 'Base Image Tag'
    | 'Base Image Active'
    | 'Base Image Manifest Digest'
    | 'Base Image First Layer Digest'
    | 'Base Image Layer Digest'
    | 'Base Image Index'
    | 'Base Image Discovered At'
    //
    // Deployment related fields
    | 'Add Capabilities'
    | 'Allow Privilege Escalation'
    | 'AppArmor Profile'
    | 'Automount Service Account Token'
    | 'Deployment Annotation'
    | 'CPU Cores Limit'
    | 'CPU Cores Request'
    | 'Container ID'
    | 'Container Image Digest'
    | 'Container Name'
    | 'Deployment ID'
    | 'Deployment'
    | 'Deployment Label'
    | 'Deployment Type'
    | 'Drop Capabilities'
    | 'Environment Key'
    | 'Environment Value'
    | 'Environment Variable Source'
    | 'Exposed Node Port'
    | 'Exposing Service'
    | 'Exposing Service Port'
    | 'Exposure Level'
    | 'External IP'
    | 'External Hostname'
    | 'Image Pull Secret'
    | 'Liveness Probe Defined'
    | 'Max Exposure Level'
    | 'Memory Limit (MB)'
    | 'Memory Request (MB)'
    | 'Mount Propagation'
    | 'Orchestrator Component'
    | 'Platform Component'
    // PolicyViolated is a fake search field to filter deployments that have violation.
    // This is handled/supported only by deployments sub-resolver of policy resolver.
    // Note that 'Policy Violated=false' is not yet supported.
    | 'Policy Violated'
    | 'Port'
    | 'Port Protocol'
    // Priority is used in risk datastore internally.
    | 'Priority'
    | 'Administration Usage Timestamp'
    | 'Administration Usage Nodes'
    | 'Administration Usage CPU Units'
    | 'Cluster Risk Priority'
    | 'Namespace Risk Priority'
    | 'Privileged'
    | 'Process Tag'
    | 'Read Only Root Filesystem'
    | 'Replicas'
    | 'Readiness Probe Defined'
    | 'Secret ID'
    | 'Secret'
    | 'Secret Path'
    | 'Seccomp Profile Type'
    | 'Service Account'
    | 'Service Account Permission Level'
    | 'Service Account Label'
    | 'Service Account Annotation'
    | 'Created'
    | 'Volume Name'
    | 'Volume Source'
    | 'Volume Destination'
    | 'Volume ReadOnly'
    | 'Volume Type'
    | 'Taint Key'
    | 'Taint Value'
    | 'Toleration Key'
    | 'Toleration Value'
    | 'Taint Effect'
    //
    | 'Alert ID'
    | 'Violation'
    | 'Violation State'
    | 'Violation Time'
    | 'Tag'
    | 'Entity Type'
    //
    // Pod Search fields
    | 'Pod UID'
    | 'Pod ID'
    | 'Pod Name'
    | 'Pod Label'
    //
    // ProcessIndicator Search fields
    | 'Process ID'
    | 'Process Path'
    | 'Process Name'
    | 'Process Arguments'
    | 'Process Ancestor'
    | 'Process UID'
    | 'Process Creation Time'
    | 'Process Container Start Time'
    //
    // FileActivity Search fields
    | 'Effective Path'
    | 'Actual Path'
    | 'File Operation'
    //
    // ProcessListeningOnPort Search fields
    | 'Closed'
    | 'Closed Time'
    //
    // Secret search fields
    | 'Secret Type'
    | 'Cert Expiration'
    | 'Image Pull Secret Registry'
    //
    // Compliance search fields
    | 'Standard'
    | 'Standard ID'
    //
    | 'Control Group ID'
    | 'Control Group'
    //
    | 'Control ID'
    | 'Control'
    //
    | 'Compliance Operator Integration ID'
    | 'Compliance Operator Version'
    | 'Compliance Scan Name'
    | 'Compliance Operator Installed'
    | 'Compliance Rule Severity'
    | 'Compliance Operator Status'
    | 'Compliance Check Status'
    | 'Compliance Rule Name'
    | 'Compliance Profile ID'
    | 'Compliance Profile Name'
    | 'Compliance Config Profile Name'
    | 'Compliance Profile Product Type'
    | 'Compliance Profile Version'
    | 'Compliance Standard'
    | 'Compliance Control'
    | 'Compliance Scan Config ID'
    | 'Compliance Scan Config Name'
    | 'Compliance Check ID'
    | 'Compliance Check UID'
    | 'Compliance Check Name'
    | 'Compliance Check Rationale'
    | 'Compliance Check Last Started Time'
    | 'Compliance Scan Config Last Updated Time'
    | 'Compliance Check Result Created Time'
    | 'Compliance Scan Last Executed Time'
    | 'Compliance Scan Last Started Time'
    | 'Compliance Rule Type'
    | 'Compliance Scan Setting Binding Name'
    | 'Compliance Suite Name'
    | 'Compliance Scan Result'
    | 'Profile Ref ID'
    | 'Scan Ref ID'
    | 'Rule Ref ID'
    | 'Compliance Remediation Name'
    | 'Compliance Benchmark Name'
    | 'Compliance Benchmark Short Name'
    | 'Compliance Benchmark Version'
    | 'Compliance Report Name'
    | 'Compliance Report State'
    | 'Compliance Report Started Time'
    | 'Compliance Report Completed Time'
    | 'Compliance Report Request Type'
    | 'Compliance Report Notification Method'
    //
    // Node search fields
    | 'Node'
    | 'Node ID'
    | 'Operating System'
    | 'Container Runtime'
    | 'Node Join Time'
    | 'Node Label'
    | 'Node Annotation'
    //
    // Namespace Search Fields
    | 'Namespace ID'
    | 'Namespace'
    | 'Namespace Annotation'
    | 'Namespace Label'
    //
    // Role Search Fields
    | 'Role ID'
    | 'Role'
    | 'Role Label'
    | 'Role Annotation'
    | 'Cluster Role'
    //
    // Role Binding Search Fields
    | 'Role Binding ID'
    | 'Role Binding'
    | 'Role Binding Label'
    | 'Role Binding Annotation'
    //
    // Subject search fields
    | 'Subject Kind'
    | 'Subject'
    //
    // General
    | 'Created Time'
    //
    // Inactive Deployment
    | 'Inactive Deployment'
    //
    // Risk Search Fields
    | 'Risk Score'
    | 'Node Risk Score'
    | 'Deployment Risk Score'
    | 'Image Risk Score'
    | 'Component Risk Score'
    | 'Risk Subject Type'
    | 'Component Layer Type'
    //
    | 'Policy Last Updated'
    //
    // Following are helper fields used for sorting
    // For example, "SORTPolicyName" field should be used to sort policies when the query sort field is "PolicyName"
    | 'SORT_Policy'
    | 'SORT_Lifecycle Stage'
    | 'SORT_Enforcement'
    //
    // Omit derived fields
    //
    // External network sources fields
    | 'Default External Source'
    | 'Discovered External Source'
    | 'External Source Address'
    //
    // Report configurations search fields
    | 'Report Name'
    | 'Report Type'
    | 'Report Configuration ID'
    // View Based report search fields
    | 'Area Of Concern'
    //
    // Resource alerts search fields
    | 'Resource'
    | 'Resource Type'
    //
    // Vulnerability Watch Request fields
    | 'Request Name'
    | 'Request Status'
    | 'Expired Request'
    | 'Expiry Type'
    | 'Request Expiry Time'
    | 'Request Expires When Fixed'
    | 'Requested Vulnerability State'
    | 'User ID'
    | 'User Name'
    | 'Image Registry Scope'
    | 'Image Remote Scope'
    | 'Image Tag Scope'
    | 'Requester User ID'
    | 'Requester User Name'
    | 'Approver User ID'
    | 'Approver User Name'
    | 'Deferral Update CVEs'
    | 'False Positive Update CVEs'
    //
    | 'Compliance Domain ID'
    | 'Compliance Run ID'
    | 'Compliance Run Finished Timestamp'
    //
    // Resource Collection fields
    | 'Collection ID'
    | 'Collection Name'
    | 'Embedded Collection ID'
    //
    // Group fields
    | 'Group Auth Provider'
    | 'Group Key'
    | 'Group Value'
    //
    // API Token fields
    | 'Expiration'
    | 'Revoked'
    //
    // Version fields
    | 'Version'
    | 'Minimum Sequence Number'
    | 'Current Sequence Number'
    | 'Last Persisted'
    //
    // Blob store fields
    | 'Blob Name'
    | 'Blob Length'
    | 'Blob Modified On'
    //
    // Report Metadata fields
    | 'Report State'
    | 'Report Init Time'
    | 'Report Completion Time'
    | 'Report Request Type'
    | 'Report Notification Method'
    //
    // Event fields.
    | 'Event Domain'
    | 'Event Type'
    | 'Event Level'
    | 'Event Occurrence'
    //
    // Integration fields.
    | 'Integration ID'
    | 'Integration Name'
    | 'Integration Type'
    //
    // AuthProvider fields.
    | 'AuthProvider Name'
    //
    // Virtual Machine fields.
    | 'Virtual Machine ID'
    | 'Virtual Machine Name'
    | 'SCANNABLE'; // frontend only pending backend TODO replace SCANNABLE with Scan Status
