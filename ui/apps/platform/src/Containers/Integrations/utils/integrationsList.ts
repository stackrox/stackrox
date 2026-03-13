import type { ComponentType, SVGProps } from 'react';
import AcscsEmailSvg from 'images/acsEmailNotifier.svg?react';
import ArtifactorySvg from 'images/artifactory.svg?react';
import AwsSvg from 'images/aws.svg?react';
import AwsSecurityHubSvg from 'images/aws-security-hub.svg?react';
import AzureSvg from 'images/azure.svg?react';
import ClairSvg from 'images/clair.svg?react';
import DockerSvg from 'images/docker.svg?react';
import EmailSvg from 'images/email.svg?react';
import GhcrSvg from 'images/ghcr.svg?react';
import GoogleSvg from 'images/google-cloud.svg?react';
import GoogleArtifactSvg from 'images/google-artifact.svg?react';
import GoogleRegistrySvg from 'images/google-container.svg?react';
import IbmSvg from 'images/ibm-ccr.svg?react';
import JiraSvg from 'images/jira.svg?react';
import LogoSvg from 'images/StackRox-integration-logo.svg?react';
import MicrosoftSentinelSvg from 'images/microsoft_sentinel.svg?react';
import NexusSvg from 'images/nexus.svg?react';
import PagerDutySvg from 'images/pagerduty.svg?react';
import QuaySvg from 'images/quay.svg?react';
import RedhatSvg from 'images/redhat.svg?react';
import SignatureSvg from 'images/signature.svg?react';
import SlackSvg from 'images/slack.svg?react';
import SplunkSvg from 'images/splunk.svg?react';
import SumologicSvg from 'images/sumologic.svg?react';
import S3Svg from 'images/s3.svg?react';
import S3CompatibleSvg from 'images/s3-compatible.svg?react';
import SyslogSvg from 'images/syslog.svg?react';
import TeamsSvg from 'images/teams.svg?react';
import PaladinCloudSvg from 'images/paladinCloud.svg?react';
import { integrationsPath } from 'routePaths';

/*
 * To add an integration tile behind a feature flag:
 * 1. Add to string union type in types/featureFlag.ts file.
 * 2. Add the following property to the integration descriptor:
 *    featureFlagDependency: ['ROX_WHATEVER_1', 'ROX_WHATEVER_2'],
 */

import type { IsCentralCapabilityAvailable } from 'hooks/useCentralCapabilities';
import type { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import type { CentralCapabilitiesFlags } from 'services/MetadataService';
import type { FeatureFlagEnvVar } from 'types/featureFlag';
import { integrationSources } from 'types/integration';
import type {
    AuthProviderType,
    BackupIntegrationType,
    CloudSourceIntegrationType,
    ImageIntegrationType,
    IntegrationSource,
    IntegrationType,
    NotifierIntegrationType,
    SignatureIntegrationType,
} from 'types/integration';
// import { allEnabled } from 'utils/featureFlagUtils'; // uncomment when needed
import type { FeatureFlagPredicate } from 'utils/featureFlagUtils';

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
    Logo: ComponentType<SVGProps<SVGSVGElement>>;
    centralCapabilityRequirement?: CentralCapabilitiesFlags;
    featureFlagDependency?: FeatureFlagEnvVar[];
};

export const imageIntegrationsSource: IntegrationSource = 'imageIntegrations';

export const imageIntegrationsDescriptors: ImageIntegrationDescriptor[] = [
    {
        categories: 'Image Scanner + Node Scanner',
        Logo: LogoSvg,
        label: 'Scanner V4',
        type: 'scannerv4',
        featureFlagDependency: ['ROX_SCANNER_V4'],
    },
    {
        categories: 'Registry',
        Logo: DockerSvg,
        label: 'Generic Docker Registry',
        type: 'docker',
    },
    {
        categories: 'Registry',
        Logo: AwsSvg,
        label: 'Amazon ECR',
        type: 'ecr',
    },
    {
        categories: 'Registry',
        Logo: GoogleArtifactSvg,
        label: 'Google Artifact Registry',
        type: 'artifactregistry',
    },
    {
        categories: 'Registry',
        Logo: AzureSvg,
        label: 'Microsoft ACR',
        type: 'azure',
    },
    {
        categories: 'Registry',
        Logo: ArtifactorySvg,
        label: 'JFrog Artifactory',
        type: 'artifactory',
    },
    {
        categories: 'Registry + Scanner',
        Logo: QuaySvg,
        label: 'Quay.io',
        type: 'quay',
    },
    {
        categories: 'Scanner',
        Logo: ClairSvg,
        label: 'Clair v4',
        type: 'clairV4',
    },
    {
        categories: 'Registry',
        Logo: NexusSvg,
        label: 'Sonatype Nexus',
        type: 'nexus',
    },
    {
        categories: 'Registry',
        Logo: IbmSvg,
        label: 'IBM Cloud',
        type: 'ibm',
    },
    {
        categories: 'Registry',
        Logo: RedhatSvg,
        label: 'Red Hat',
        type: 'rhel',
    },
    {
        categories: 'Registry',
        Logo: GhcrSvg,
        label: 'GitHub Container Registry',
        type: 'ghcr',
    },
    {
        categories: 'Image Scanner + Node Scanner',
        Logo: LogoSvg,
        label: '[DEPRECATED] StackRox Scanner',
        type: 'clairify',
    },
    {
        categories: 'Scanner',
        Logo: ClairSvg,
        label: '[DEPRECATED] CoreOS Clair',
        type: 'clair',
    },
    {
        categories: 'Registry + Scanner',
        Logo: GoogleRegistrySvg,
        label: '[DEPRECATED] Google Container Registry',
        type: 'google',
    },
];

export const signatureIntegrationsSource = 'signatureIntegrations';

export const signatureIntegrationDescriptor: SignatureIntegrationDescriptor = {
    Logo: SignatureSvg,
    label: 'Signature',
    type: 'signature',
};

const signatureIntegrationsDescriptors = [signatureIntegrationDescriptor];

export const notifierIntegrationsSource = 'notifiers';

export const notifierIntegrationsDescriptors: NotifierIntegrationDescriptor[] = [
    {
        Logo: SlackSvg,
        label: 'Slack',
        type: 'slack',
    },
    {
        Logo: LogoSvg,
        label: 'Generic Webhook',
        type: 'generic',
    },
    {
        Logo: JiraSvg,
        label: 'Jira',
        type: 'jira',
    },
    {
        Logo: EmailSvg,
        label: 'Email',
        type: 'email',
    },
    {
        Logo: AcscsEmailSvg,
        label: 'RHACS Cloud Service',
        type: 'acscsEmail',
    },
    {
        Logo: GoogleSvg,
        label: 'Google Cloud SCC',
        type: 'cscc',
    },
    {
        Logo: SplunkSvg,
        label: 'Splunk',
        type: 'splunk',
    },
    {
        Logo: PagerDutySvg,
        label: 'PagerDuty',
        type: 'pagerduty',
    },
    {
        Logo: SumologicSvg,
        label: 'Sumo Logic',
        type: 'sumologic',
    },
    {
        Logo: TeamsSvg,
        label: 'Microsoft Teams',
        type: 'teams',
    },
    {
        Logo: AwsSecurityHubSvg,
        label: 'AWS Security Hub',
        type: 'awsSecurityHub',
    },
    {
        Logo: SyslogSvg,
        label: 'Syslog',
        type: 'syslog',
    },
    {
        Logo: MicrosoftSentinelSvg,
        label: 'Microsoft Sentinel',
        type: 'microsoftSentinel',
    },
];

export const backupIntegrationsSource = 'backups';

export const backupIntegrationsDescriptors: BackupIntegrationDescriptor[] = [
    {
        Logo: S3Svg,
        label: 'Amazon S3',
        type: 's3',
    },
    {
        Logo: S3CompatibleSvg,
        label: 'S3 API Compatible',
        type: 's3compatible',
    },
    {
        Logo: GoogleSvg,
        label: 'Google Cloud Storage',
        type: 'gcs',
    },
];

export const authenticationTokensSource = 'authProviders';

export const apiTokenDescriptor: AuthProviderDescriptor = {
    Logo: LogoSvg,
    label: 'API Token',
    type: 'apitoken',
};

export const machineAccessDescriptor: AuthProviderDescriptor = {
    Logo: LogoSvg,
    label: 'Machine access configuration',
    type: 'machineAccess',
};

const authenticationTokensDescriptors = [apiTokenDescriptor, machineAccessDescriptor];

export const cloudSourcesSource = 'cloudSources';

export const paladinCloudDescriptor: CloudSourceDescriptor = {
    Logo: PaladinCloudSvg,
    label: 'Paladin Cloud',
    type: 'paladinCloud',
};

export const ocmDescriptor: CloudSourceDescriptor = {
    Logo: RedhatSvg,
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

// Adapted from RouteRequirements and routeRequirementsMap from routePaths.ts file.

type IntegrationsRouteRequirements = {
    centralCapabilityRequirement?: CentralCapabilitiesFlags;
    featureFlagRequirements?: FeatureFlagPredicate;
};

const integrationSourceRequirementsMap: Record<IntegrationSource, IntegrationsRouteRequirements> = {
    imageIntegrations: {},
    signatureIntegrations: {},
    notifiers: {},
    backups: { centralCapabilityRequirement: 'centralCanUseCloudBackupIntegrations' },
    cloudSources: {},
    authProviders: {},
};

export type IntegrationsRoutePredicates = {
    isCentralCapabilityAvailable: IsCentralCapabilityAvailable;
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
};

function isIntegrationsRouteEnabled(
    { isCentralCapabilityAvailable, isFeatureFlagEnabled }: IntegrationsRoutePredicates,
    { centralCapabilityRequirement, featureFlagRequirements }: IntegrationsRouteRequirements
) {
    if (
        centralCapabilityRequirement &&
        !isCentralCapabilityAvailable(centralCapabilityRequirement)
    ) {
        return false;
    }

    if (featureFlagRequirements && !featureFlagRequirements(isFeatureFlagEnabled)) {
        return false;
    }

    return true;
}

export function getSourcesEnabled(predicates: IntegrationsRoutePredicates): IntegrationSource[] {
    return integrationSources.filter((source) =>
        isIntegrationsRouteEnabled(predicates, integrationSourceRequirementsMap[source])
    );
}

export function getTypesEnabled(
    predicates: IntegrationsRoutePredicates,
    source: IntegrationSource
): IntegrationType[] {
    return getDescriptors(source)
        .filter((descriptor) => isIntegrationsRouteEnabled(predicates, descriptor))
        .map(({ type }) => type as IntegrationType);
}

export const integrationSourceTitleMap: Record<IntegrationSource, string> = {
    imageIntegrations: 'Image',
    signatureIntegrations: 'Signature',
    notifiers: 'Notifier',
    backups: 'Backup',
    cloudSources: 'Cloud source',
    authProviders: 'Authentication',
};

export function getIntegrationTabPath(source: IntegrationSource) {
    return `${integrationsPath}/${source}`; // tabs need full instead of relative path
}

export function getIntegrationsListPath(source: IntegrationSource, type: string) {
    return `${integrationsPath}/${source}/${type}`;
}
