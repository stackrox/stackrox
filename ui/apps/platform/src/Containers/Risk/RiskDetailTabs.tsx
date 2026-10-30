import { Alert, Flex, Label, PageSection, Tab, TabTitleText, Tabs } from '@patternfly/react-core';

import usePermissions from 'hooks/usePermissions';
import useURLStringUnion from 'hooks/useURLStringUnion';
import type { Deployment } from 'types/deployment.proto';
import type { Risk } from 'services/DeploymentsService';
import type { RiskPriorityClassification } from 'types/risk.proto';

import DeploymentDetails from './DeploymentDetails';
import ProcessDiscovery from './Process/ProcessDiscovery';
import RiskIndicatorCard from './Indicators/RiskIndicatorCard';

const riskIndicatorsTab = 'Risk indicators';
const deploymentDetailsTab = 'Deployment details';
const processDiscoveryTab = 'Process discovery';

export type RiskDetailTabsProps = {
    deployment: Deployment;
    risk: Risk | null | undefined;
};

const riskClassLabels: Record<RiskPriorityClassification, string> = {
    RISK_PRIORITY_UNKNOWN: 'Unknown',
    RISK_PRIORITY_RESPOND_NOW: 'Respond now',
    RISK_PRIORITY_INVESTIGATE_NOW: 'Investigate now',
    RISK_PRIORITY_REMEDIATE_NOW: 'Remediate now',
    RISK_PRIORITY_REMEDIATE: 'Remediate',
    RISK_PRIORITY_IMPROVE_POSTURE: 'Improve posture',
};

const riskClassColors: Record<
    RiskPriorityClassification,
    'red' | 'orange' | 'yellow' | 'blue' | 'green' | 'grey'
> = {
    RISK_PRIORITY_UNKNOWN: 'grey',
    RISK_PRIORITY_RESPOND_NOW: 'red',
    RISK_PRIORITY_INVESTIGATE_NOW: 'orange',
    RISK_PRIORITY_REMEDIATE_NOW: 'red',
    RISK_PRIORITY_REMEDIATE: 'yellow',
    RISK_PRIORITY_IMPROVE_POSTURE: 'blue',
};

const riskClassDescriptions: Record<RiskPriorityClassification, string> = {
    RISK_PRIORITY_UNKNOWN:
        'ACS did not find enough evidence to assign this workload to an action-oriented risk class.',
    RISK_PRIORITY_RESPOND_NOW:
        'Observed evidence is consistent with active or confirmed malicious behavior. Security response should take priority over remediation planning.',
    RISK_PRIORITY_INVESTIGATE_NOW:
        'Observed suspicious runtime behavior needs investigation, but ACS does not have enough evidence to classify it as confirmed compromise.',
    RISK_PRIORITY_REMEDIATE_NOW:
        'A material exposure exists and should be remediated immediately, but ACS has not observed exploitation.',
    RISK_PRIORITY_REMEDIATE:
        'A material security issue should be fixed, but ACS does not currently see the exposure or runtime evidence that would make it immediate.',
    RISK_PRIORITY_IMPROVE_POSTURE:
        'The workload has hygiene or backlog findings that should be planned, but they should not compete with urgent response or remediation work.',
};

function hasRiskResult(risk: Risk, resultName: string): boolean {
    return risk.results.some(({ name }) => name === resultName);
}

function getRiskClassReasons(deployment: Deployment, risk: Risk): string[] {
    const reasons: string[] = [];
    const hasExternalExposure = deployment.ports.some(
        ({ exposure }) => exposure === 'EXTERNAL' || exposure === 'ROUTE'
    );
    const mountsSecret = deployment.containers.some(({ secrets }) => secrets.length !== 0);
    const hasPowerfulIdentity =
        deployment.automountServiceAccountToken &&
        (deployment.serviceAccountPermissionLevel === 'ELEVATED_CLUSTER_WIDE' ||
            deployment.serviceAccountPermissionLevel === 'CLUSTER_ADMIN');
    const hasDangerousPrivilege =
        deployment.hostNetwork ||
        deployment.hostPid ||
        deployment.hostIpc ||
        deployment.containers.some(({ securityContext }) => {
            const addedCapabilities = securityContext?.addCapabilities ?? [];
            return (
                securityContext?.privileged ||
                addedCapabilities.some((capability) =>
                    ['SYS_ADMIN', 'NET_ADMIN', 'SYS_MODULE'].includes(capability.toUpperCase())
                )
            );
        });

    if (hasExternalExposure) {
        reasons.push('Workload has Route or external Service exposure.');
    }
    if (hasRiskResult(risk, 'Image Vulnerabilities')) {
        reasons.push('Image vulnerability findings contributed to this classification.');
    }
    if (hasRiskResult(risk, 'Suspicious Process Executions')) {
        reasons.push('Suspicious process-baseline activity was observed.');
    }
    if (hasRiskResult(risk, 'Policy Violations')) {
        reasons.push('Active policy violations contributed evidence.');
    }
    if (mountsSecret) {
        reasons.push('Workload mounts one or more Secrets.');
    }
    if (hasPowerfulIdentity) {
        reasons.push(
            'Workload uses an automounted elevated or cluster-wide service account token.'
        );
    }
    if (hasDangerousPrivilege) {
        reasons.push('Workload has dangerous privilege or host namespace configuration.');
    }
    if (hasRiskResult(risk, 'Components Useful for Attackers')) {
        reasons.push('Image contains attacker-useful utilities.');
    }
    if (hasRiskResult(risk, 'Number of Components in Image')) {
        reasons.push('Image component count contributed posture context.');
    }
    if (hasRiskResult(risk, 'Image Freshness')) {
        reasons.push('Image age contributed posture context.');
    }

    return reasons;
}

function RiskDetailTabs({ deployment, risk }: RiskDetailTabsProps) {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForDeploymentExtension = hasReadAccess('DeploymentExtension');

    const [activeTabKey, setActiveTabKey] = useURLStringUnion('contentTab', [
        riskIndicatorsTab,
        deploymentDetailsTab,
        processDiscoveryTab,
    ]);

    return (
        <>
            <PageSection type="tabs" padding={{ default: 'noPadding' }}>
                <Tabs
                    activeKey={activeTabKey}
                    onSelect={(_event, tabKey) => setActiveTabKey(tabKey)}
                    role="region"
                    usePageInsets
                >
                    <Tab
                        eventKey={riskIndicatorsTab}
                        title={<TabTitleText>{riskIndicatorsTab}</TabTitleText>}
                        tabContentId={riskIndicatorsTab}
                    />
                    <Tab
                        eventKey={deploymentDetailsTab}
                        title={<TabTitleText>{deploymentDetailsTab}</TabTitleText>}
                        tabContentId={deploymentDetailsTab}
                    />
                    {hasReadAccessForDeploymentExtension && (
                        <Tab
                            eventKey={processDiscoveryTab}
                            title={<TabTitleText>{processDiscoveryTab}</TabTitleText>}
                            tabContentId={processDiscoveryTab}
                        />
                    )}
                </Tabs>
            </PageSection>
            <PageSection id={activeTabKey}>
                {activeTabKey === riskIndicatorsTab &&
                    (risk ? (
                        <Flex
                            direction={{ default: 'column' }}
                            spaceItems={{ default: 'spaceItemsMd' }}
                        >
                            {risk.priorityClassification && (
                                <Alert
                                    variant="info"
                                    isInline
                                    component="p"
                                    title={
                                        <>
                                            Risk class:{' '}
                                            <Label
                                                color={riskClassColors[risk.priorityClassification]}
                                            >
                                                {riskClassLabels[risk.priorityClassification]}
                                            </Label>
                                        </>
                                    }
                                >
                                    <p>{riskClassDescriptions[risk.priorityClassification]}</p>
                                    {getRiskClassReasons(deployment, risk).length !== 0 && (
                                        <>
                                            <p>Classification evidence:</p>
                                            <ul>
                                                {getRiskClassReasons(deployment, risk).map(
                                                    (reason) => (
                                                        <li key={reason}>{reason}</li>
                                                    )
                                                )}
                                            </ul>
                                        </>
                                    )}
                                </Alert>
                            )}
                            {risk.results.map((result) => (
                                <RiskIndicatorCard key={result.name} result={result} />
                            ))}
                        </Flex>
                    ) : (
                        <Alert variant="warning" isInline title="Risk not found" component="p">
                            Risk for selected deployment may not have been processed.
                        </Alert>
                    ))}
                {activeTabKey === deploymentDetailsTab && (
                    <div className="flex flex-1 flex-col relative">
                        <div className="absolute w-full">
                            <DeploymentDetails deployment={deployment} />
                        </div>
                    </div>
                )}
                {activeTabKey === processDiscoveryTab && hasReadAccessForDeploymentExtension && (
                    <div className="flex flex-1 flex-col h-full relative">
                        <ProcessDiscovery deploymentId={deployment.id} />
                    </div>
                )}
            </PageSection>
        </>
    );
}

export default RiskDetailTabs;
