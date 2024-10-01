import React, { FunctionComponent, ReactElement } from 'react';
import { useHistory } from 'react-router-dom';

import { isUserResource } from 'Containers/AccessControl/traits';
import useCentralCapabilities from 'hooks/useCentralCapabilities';
import { integrationsPath } from 'routePaths';
import { Integration, IntegrationSource, IntegrationType } from '../utils/integrationUtils';

// image integrations
import ClairifyIntegrationForm from './Forms/ClairifyIntegrationForm';
import ClairIntegrationForm from './Forms/ClairIntegrationForm';
import ClairV4IntegrationForm from './Forms/ClairV4IntegrationForm';
import DockerIntegrationForm from './Forms/DockerIntegrationForm';
import EcrIntegrationForm from './Forms/EcrIntegrationForm';
import GoogleIntegrationForm from './Forms/GoogleIntegrationForm';
import ArtifactRegistryIntegrationForm from './Forms/ArtifactRegistryIntegrationForm';
import AzureIntegrationForm from './Forms/AzureIntegrationForm';
import ArtifactoryIntegrationForm from './Forms/ArtifactoryIntegrationForm';
import QuayIntegrationForm from './Forms/QuayIntegrationForm';
import NexusIntegrationForm from './Forms/NexusIntegrationForm';
import IbmIntegrationForm from './Forms/IbmIntegrationForm';
import RhelIntegrationForm from './Forms/RhelIntegrationForm';
import ScannerV4IntegrationForm from './Forms/ScannerV4IntegrationForm';
// notifiers
import AwsSecurityHubIntegrationForm from './Forms/AwsSecurityHubIntegrationForm';
import EmailIntegrationForm from './Forms/EmailIntegrationForm';
import ACSCSEmailIntegrationForm from './Forms/AcscsEmailIntegrationForm';
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
import S3IntegrationForm from './Forms/ExternalBackupIntegrations/S3IntegrationForm';
import S3CompatibleIntegrationForm from './Forms/ExternalBackupIntegrations/S3CompatibleIntegrationForm';
import GcsIntegrationForm from './Forms/ExternalBackupIntegrations/GcsIntegrationForm';
// auth plugins
import ApiTokenIntegrationForm from './Forms/ApiTokenIntegrationForm';
import MachineAccessIntegrationForm from './Forms/MachineAccessIntegrationForm';
// signature integrations
import SignatureIntegrationForm from './Forms/SignatureIntegrationForm';

// cloud source integrations
import PaladinCloudIntegrationForm from './Forms/CloudSourceIntegrations/PaladinCloudIntegrationForm';
import OcmIntegrationForm from './Forms/CloudSourceIntegrations/OcmIntegrationForm';

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
        clairV4: ClairV4IntegrationForm,
        docker: DockerIntegrationForm,
        ecr: EcrIntegrationForm,
        google: GoogleIntegrationForm,
        artifactregistry: ArtifactRegistryIntegrationForm,
        azure: AzureIntegrationForm,
        artifactory: ArtifactoryIntegrationForm,
        quay: QuayIntegrationForm,
        nexus: NexusIntegrationForm,
        ibm: IbmIntegrationForm,
        rhel: RhelIntegrationForm,
        scannerv4: ScannerV4IntegrationForm,
    },
    signatureIntegrations: {
        signature: SignatureIntegrationForm,
    },
    notifiers: {
        awsSecurityHub: AwsSecurityHubIntegrationForm,
        cscc: GoogleCloudSccIntegrationForm,
        email: EmailIntegrationForm,
        acscsEmail: ACSCSEmailIntegrationForm,
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
        s3compatible: S3CompatibleIntegrationForm,
        gcs: GcsIntegrationForm,
    },
    authProviders: {
        apitoken: ApiTokenIntegrationForm,
        machineAccess: MachineAccessIntegrationForm,
    },
    cloudSources: {
        paladinCloud: PaladinCloudIntegrationForm,
        ocm: OcmIntegrationForm,
    },
} as Record<
    IntegrationSource,
    Record<IntegrationType, FunctionComponent<React.PropsWithChildren<FormProps>>>
>;

function IntegrationForm({
    source,
    type,
    initialValues,
    isEditable,
}: IntegrationFormProps): ReactElement {
    const history = useHistory();

    const { isCentralCapabilityAvailable } = useCentralCapabilities();
    const canUseCloudBackupIntegrations = isCentralCapabilityAvailable(
        'centralCanUseCloudBackupIntegrations'
    );
    if (!canUseCloudBackupIntegrations && source === 'backups') {
        history.replace(integrationsPath);
    }

    const Form: FunctionComponent<React.PropsWithChildren<FormProps>> =
        ComponentFormMap?.[source]?.[type];
    if (!Form) {
        throw new Error(
            `There are no integration form components for source (${source}) and type (${type})`
        );
    }
    return (
        <Form
            initialValues={initialValues}
            isEditable={isEditable && isUserResource(initialValues?.traits)}
        />
    );
}

export default IntegrationForm;
