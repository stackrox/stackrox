import anchore from 'images/anchore.svg';
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
import tenable from 'images/tenable.svg';
import signature from 'images/signature.svg';

// Adding an integration tile behind a feature flag
// To add a new integration, uncomment the following import

// import { knownBackendFlags } from 'utils/featureFlags';

// and then add the following property to the new tiles object definition in the list below:
//     featureFlagDependency: knownBackendFlags.ROX_<flag_constant>,

const integrationsList = {
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
            disabled: false,
        },
        {
            label: 'Generic Docker Registry',
            type: 'docker',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: docker,
            disabled: false,
        },
        {
            label: 'Anchore Scanner',
            type: 'anchore',
            categories: 'Scanner',
            source: 'imageIntegrations',
            image: anchore,
            disabled: false,
        },
        {
            label: 'Amazon ECR',
            type: 'ecr',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: aws,
            disabled: false,
        },
        {
            label: 'Google Container Registry',
            type: 'google',
            categories: 'Registry + Scanner',
            source: 'imageIntegrations',
            image: googleregistry,
            disabled: false,
        },
        {
            label: 'Google Artifact Registry',
            type: 'artifactregistry',
            typeLabel: 'artifactregistry',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: googleartifact,
            disabled: false,
        },
        {
            label: 'Microsoft ACR',
            type: 'azure',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: azure,
            disabled: false,
        },
        {
            label: 'JFrog Artifactory',
            type: 'artifactory',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: artifactory,
            disabled: false,
        },
        {
            label: 'Docker Trusted Registry',
            type: 'dtr',
            categories: 'Registry + Scanner',
            source: 'imageIntegrations',
            image: docker,
            disabled: false,
        },
        {
            label: 'Quay.io',
            type: 'quay',
            categories: 'Registry + Scanner',
            source: 'imageIntegrations',
            image: quay,
            disabled: false,
        },
        {
            label: 'CoreOS Clair',
            type: 'clair',
            categories: 'Scanner',
            source: 'imageIntegrations',
            image: clair,
            disabled: false,
        },
        {
            label: 'Sonatype Nexus',
            type: 'nexus',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: nexus,
            disabled: false,
        },
        {
            label: 'Tenable.io',
            type: 'tenable',
            categories: 'Registry + Scanner',
            source: 'imageIntegrations',
            image: tenable,
            disabled: false,
        },
        {
            label: 'IBM Cloud',
            type: 'ibm',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: ibm,
            disabled: false,
        },
        {
            label: 'Red Hat',
            type: 'rhel',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: redhat,
            disabled: false,
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
    authPlugins: [
        {
            label: 'Scoped Access Plugin',
            type: 'scopedAccess',
            source: 'authPlugins',
            image: logo,
        },
    ],
};

export default integrationsList;
