export type IntegrationSource =
    | 'authProviders'
    | 'notifiers'
    | 'imageIntegrations'
    | 'backups'
    | 'signatureIntegrations';

export type IntegrationType =
    | AuthProviderType
    | BackupIntegrationType
    | ImageIntegrationType
    | NotifierIntegrationType
    | SignatureIntegrationType;

export type AuthProviderType = 'apitoken' | 'clusterInitBundle';

// Investigate why the following occur in tableColumnDescriptor but not in integrationsList:
/*
    | 'oidc'
    | 'auth0'
    | 'saml'
    | 'iap'
*/

export type BackupIntegrationType = 'gcs' | 's3';

export type ImageIntegrationType =
    | 'artifactory'
    | 'artifactregistry'
    | 'azure'
    | 'clairV4'
    | 'clairify'
    | 'docker'
    | 'ecr'
    | 'google'
    | 'ibm'
    | 'nexus'
    | 'quay'
    | 'rhel';

export type NotifierIntegrationType =
    | 'awsSecurityHub'
    | 'cscc'
    | 'email'
    | 'generic'
    | 'jira'
    | 'pagerduty'
    | 'slack'
    | 'splunk'
    | 'sumologic'
    | 'syslog'
    | 'teams';

export type SignatureIntegrationType = 'signature';

export type BaseIntegration = {
    id: string;
    name: string;
};
