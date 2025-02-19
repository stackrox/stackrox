import acscsEmail from 'images/acsEmailNotifier.svg';
import artifactory from 'images/artifactory.svg';
import aws from 'images/aws.svg';
import awsSecurityHub from 'images/aws-security-hub.svg';
import azure from 'images/azure.svg';
import clair from 'images/clair.svg';
import docker from 'images/docker.svg';
import email from 'images/email.svg';
import ghcr from 'images/ghcr.svg';
import google from 'images/google-cloud.svg';
import googleartifact from 'images/google-artifact.svg';
import googleregistry from 'images/google-container.svg';
import ibm from 'images/ibm-ccr.svg';
import jira from 'images/jira.svg';
import logo from 'images/StackRox-integration-logo.svg';
import microsoftSentinel from 'images/microsoft_sentinel.svg';
import nexus from 'images/nexus.svg';
import pagerduty from 'images/pagerduty.svg';
import quay from 'images/quay.svg';
import redhat from 'images/redhat.svg';
import signature from 'images/signature.svg';
import slack from 'images/slack.svg';
import splunk from 'images/splunk.svg';
import sumologic from 'images/sumologic.svg';
import s3 from 'images/s3.svg';
import s3compatible from 'images/s3-compatible.svg';
import syslog from 'images/syslog.svg';
import teams from 'images/teams.svg';
import paladinCloud from 'images/paladinCloud.svg';
import { integrationsPath } from 'routePaths';

/*
 * To add an integration tile behind a feature flag:
 * 1. Add to string union type in types/featureFlag.ts file.
 * 2. Add the following property to the integration descriptor:
 *    featureFlagDependency: ['ROX_WHATEVER_1', 'ROX_WHATEVER_2'],
 */

import { FeatureFlagEnvVar } from 'types/featureFlag';
import {
    AuthProviderType,
    BackupIntegrationType,
    CloudSourceIntegrationType,
    ImageIntegrationType,
    IntegrationSource,
    NotifierIntegrationType,
    SignatureIntegrationType,
} from 'types/integration';

export type AuthProviderDescriptor = {
    type: AuthProviderType;
} & BaseIntegrationDescriptor;

export type BackupIntegrationDescriptor = {
    type: BackupIntegrationType;
} & BaseIntegrationDescriptor;

export type ImageIntegrationDescriptor = {
    type: ImageIntegrationType;
    categories: ImageIntegrationCategories;
} & BaseIntegrationDescriptor;

export type ImageIntegrationCategories =
    | 'Image Scanner + Node Scanner'
    | 'Registry'
    | 'Registry + Scanner'
    | 'Scanner';

export type NotifierIntegrationDescriptor = {
    type: NotifierIntegrationType;
} & BaseIntegrationDescriptor;

export type SignatureIntegrationDescriptor = {
    type: SignatureIntegrationType;
} & BaseIntegrationDescriptor;

export type CloudSourceDescriptor = {
    type: CloudSourceIntegrationType;
} & BaseIntegrationDescriptor;

export type BaseIntegrationDescriptor = {
    type: string;
    label: string;
    image: string;
    featureFlagDependency?: FeatureFlagEnvVar[];
};

export const imageIntegrationsSource: IntegrationSource = 'imageIntegrations';

export const imageIntegrationsDescriptors: ImageIntegrationDescriptor[] = [
    {
        categories: 'Image Scanner + Node Scanner',
        image: logo,
        label: '[DEPRECATED] StackRox Scanner',
        type: 'clairify',
    },
    {
        categories: 'Image Scanner + Node Scanner',
        image: logo,
        label: 'Scanner V4',
        type: 'scannerv4',
        featureFlagDependency: ['ROX_SCANNER_V4'],
    },
    {
        categories: 'Registry',
        image: docker,
        label: 'Generic Docker Registry',
        type: 'docker',
    },
    {
        categories: 'Registry',
        image: aws,
        label: 'Amazon ECR',
        type: 'ecr',
    },
    {
        categories: 'Registry + Scanner',
        image: googleregistry,
        label: '[DEPRECATED] Google Container Registry',
        type: 'google',
    },
    {
        categories: 'Registry',
        image: googleartifact,
        label: 'Google Artifact Registry',
        type: 'artifactregistry',
    },
    {
        categories: 'Registry',
        image: azure,
        label: 'Microsoft ACR',
        type: 'azure',
    },
    {
        categories: 'Registry',
        image: artifactory,
        label: 'JFrog Artifactory',
        type: 'artifactory',
    },
    {
        categories: 'Registry + Scanner',
        image: quay,
        label: 'Quay.io',
        type: 'quay',
    },
    {
        categories: 'Scanner',
        image: clair,
        label: '[DEPRECATED] CoreOS Clair',
        type: 'clair',
    },
    {
        categories: 'Scanner',
        image: clair,
        label: 'Clair v4',
        type: 'clairV4',
    },
    {
        categories: 'Registry',
        image: nexus,
        label: 'Sonatype Nexus',
        type: 'nexus',
    },
    {
        categories: 'Registry',
        image: ibm,
        label: 'IBM Cloud',
        type: 'ibm',
    },
    {
        categories: 'Registry',
        image: redhat,
        label: 'Red Hat',
        type: 'rhel',
    },
    {
        categories: 'Registry',
        image: ghcr,
        label: 'GitHub Container Registry',
        type: 'ghcr',
    },
];

export const signatureIntegrationsSource = 'signatureIntegrations';

export const signatureIntegrationDescriptor: SignatureIntegrationDescriptor = {
    image: signature,
    label: 'Signature',
    type: 'signature',
};

const signatureIntegrationsDescriptors = [signatureIntegrationDescriptor];

export const notifierIntegrationsSource = 'notifiers';

export const notifierIntegrationsDescriptors: NotifierIntegrationDescriptor[] = [
    {
        image: slack,
        label: 'Slack',
        type: 'slack',
    },
    {
        image: logo,
        label: 'Generic Webhook',
        type: 'generic',
    },
    {
        image: jira,
        label: 'Jira',
        type: 'jira',
    },
    {
        image: email,
        label: 'Email',
        type: 'email',
    },
    {
        image: acscsEmail,
        label: 'RHACS Cloud Service',
        type: 'acscsEmail',
    },
    {
        image: google,
        label: 'Google Cloud SCC',
        type: 'cscc',
    },
    {
        image: splunk,
        label: 'Splunk',
        type: 'splunk',
    },
    {
        image: pagerduty,
        label: 'PagerDuty',
        type: 'pagerduty',
    },
    {
        image: sumologic,
        label: 'Sumo Logic',
        type: 'sumologic',
    },
    {
        image: teams,
        label: 'Microsoft Teams',
        type: 'teams',
    },
    {
        image: awsSecurityHub,
        label: 'AWS Security Hub',
        type: 'awsSecurityHub',
    },
    {
        image: syslog,
        label: 'Syslog',
        type: 'syslog',
    },
    {
        image: microsoftSentinel,
        label: 'Microsoft Sentinel',
        type: 'microsoftSentinel',
    },
];

export const backupIntegrationsSource = 'backups';

export const backupIntegrationsDescriptors: BackupIntegrationDescriptor[] = [
    {
        image: s3,
        label: 'Amazon S3',
        type: 's3',
    },
    {
        image: s3compatible,
        label: 'S3 API Compatible',
        type: 's3compatible',
    },
    {
        image: google,
        label: 'Google Cloud Storage',
        type: 'gcs',
    },
];

export const authenticationTokensSource = 'authProviders';

export const apiTokenDescriptor: AuthProviderDescriptor = {
    image: logo,
    label: 'API Token',
    type: 'apitoken',
};

export const clusterInitBundleDescriptor: AuthProviderDescriptor = {
    image: logo,
    label: 'Cluster Init Bundle',
    type: 'clusterInitBundle',
};

export const machineAccessDescriptor: AuthProviderDescriptor = {
    image: logo,
    label: 'Machine access configuration',
    type: 'machineAccess',
};

const authenticationTokensDescriptors = [
    apiTokenDescriptor,
    clusterInitBundleDescriptor,
    machineAccessDescriptor,
];

export const cloudSourcesSource = 'cloudSources';

export const paladinCloudDescriptor: CloudSourceDescriptor = {
    image: paladinCloud,
    label: 'Paladin Cloud',
    type: 'paladinCloud',
};

export const ocmDescriptor: CloudSourceDescriptor = {
    image: redhat,
    label: 'OpenShift Cluster Manager',
    type: 'ocm',
};

const cloudSourceDescriptors = [paladinCloudDescriptor, ocmDescriptor];

function getDescriptors(source: string): BaseIntegrationDescriptor[] {
    switch (source) {
        case 'imageIntegrations':
            return imageIntegrationsDescriptors;
        case 'signatureIntegrations':
            return signatureIntegrationsDescriptors;
        case 'notifiers':
            return notifierIntegrationsDescriptors;
        case 'backups':
            return backupIntegrationsDescriptors;
        case 'authProviders':
            return authenticationTokensDescriptors;
        case 'cloudSources':
            return cloudSourceDescriptors;
        default:
            return [];
    }
}

export function getIntegrationLabel(source: string, type: string): string {
    const descriptorFound = getDescriptors(source).find((descriptor) => descriptor.type === type);
    return descriptorFound ? descriptorFound.label : '';
}

export function getIntegrationsListPath(source: IntegrationSource, type: string) {
    return `${integrationsPath}/${source}/${type}`;
}
