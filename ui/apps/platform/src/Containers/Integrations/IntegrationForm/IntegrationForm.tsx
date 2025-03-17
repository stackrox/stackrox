import React, { FunctionComponent, ReactElement, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

import { isUserResource } from 'Containers/AccessControl/traits';
import useCentralCapabilities from 'hooks/useCentralCapabilities';
import { integrationsPath } from 'routePaths';
import { Integration, IntegrationSource, IntegrationType } from '../utils/integrationUtils';

// image integrations
import ArtifactRegistryIntegrationForm from './Forms/ArtifactRegistryIntegrationForm';
import ArtifactoryIntegrationForm from './Forms/ArtifactoryIntegrationForm';
import AzureIntegrationForm from './Forms/AzureIntegrationForm';
import ClairIntegrationForm from './Forms/ClairIntegrationForm';
import ClairV4IntegrationForm from './Forms/ClairV4IntegrationForm';
import ClairifyIntegrationForm from './Forms/ClairifyIntegrationForm';
import DockerIntegrationForm from './Forms/DockerIntegrationForm';
import EcrIntegrationForm from './Forms/EcrIntegrationForm';
import GhcrIntegrationForm from './Forms/GhcrIntegrationForm';
import GoogleIntegrationForm from './Forms/GoogleIntegrationForm';
import IbmIntegrationForm from './Forms/IbmIntegrationForm';
import NexusIntegrationForm from './Forms/NexusIntegrationForm';
import QuayIntegrationForm from './Forms/QuayIntegrationForm';
import RhelIntegrationForm from './Forms/RhelIntegrationForm';
import ScannerV4IntegrationForm from './Forms/ScannerV4IntegrationForm';
// notifiers
import ACSCSEmailIntegrationForm from './Forms/AcscsEmailIntegrationForm';
import AwsSecurityHubIntegrationForm from './Forms/AwsSecurityHubIntegrationForm';
import EmailIntegrationForm from './Forms/EmailIntegrationForm';
import GenericWebhookIntegrationForm from './Forms/GenericWebhookIntegrationForm';
import GoogleCloudSccIntegrationForm from './Forms/GoogleCloudSccIntegrationForm';
import JiraIntegrationForm from './Forms/JiraIntegrationForm';
import MicrosoftSentinelForm from './Forms/MicrosoftSentinelForm';
import PagerDutyIntegrationForm from './Forms/PagerDutyIntegrationForm';
import SlackIntegrationForm from './Forms/SlackIntegrationForm';
import SplunkIntegrationForm from './Forms/SplunkIntegrationForm';
import SumoLogicIntegrationForm from './Forms/SumoLogicIntegrationForm';
import SyslogIntegrationForm from './Forms/SyslogIntegrationForm';
import TeamsIntegrationForm from './Forms/TeamsIntegrationForm';
// external backups
import GcsIntegrationForm from './Forms/ExternalBackupIntegrations/GcsIntegrationForm';
import S3CompatibleIntegrationForm from './Forms/ExternalBackupIntegrations/S3CompatibleIntegrationForm';
import S3IntegrationForm from './Forms/ExternalBackupIntegrations/S3IntegrationForm';
// auth plugins
import ApiTokenIntegrationForm from './Forms/ApiTokenIntegrationForm';
import MachineAccessIntegrationForm from './Forms/MachineAccessIntegrationForm';
// signature integrations
import SignatureIntegrationForm from './Forms/SignatureIntegrationForm';

// cloud source integrations
import OcmIntegrationForm from './Forms/CloudSourceIntegrations/OcmIntegrationForm';
import PaladinCloudIntegrationForm from './Forms/CloudSourceIntegrations/PaladinCloudIntegrationForm';

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
    authProviders: {
        apitoken: ApiTokenIntegrationForm,
        machineAccess: MachineAccessIntegrationForm,
    },
    backups: {
        gcs: GcsIntegrationForm,
        s3: S3IntegrationForm,
        s3compatible: S3CompatibleIntegrationForm,
    },
    cloudSources: {
        ocm: OcmIntegrationForm,
        paladinCloud: PaladinCloudIntegrationForm,
    },
    imageIntegrations: {
        artifactory: ArtifactoryIntegrationForm,
        artifactregistry: ArtifactRegistryIntegrationForm,
        azure: AzureIntegrationForm,
        clair: ClairIntegrationForm,
        clairV4: ClairV4IntegrationForm,
        clairify: ClairifyIntegrationForm,
        docker: DockerIntegrationForm,
        ecr: EcrIntegrationForm,
        ghcr: GhcrIntegrationForm,
        google: GoogleIntegrationForm,
        ibm: IbmIntegrationForm,
        nexus: NexusIntegrationForm,
        quay: QuayIntegrationForm,
        rhel: RhelIntegrationForm,
        scannerv4: ScannerV4IntegrationForm,
    },
    notifiers: {
        acscsEmail: ACSCSEmailIntegrationForm,
        awsSecurityHub: AwsSecurityHubIntegrationForm,
        cscc: GoogleCloudSccIntegrationForm,
        email: EmailIntegrationForm,
        generic: GenericWebhookIntegrationForm,
        jira: JiraIntegrationForm,
        microsoftSentinel: MicrosoftSentinelForm,
        pagerduty: PagerDutyIntegrationForm,
        slack: SlackIntegrationForm,
        splunk: SplunkIntegrationForm,
        sumologic: SumoLogicIntegrationForm,
        syslog: SyslogIntegrationForm,
        teams: TeamsIntegrationForm,
    },
    signatureIntegrations: {
        signature: SignatureIntegrationForm,
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
    const navigate = useNavigate();

    const { isCentralCapabilityAvailable } = useCentralCapabilities();
    const canUseCloudBackupIntegrations = isCentralCapabilityAvailable(
        'centralCanUseCloudBackupIntegrations'
    );
    useEffect(() => {
        if (!canUseCloudBackupIntegrations && source === 'backups') {
            navigate(integrationsPath, { replace: true });
        }
    }, [canUseCloudBackupIntegrations, source, navigate]);

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
