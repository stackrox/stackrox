import React, { FunctionComponent, ReactElement } from 'react';

import { Integration, IntegrationSource, IntegrationType } from '../utils/integrationUtils';
// image integrations
import ClairifyIntegrationForm from './Forms/ClairifyIntegrationForm';
import ClairIntegrationForm from './Forms/ClairIntegrationForm';
import DockerIntegrationForm from './Forms/DockerIntegrationForm';
import AnchoreIntegrationForm from './Forms/AnchoreIntegrationForm';
import EcrIntegrationForm from './Forms/EcrIntegrationForm';
import GoogleIntegrationForm from './Forms/GoogleIntegrationForm';
import ArtifactRegistryIntegrationForm from './Forms/ArtifactRegistryIntegrationForm';
import AzureIntegrationForm from './Forms/AzureIntegrationForm';
import ArtifactoryIntegrationForm from './Forms/ArtifactoryIntegrationForm';
import DtrIntegrationForm from './Forms/DtrIntegrationForm';
import QuayIntegrationForm from './Forms/QuayIntegrationForm';
import NexusIntegrationForm from './Forms/NexusIntegrationForm';
import TenableIntegrationForm from './Forms/TenableIntegrationForm';
import IbmIntegrationForm from './Forms/IbmIntegrationForm';
import RhelIntegrationForm from './Forms/RhelIntegrationForm';
// notifiers
import AwsSecurityHubIntegrationForm from './Forms/AwsSecurityHubIntegrationForm';
import EmailIntegrationForm from './Forms/EmailIntegrationForm';
import GenericWebhookIntegrationForm from './Forms/GenericWebhookIntegrationForm';
import GoogleCloudSccIntegrationForm from './Forms/GoogleCloudSccIntegrationForm';
import JiraIntegrationForm from './Forms/JiraIntegrationForm';
import PagerDutyIntegrationForm from './Forms/PagerDutyIntegrationForm';
import SlackIntegrationForm from './Forms/SlackIntegrationForm';
import SplunkIntegrationForm from './Forms/SplunkIntegrationForm';
import SumoLogicIntegrationForm from './Forms/SumoLogicIntegrationForm';
import SyslogIntegrationForm from './Forms/SyslogIntegrationForm';
import TeamsIntegrationForm from './Forms/TeamsIntegrationForm';
// external backups
import S3IntegrationForm from './Forms/S3IntegrationForm';
import GcsIntegrationForm from './Forms/GcsIntegrationForm';
// auth plugins
import ApiTokenIntegrationForm from './Forms/ApiTokenIntegrationForm';
import ClusterInitBundleIntegrationForm from './Forms/ClusterInitBundleIntegrationForm';
// authorization plugins
import ScopedAccessPluginIntegrationForm from './Forms/ScopedAccessPluginIntegrationForm';
// signature integrations
import SignatureIntegrationForm from './Forms/SignatureIntegrationForm';

import './IntegrationForm.css';

type IntegrationFormProps = {
    source: IntegrationSource;
    type: IntegrationType;
    initialValues?: Integration | null;
    isEditable?: boolean;
};

type FormProps = {
    initialValues?: Integration | null;
    isEditable?: boolean;
};

const ComponentFormMap = {
    imageIntegrations: {
        clairify: ClairifyIntegrationForm,
        clair: ClairIntegrationForm,
        docker: DockerIntegrationForm,
        anchore: AnchoreIntegrationForm,
        ecr: EcrIntegrationForm,
        google: GoogleIntegrationForm,
        artifactregistry: ArtifactRegistryIntegrationForm,
        azure: AzureIntegrationForm,
        artifactory: ArtifactoryIntegrationForm,
        dtr: DtrIntegrationForm,
        quay: QuayIntegrationForm,
        nexus: NexusIntegrationForm,
        tenable: TenableIntegrationForm,
        ibm: IbmIntegrationForm,
        rhel: RhelIntegrationForm,
    },
    signatureIntegrations: {
        signature: SignatureIntegrationForm,
    },
    notifiers: {
        awsSecurityHub: AwsSecurityHubIntegrationForm,
        cscc: GoogleCloudSccIntegrationForm,
        email: EmailIntegrationForm,
        generic: GenericWebhookIntegrationForm,
        jira: JiraIntegrationForm,
        pagerduty: PagerDutyIntegrationForm,
        slack: SlackIntegrationForm,
        splunk: SplunkIntegrationForm,
        sumologic: SumoLogicIntegrationForm,
        syslog: SyslogIntegrationForm,
        teams: TeamsIntegrationForm,
    },
    backups: {
        s3: S3IntegrationForm,
        gcs: GcsIntegrationForm,
    },
    authProviders: {
        apitoken: ApiTokenIntegrationForm,
        clusterInitBundle: ClusterInitBundleIntegrationForm,
    },
    authPlugins: {
        scopedAccess: ScopedAccessPluginIntegrationForm,
    },
} as Record<IntegrationSource, Record<IntegrationType, FunctionComponent<FormProps>>>;

function IntegrationForm({
    source,
    type,
    initialValues,
    isEditable,
}: IntegrationFormProps): ReactElement {
    const Form: FunctionComponent<FormProps> = ComponentFormMap?.[source]?.[type];
    if (!Form) {
        throw new Error(
            `There are no integration form components for source (${source}) and type (${type})`
        );
    }
    return <Form initialValues={initialValues} isEditable={isEditable} />;
}

export default IntegrationForm;
