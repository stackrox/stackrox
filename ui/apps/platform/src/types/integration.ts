export type IntegrationSource =
    | 'authProviders'
    | 'notifiers'
    | 'imageIntegrations'
    | 'backups'
    | 'authPlugins'
    | 'signatureIntegrations';

export type IntegrationType =
    | AuthPluginType
    | AuthProviderType
    | BackupIntegrationType
    | ImageIntegrationType
    | NotifierIntegrationType
    | SignatureIntegrationType;

export type AuthPluginType = 'scopedAccess';

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
    | 'anchore'
    | 'artifactory'
    | 'artifactregistry'
    | 'azure'
    | 'clair'
    | 'clairify'
    | 'docker'
    | 'dtr'
    | 'ecr'
    | 'google'
    | 'ibm'
    | 'nexus'
    | 'quay'
    | 'rhel'
    | 'tenable';

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
