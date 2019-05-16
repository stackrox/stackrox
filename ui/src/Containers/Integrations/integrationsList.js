import anchore from 'images/anchore.svg';
import artifactory from 'images/artifactory.svg';
import aws from 'images/aws.svg';
import azure from 'images/azure.svg';
import clair from 'images/clair.svg';
import docker from 'images/docker.svg';
import email from 'images/email.svg';
import google from 'images/google-cloud.svg';
import jira from 'images/jira.svg';
import kubernetes from 'images/kubernetes.svg';
import logo from 'images/logo-tall.svg';
import nexus from 'images/nexus.svg';
import openshift from 'images/openshift.svg';
import quay from 'images/quay.svg';
import slack from 'images/slack.svg';
import splunk from 'images/splunk.svg';
import s3 from 'images/s3.svg';
import pagerduty from 'images/pagerduty.svg';
import tenable from 'images/tenable.svg';

const integrationsList = {
    authProviders: [
        {
            label: 'API Token',
            type: 'apitoken',
            source: 'authProviders',
            image: logo
        }
    ],
    imageIntegrations: [
        {
            label: 'StackRox Scanner',
            type: 'clairify',
            categories: 'Scanner',
            source: 'imageIntegrations',
            image: logo,
            disabled: false
        },
        {
            label: 'Generic Docker Registry',
            type: 'docker',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: docker,
            disabled: false
        },
        {
            label: 'Anchore Scanner',
            type: 'anchore',
            categories: 'Scanner',
            source: 'imageIntegrations',
            image: anchore,
            disabled: false
        },
        {
            label: 'AWS ECR',
            type: 'ecr',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: aws,
            disabled: false
        },
        {
            label: 'Google Cloud',
            type: 'google',
            categories: 'Registry + Scanner',
            source: 'imageIntegrations',
            image: google,
            disabled: false
        },
        {
            label: 'Microsoft ACR',
            type: 'azure',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: azure,
            disabled: false
        },
        {
            label: 'JFrog Artifactory',
            type: 'artifactory',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: artifactory,
            disabled: false
        },
        {
            label: 'Docker Trusted Registry',
            type: 'dtr',
            categories: 'Registry + Scanner',
            source: 'imageIntegrations',
            image: docker,
            disabled: false
        },
        {
            label: 'Quay.io',
            type: 'quay',
            categories: 'Registry + Scanner',
            source: 'imageIntegrations',
            image: quay,
            disabled: false
        },
        {
            label: 'CoreOS Clair',
            type: 'clair',
            categories: 'Scanner',
            source: 'imageIntegrations',
            image: clair,
            disabled: false
        },
        {
            label: 'Sonatype Nexus',
            type: 'nexus',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: nexus,
            disabled: false
        },
        {
            label: 'Tenable.io',
            type: 'tenable',
            categories: 'Registry + Scanner',
            source: 'imageIntegrations',
            image: tenable,
            disabled: false
        }
    ],
    orchestrators: [
        {
            label: 'Kubernetes',
            image: kubernetes,
            source: 'clusters',
            type: 'KUBERNETES_CLUSTER'
        },
        {
            label: 'OpenShift',
            image: openshift,
            source: 'clusters',
            type: 'OPENSHIFT_CLUSTER'
        }
    ],
    plugins: [
        {
            label: 'Slack',
            type: 'slack',
            source: 'notifiers',
            image: slack
        },
        {
            label: 'Generic Webhook',
            type: 'generic',
            source: 'notifiers',
            image: logo
        },
        {
            label: 'Jira',
            type: 'jira',
            source: 'notifiers',
            image: jira
        },
        {
            label: 'Email',
            type: 'email',
            source: 'notifiers',
            image: email
        },
        {
            label: 'Google Cloud SCC',
            type: 'cscc',
            source: 'notifiers',
            image: google
        },
        {
            label: 'Splunk',
            type: 'splunk',
            source: 'notifiers',
            image: splunk
        },
        {
            label: 'PagerDuty',
            type: 'pagerduty',
            source: 'notifiers',
            image: pagerduty
        }
    ],
    backups: [
        {
            label: 'S3',
            type: 's3',
            source: 'backups',
            image: s3
        }
    ]
};

export default integrationsList;
