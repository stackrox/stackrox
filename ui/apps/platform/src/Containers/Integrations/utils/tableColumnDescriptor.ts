import { BaseBackupIntegration } from 'types/externalBackup.proto';
import { FeatureFlagEnvVar } from 'types/featureFlag';
import {
    BaseImageIntegration,
    ClairifyImageIntegration,
    GoogleImageIntegration,
    QuayImageIntegration,
} from 'types/imageIntegration.proto';
import {
    AuthProviderType,
    BackupIntegrationType,
    BaseIntegration,
    ImageIntegrationType,
    NotifierIntegrationType,
    SignatureIntegrationType,
} from 'types/integration';
import { SumoLogicNotifierIntegration, SyslogNotifierIntegration } from 'types/notifier.proto';
import { SignatureIntegration } from 'types/signatureIntegration.proto';

import {
    categoriesUtilsForClairifyScanner,
    categoriesUtilsForRegistryScanner,
    daysOfWeek,
    timesOfDay,
} from './integrationUtils';

const { getCategoriesText: getCategoriesTextForClairifyScanner } =
    categoriesUtilsForClairifyScanner;
const { getCategoriesText: getCategoriesTextForRegistryScanner } =
    categoriesUtilsForRegistryScanner;

export type AccessorFunction = (integration: BaseIntegration) => string;

export type IntegrationTableColumnDescriptor = {
    Header: string;
    accessor: string | AccessorFunction;
    featureFlagDependency?: FeatureFlagEnvVar;
};

/*
 * To add a table column behind a feature flag:
 * 1. Add to string union type in types/featureFlag.ts file.
 * 2. Add the following property to the table column descriptor:
 *    featureFlagDependency: 'ROX_WHATEVER',
 */

type IntegrationTableColumnDescriptorMap = {
    authProviders: Record<AuthProviderType, IntegrationTableColumnDescriptor[]>;
    backups: Record<BackupIntegrationType, IntegrationTableColumnDescriptor[]>;
    imageIntegrations: Record<ImageIntegrationType, IntegrationTableColumnDescriptor[]>;
    notifiers: Record<NotifierIntegrationType, IntegrationTableColumnDescriptor[]>;
    signatureIntegrations: Record<SignatureIntegrationType, IntegrationTableColumnDescriptor[]>;
};

const tableColumnDescriptor: Readonly<IntegrationTableColumnDescriptorMap> = {
    authProviders: {
        clusterInitBundle: [{ accessor: 'name', Header: 'Name' }],
        apitoken: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'role', Header: 'Role' },
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
            {
                accessor: 'splunk.httpEndpoint',
                Header: 'URL',
            },
            { accessor: 'splunk.truncate', Header: 'HEC Truncate Limit' },
        ],
        pagerduty: [{ accessor: 'name', Header: 'Name' }],
        generic: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'generic.endpoint', Header: 'Endpoint' },
        ],
        sumologic: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'sumologic.httpSourceAddress', Header: 'HTTP Collector Source Address' },
            {
                Header: 'Skip TLS Certificate Verification',
                accessor: (integration) =>
                    (integration as SumoLogicNotifierIntegration).sumologic.skipTLSVerify
                        ? 'Yes (Insecure)'
                        : 'No (Secure)',
            },
        ],
        syslog: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'syslog.tcpConfig.hostname', Header: 'Receiver Host' },
            {
                Header: 'Skip TLS Certificate Verification',
                accessor: (integration) =>
                    (integration as SyslogNotifierIntegration).syslog.tcpConfig.skipTlsVerify
                        ? 'Yes (Insecure)'
                        : 'No (Secure)',
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
                accessor: (integration) =>
                    (integration as BaseImageIntegration).autogenerated ? 'True' : 'False',
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
                    getCategoriesTextForRegistryScanner(
                        (integration as QuayImageIntegration).categories
                    ),
            },
        ],
        clair: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'clair.endpoint', Header: 'Endpoint' },
        ],
        clairify: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'clairify.endpoint', Header: 'Endpoint' },
            {
                Header: 'Type',
                accessor: (integration) =>
                    getCategoriesTextForClairifyScanner(
                        (integration as ClairifyImageIntegration).categories
                    ),
            },
        ],
        google: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'google.endpoint', Header: 'Endpoint' },
            { accessor: 'google.project', Header: 'Project' },
            {
                Header: 'Type',
                accessor: (integration) =>
                    getCategoriesTextForRegistryScanner(
                        (integration as GoogleImageIntegration).categories
                    ),
            },
        ],
        ecr: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'ecr.registryId', Header: 'Registry ID' },
            { accessor: 'ecr.region', Header: 'Region' },
            {
                Header: 'Autogenerated',
                accessor: (integration) =>
                    (integration as BaseImageIntegration).autogenerated ? 'True' : 'False',
                featureFlagDependency: 'ROX_ECR_AUTO_INTEGRATION',
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
                accessor: (integration) => {
                    return (integration as SignatureIntegration).cosign ? 'Cosign' : '';
                },
                Header: 'Verification methods',
            },
        ],
    },
    backups: {
        s3: [
            { accessor: 'name', Header: 'Name' },
            { accessor: 's3.bucket', Header: 'Bucket' },
            {
                accessor: (integration) => {
                    const { schedule } = integration as BaseBackupIntegration;
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
                accessor: (integration) => {
                    const { schedule } = integration as BaseBackupIntegration;
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
};

export default tableColumnDescriptor;
