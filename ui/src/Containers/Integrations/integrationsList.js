import artifactory from 'images/artifactory.svg';
import auth0 from 'images/auth0.svg';
import aws from 'images/aws.svg';
import clair from 'images/clair.svg';
import docker from 'images/docker.svg';
import email from 'images/email.svg';
import google from 'images/google-cloud.svg';
import jira from 'images/jira.svg';
import kubernetes from 'images/kubernetes.svg';
import logo from 'images/logo-tall.svg';
import openshift from 'images/openshift.svg';
import quay from 'images/quay.svg';
import slack from 'images/slack.svg';
import tenable from 'images/tenable.svg';

const integrationsList = {
    authProviders: [
        {
            label: 'Auth0',
            type: 'auth0',
            source: 'authProviders',
            image: auth0
        },
        {
            label: 'API Token',
            type: 'apitoken',
            source: 'authProviders',
            image: logo
        }
    ],
    dnrIntegrations: [
        {
            label: 'StackRox Detect & Respond',
            type: 'D&R',
            source: 'dnrIntegrations',
            image: logo
        }
    ],
    imageIntegrations: [
        {
            label: 'Generic Docker Registry',
            type: 'docker',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: docker,
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
            label: 'Tenable.io',
            type: 'tenable',
            categories: 'Registry + Scanner',
            source: 'imageIntegrations',
            image: tenable,
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
            label: 'Clairify',
            type: 'clairify',
            categories: 'Scanner',
            source: 'imageIntegrations',
            image: clair,
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
            label: 'AWS ECR',
            type: 'ecr',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: aws,
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
        },
        {
            label: 'Docker Swarm',
            image: docker,
            source: 'clusters',
            type: 'SWARM_CLUSTER'
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
        }
    ]
};

export default integrationsList;
