import { AuthMachineToMachineConfig } from 'services/MachineAccessService';

export type IntegrationSource =
    | 'authProviders'
    | 'notifiers'
    | 'imageIntegrations'
    | 'backups'
    | 'signatureIntegrations'
    | 'cloudSources';

export type IntegrationType =
    | AuthProviderType
    | BackupIntegrationType
    | ImageIntegrationType
    | NotifierIntegrationType
    | SignatureIntegrationType
    | CloudSourceIntegrationType;

export type AuthProviderType = 'apitoken' | 'clusterInitBundle' | 'clusterRegistrationSecret' | 'machineAccess';

// Investigate why the following occur in tableColumnDescriptor but not in integrationsList:
/*
    | 'oidc'
    | 'auth0'
    | 'saml'
    | 'iap'
*/

export type BackupIntegrationType = 'gcs' | 's3' | 's3compatible';

export type ImageIntegrationType =
    | 'artifactory'
    | 'artifactregistry'
    | 'azure'
    | 'clair'
    | 'clairV4'
    | 'clairify'
    | 'docker'
    | 'ecr'
    | 'ghcr'
    | 'google'
    | 'ibm'
    | 'nexus'
    | 'quay'
    | 'rhel'
    | 'scannerv4';

export type NotifierIntegrationType =
    | 'awsSecurityHub'
    | 'acscsEmail'
    | 'cscc'
    | 'email'
    | 'generic'
    | 'jira'
    | 'microsoftSentinel'
    | 'pagerduty'
    | 'slack'
    | 'splunk'
    | 'sumologic'
    | 'syslog'
    | 'teams';

export type SignatureIntegrationType = 'signature';

export type CloudSourceIntegrationType = 'paladinCloud' | 'ocm';

export type BaseIntegration = {
    id: string;
    name: string;
};

export type AuthProviderIntegration = BaseIntegration | AuthMachineToMachineConfig;
