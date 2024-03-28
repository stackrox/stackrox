import { BaseBackupIntegration } from 'types/externalBackup.proto';
import { FeatureFlagEnvVar } from 'types/featureFlag';
import {
    BaseImageIntegration,
    ClairifyImageIntegration,
    GoogleImageIntegration,
    QuayImageIntegration,
} from 'types/imageIntegration.proto';
import {
    AuthProviderIntegration,
    AuthProviderType,
    BackupIntegrationType,
    CloudSourceIntegrationType,
    ImageIntegrationType,
    NotifierIntegrationType,
    SignatureIntegrationType,
} from 'types/integration';
import {
    BaseNotifierIntegration,
    SumoLogicNotifierIntegration,
    SyslogNotifierIntegration,
} from 'types/notifier.proto';
import { SignatureIntegration } from 'types/signatureIntegration.proto';

import { getOriginLabel } from 'Containers/AccessControl/traits';
import { AuthMachineToMachineConfig } from 'services/MachineAccessService';
import { CloudSourceIntegration } from 'services/CloudSourceService';
import {
    categoriesUtilsForClairifyScanner,
    categoriesUtilsForRegistryScanner,
    daysOfWeek,
    timesOfDay,
    transformDurationLongForm,
} from './integrationUtils';

const { getCategoriesText: getCategoriesTextForClairifyScanner } =
    categoriesUtilsForClairifyScanner;
const { getCategoriesText: getCategoriesTextForRegistryScanner } =
    categoriesUtilsForRegistryScanner;

export type AccessorFunction<Integration> = (integration: Integration) => string;

export type IntegrationTableColumnDescriptor<Integration> = {
    Header: string;
    accessor: string | AccessorFunction<Integration>;
    featureFlagDependency?: FeatureFlagEnvVar;
};

/*
 * To add a table column behind a feature flag:
 * 1. Add to string union type in types/featureFlag.ts file.
 * 2. Add the following property to the table column descriptor:
 *    featureFlagDependency: 'ROX_WHATEVER',
 */

type IntegrationTableColumnDescriptorMap = {
    authProviders: Record<
        AuthProviderType,
        IntegrationTableColumnDescriptor<AuthProviderIntegration>[]
    >;
    backups: Record<
        BackupIntegrationType,
        IntegrationTableColumnDescriptor<BaseBackupIntegration>[]
    >;
    imageIntegrations: Record<
        ImageIntegrationType,
        IntegrationTableColumnDescriptor<BaseImageIntegration>[]
    > & {
        clairify: IntegrationTableColumnDescriptor<ClairifyImageIntegration>[];
        google: IntegrationTableColumnDescriptor<GoogleImageIntegration>[];
        quay: IntegrationTableColumnDescriptor<QuayImageIntegration>[];
    };
    notifiers: Record<
        NotifierIntegrationType,
        IntegrationTableColumnDescriptor<BaseNotifierIntegration>[]
    > & {
        sumologic: IntegrationTableColumnDescriptor<SumoLogicNotifierIntegration>[];
        syslog: IntegrationTableColumnDescriptor<SyslogNotifierIntegration>[];
    };
    signatureIntegrations: Record<
        SignatureIntegrationType,
        IntegrationTableColumnDescriptor<SignatureIntegration>[]
    >;
    cloudSources: Record<
        CloudSourceIntegrationType,
        IntegrationTableColumnDescriptor<CloudSourceIntegration>[]
    >;
};

const originColumnDescriptor = {
    accessor: (integration) => {
        return getOriginLabel(integration.traits);
    },
    Header: 'Origin',
};

const tableColumnDescriptor: Readonly<IntegrationTableColumnDescriptorMap> = {
    authProviders: {
        clusterInitBundle: [{ accessor: 'name', Header: 'Name' }],
        apitoken: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'role', Header: 'Role' },
            { accessor: 'expiration', Header: 'Expiration' },
        ],
        machineAccess: [
            {
                accessor: (config) => {
                    const { type } = <AuthMachineToMachineConfig>config;
                    if (type === 'GENERIC') {
                        return 'Generic';
                    }
                    if (type === 'GITHUB_ACTIONS') {
                        return 'Github action';
                    }
                    return 'Unknown';
                },
                Header: 'Configuration',
            },
            { accessor: 'issuer', Header: 'Issuer' },
            {
                accessor: (config) => {
                    return transformDurationLongForm(
                        (<AuthMachineToMachineConfig>config).tokenExpirationDuration
                    );
                },
                Header: 'Token lifetime',
            },
        ],
    },
    notifiers: {
        awsSecurityHub: [
            { accessor: 'name', Header: 'Name' },
            {
                accessor: 'awsSecurityHub.accountId',
                Header: 'AWS Account Number',
            },
            { accessor: 'awsSecurityHub.region', Header: 'AWS Region' },
        ],
        slack: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'labelDefault', Header: 'Default Webhook' },
            { accessor: 'labelKey', Header: 'Webhook Annotation Key' },
        ],
        teams: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'labelDefault', Header: 'Default Webhook' },
            { accessor: 'labelKey', Header: 'Webhook Annotation Key' },
        ],
        jira: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'labelDefault', Header: 'Default Project' },
            { accessor: 'labelKey', Header: 'Project Annotation Key' },
            {
                accessor: 'jira.url',
                Header: 'URL',
            },
        ],
        email: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'labelDefault', Header: 'Default Recipient' },
            { accessor: 'labelKey', Header: 'Recipient Annotation Key' },
            { accessor: 'email.server', Header: 'Server' },
        ],
        cscc: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'cscc.sourceId', Header: 'Google Cloud SCC Source ID' },
        ],
        splunk: [
            { accessor: 'name', Header: 'Name' },
            originColumnDescriptor,
            {
                accessor: 'splunk.httpEndpoint',
                Header: 'URL',
            },
            { accessor: 'splunk.truncate', Header: 'HEC Truncate Limit' },
        ],
        pagerduty: [{ accessor: 'name', Header: 'Name' }],
        generic: [
            { accessor: 'name', Header: 'Name' },
            originColumnDescriptor,
            { accessor: 'generic.endpoint', Header: 'Endpoint' },
        ],
        sumologic: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'sumologic.httpSourceAddress', Header: 'HTTP Collector Source Address' },
            {
                Header: 'Skip TLS Certificate Verification',
                accessor: (integration) =>
                    integration.sumologic.skipTLSVerify ? 'Yes (Insecure)' : 'No (Secure)',
            },
        ],
        syslog: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'syslog.tcpConfig.hostname', Header: 'Receiver Host' },
            {
                Header: 'Skip TLS Certificate Verification',
                accessor: (integration) =>
                    integration.syslog.tcpConfig.skipTlsVerify ? 'Yes (Insecure)' : 'No (Secure)',
            },
        ],
    },
    imageIntegrations: {
        docker: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'docker.endpoint', Header: 'Endpoint' },
            { accessor: 'docker.username', Header: 'Username' },
            {
                Header: 'Autogenerated',
                accessor: (integration) => (integration.autogenerated ? 'True' : 'False'),
            },
        ],
        artifactregistry: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'google.endpoint', Header: 'Endpoint' },
            { accessor: 'google.project', Header: 'Project' },
        ],
        artifactory: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'docker.endpoint', Header: 'Endpoint' },
            { accessor: 'docker.username', Header: 'Username' },
        ],
        quay: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'quay.endpoint', Header: 'Endpoint' },
            {
                Header: 'Type',
                accessor: (integration) =>
                    getCategoriesTextForRegistryScanner(integration.categories),
            },
        ],
        clair: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'clair.endpoint', Header: 'Endpoint' },
        ],
        clairV4: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'clairV4.endpoint', Header: 'Endpoint' },
        ],
        clairify: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'clairify.endpoint', Header: 'Endpoint' },
            {
                Header: 'Type',
                accessor: (integration) =>
                    getCategoriesTextForClairifyScanner(integration.categories),
            },
        ],
        scannerv4: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'scannerV4.indexerEndpoint', Header: 'Indexer Endpoint' },
            { accessor: 'scannerV4.matcherEndpoint', Header: 'Matcher Endpoint' },
        ],
        google: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'google.endpoint', Header: 'Endpoint' },
            { accessor: 'google.project', Header: 'Project' },
            {
                Header: 'Type',
                accessor: (integration) =>
                    getCategoriesTextForRegistryScanner(integration.categories),
            },
        ],
        ecr: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'ecr.registryId', Header: '12-digit AWS ID' },
            { accessor: 'ecr.region', Header: 'Region' },
            {
                Header: 'Autogenerated',
                accessor: (integration) => (integration.autogenerated ? 'True' : 'False'),
            },
        ],
        nexus: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'docker.endpoint', Header: 'Endpoint' },
            { accessor: 'docker.username', Header: 'Username' },
        ],
        azure: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'docker.endpoint', Header: 'Endpoint' },
            { accessor: 'docker.username', Header: 'Username' },
        ],
        ibm: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'ibm.endpoint', Header: 'Endpoint' },
        ],
        rhel: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'docker.endpoint', Header: 'Endpoint' },
            { accessor: 'docker.username', Header: 'Username' },
        ],
    },
    signatureIntegrations: {
        signature: [
            { accessor: 'name', Header: 'Name' },
            {
                accessor: (integration) => (integration.cosign ? 'Cosign' : ''),
                Header: 'Verification methods',
            },
        ],
    },
    backups: {
        s3: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 's3.bucket', Header: 'Bucket' },
            {
                accessor: ({ schedule }) => {
                    if (schedule.intervalType === 'WEEKLY') {
                        return `Weekly on ${daysOfWeek[schedule.weekly.day]} @ ${
                            timesOfDay[schedule.hour]
                        } UTC`;
                    }
                    return `Daily @ ${timesOfDay[schedule.hour]} UTC`;
                },
                Header: 'Schedule',
            },
        ],
        gcs: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'gcs.bucket', Header: 'Bucket' },
            {
                accessor: ({ schedule }) => {
                    if (schedule.intervalType === 'WEEKLY') {
                        return `Weekly on ${daysOfWeek[schedule.weekly.day]} @ ${
                            timesOfDay[schedule.hour]
                        } UTC`;
                    }
                    return `Daily @ ${timesOfDay[schedule.hour]} UTC`;
                },
                Header: 'Schedule',
            },
        ],
    },
    cloudSources: {
        paladinCloud: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'paladinCloud.endpoint', Header: 'Endpoint' },
        ],
        ocm: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'ocm.endpoint', Header: 'Endpoint' },
        ],
    },
};

export default tableColumnDescriptor;
