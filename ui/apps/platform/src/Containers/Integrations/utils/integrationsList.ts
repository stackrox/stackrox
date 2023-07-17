import artifactory from 'images/artifactory.svg';
import aws from 'images/aws.svg';
import awsSecurityHub from 'images/aws-security-hub.svg';
import azure from 'images/azure.svg';
import clair from 'images/clair.svg';
import docker from 'images/docker.svg';
import email from 'images/email.svg';
import google from 'images/google-cloud.svg';
import googleregistry from 'images/google-container.svg';
import googleartifact from 'images/google-artifact.svg';
import ibm from 'images/ibm-ccr.svg';
import jira from 'images/jira.svg';
import logo from 'images/StackRox-integration-logo.svg';
import nexus from 'images/nexus.svg';
import quay from 'images/quay.svg';
import redhat from 'images/redhat.svg';
import slack from 'images/slack.svg';
import splunk from 'images/splunk.svg';
import sumologic from 'images/sumologic.svg';
import s3 from 'images/s3.svg';
import syslog from 'images/syslog.svg';
import teams from 'images/teams.svg';
import pagerduty from 'images/pagerduty.svg';
import signature from 'images/signature.svg';

/*
 * To add an integration tile behind a feature flag:
 * 1. Add to string union type in types/featureFlag.ts file.
 * 2. Add the following property to the integration descriptor:
 *    featureFlagDependency: 'ROX_WHATEVER',
 */

import { FeatureFlagEnvVar } from 'types/featureFlag';
import {
    AuthProviderType,
    BackupIntegrationType,
    ImageIntegrationType,
    IntegrationSource,
    NotifierIntegrationType,
    SignatureIntegrationType,
} from 'types/integration';

export type IntegrationDescriptor =
    | AuthProviderDescriptor
    | BackupIntegrationDescriptor
    | ImageIntegrationDescriptor
    | NotifierIntegrationDescriptor
    | SignatureIntegrationDescriptor;

export type AuthProviderDescriptor = {
    source: 'authProviders';
    type: AuthProviderType;
} & BaseIntegrationDescriptor;

export type BackupIntegrationDescriptor = {
    source: 'backups';
    type: BackupIntegrationType;
} & BaseIntegrationDescriptor;

export type ImageIntegrationDescriptor = {
    source: 'imageIntegrations';
    type: ImageIntegrationType;
    categories: ImageIntegrationCategories;
} & BaseIntegrationDescriptor;

export type ImageIntegrationCategories =
    | 'Image Scanner + Node Scanner'
    | 'Registry'
    | 'Registry + Scanner'
    | 'Scanner';

export type NotifierIntegrationDescriptor = {
    source: 'notifiers';
    type: NotifierIntegrationType;
} & BaseIntegrationDescriptor;

export type SignatureIntegrationDescriptor = {
    source: 'signatureIntegrations';
    type: SignatureIntegrationType;
} & BaseIntegrationDescriptor;

export type BaseIntegrationDescriptor = {
    source: IntegrationSource;
    type: string;
    label: string;
    image: string;
    featureFlagDependency?: FeatureFlagEnvVar;
};

type IntegrationDescriptorMap = {
    authProviders: AuthProviderDescriptor[];
    backups: BackupIntegrationDescriptor[];
    imageIntegrations: ImageIntegrationDescriptor[];
    notifiers: NotifierIntegrationDescriptor[];
    signatureIntegrations: SignatureIntegrationDescriptor[];
};

const integrationsList: IntegrationDescriptorMap = {
    authProviders: [
        {
            label: 'API Token',
            type: 'apitoken',
            source: 'authProviders',
            image: logo,
        },
        {
            label: 'Cluster Init Bundle',
            type: 'clusterInitBundle',
            source: 'authProviders',
            image: logo,
        },
    ],
    imageIntegrations: [
        {
            label: 'StackRox Scanner',
            type: 'clairify',
            categories: 'Image Scanner + Node Scanner',
            source: 'imageIntegrations',
            image: logo,
        },
        {
            label: 'Generic Docker Registry',
            type: 'docker',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: docker,
        },
        {
            label: 'Amazon ECR',
            type: 'ecr',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: aws,
        },
        {
            label: 'Google Container Registry',
            type: 'google',
            categories: 'Registry + Scanner',
            source: 'imageIntegrations',
            image: googleregistry,
        },
        {
            label: 'Google Artifact Registry',
            type: 'artifactregistry',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: googleartifact,
        },
        {
            label: 'Microsoft ACR',
            type: 'azure',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: azure,
        },
        {
            label: 'JFrog Artifactory',
            type: 'artifactory',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: artifactory,
        },
        {
            label: 'Quay.io',
            type: 'quay',
            categories: 'Registry + Scanner',
            source: 'imageIntegrations',
            image: quay,
        },
        {
            label: '[DEPRECATED] CoreOS Clair',
            type: 'clair',
            categories: 'Scanner',
            source: 'imageIntegrations',
            image: clair,
        },
        {
            label: 'Clair v4',
            type: 'clairV4',
            categories: 'Scanner',
            source: 'imageIntegrations',
            image: clair,
        },
        {
            label: 'Sonatype Nexus',
            type: 'nexus',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: nexus,
        },
        {
            label: 'IBM Cloud',
            type: 'ibm',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: ibm,
        },
        {
            label: 'Red Hat',
            type: 'rhel',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: redhat,
        },
    ],
    signatureIntegrations: [
        {
            label: 'Signature',
            type: 'signature',
            source: 'signatureIntegrations',
            image: signature,
        },
    ],
    notifiers: [
        {
            label: 'Slack',
            type: 'slack',
            source: 'notifiers',
            image: slack,
        },
        {
            label: 'Generic Webhook',
            type: 'generic',
            source: 'notifiers',
            image: logo,
        },
        {
            label: 'Jira',
            type: 'jira',
            source: 'notifiers',
            image: jira,
        },
        {
            label: 'Email',
            type: 'email',
            source: 'notifiers',
            image: email,
        },
        {
            label: 'Google Cloud SCC',
            type: 'cscc',
            source: 'notifiers',
            image: google,
        },
        {
            label: 'Splunk',
            type: 'splunk',
            source: 'notifiers',
            image: splunk,
        },
        {
            label: 'PagerDuty',
            type: 'pagerduty',
            source: 'notifiers',
            image: pagerduty,
        },
        {
            label: 'Sumo Logic',
            type: 'sumologic',
            source: 'notifiers',
            image: sumologic,
        },
        {
            label: 'Microsoft Teams',
            type: 'teams',
            source: 'notifiers',
            image: teams,
        },
        {
            label: 'AWS Security Hub',
            type: 'awsSecurityHub',
            source: 'notifiers',
            image: awsSecurityHub,
        },
        {
            label: 'Syslog',
            type: 'syslog',
            source: 'notifiers',
            image: syslog,
        },
    ],
    backups: [
        {
            label: 'Amazon S3',
            type: 's3',
            source: 'backups',
            image: s3,
        },
        {
            label: 'Google Cloud Storage',
            type: 'gcs',
            source: 'backups',
            image: google,
        },
    ],
};

export default integrationsList;
