import integrationsList from 'Containers/Integrations/integrationsList';

export type IntegrationSource =
    | 'authProviders'
    | 'notifiers'
    | 'imageIntegrations'
    | 'backups'
    | 'authPlugins';
export type IntegrationType =
    | 'oidc'
    | 'auth0'
    | 'saml'
    | 'iap'
    | 'generic'
    | 'awsSecurityHub'
    | 'jira'
    | 'email'
    | 'slack'
    | 'teams'
    | 'cscc'
    | 'splunk'
    | 'sumologic'
    | 'pagerduty'
    | 'syslog'
    | 'tenable'
    | 'docker'
    | 'dtr'
    | 'artifactory'
    | 'quay'
    | 'clair'
    | 'clairify'
    | 'artifactregistry'
    | 'google'
    | 'ecr'
    | 'nexus'
    | 'azure'
    | 'anchore'
    | 'ibm'
    | 'rhel'
    | 's3'
    | 'gcs'
    | 'scopedAccess'
    | 'apitoken'
    | 'clusterInitBundle';

export type Integration = {
    type: IntegrationType;
    id: string;
};

export type IntegrationTile = {
    source: string;
    type: string;
    label: string;
};

export function getIntegrationLabel(source: string, type: string): string {
    const integrationTile = integrationsList[source]?.find(
        (integration: IntegrationTile) => integration.type === type
    ) as IntegrationTile;
    return integrationTile.label;
}

export function getIsAPIToken(source: IntegrationSource, type: IntegrationType): boolean {
    return source === 'authProviders' && type === 'apitoken';
}

export function getIsClusterInitBundle(source: IntegrationSource, type: IntegrationType): boolean {
    return source === 'authProviders' && type === 'clusterInitBundle';
}
