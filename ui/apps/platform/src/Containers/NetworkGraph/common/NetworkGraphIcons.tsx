import { Badge } from '@patternfly/react-core';
import type { BadgeProps } from '@patternfly/react-core';

export const clusterBadgeText = 'CL';
export const namespaceBadgeText = 'NS';
export const deploymentBadgeText = 'D';
export const cidrBlockBadgeText = 'CB';
export const externalEntitiesBadgeText = 'E';
export const internalEntitiesBadgeText = 'IE';

export const clusterBadgeColor = 'var(--pf-t--color--purple--50)';
export const namespaceBadgeColor = 'var(--pf-t--color--green--50)';
export const deploymentBadgeColor = 'var(--pf-t--color--blue--50)';
export const cidrBlockBadgeColor = 'var(--pf-t--color--blue--60)';
export const externalEntitiesBadgeColor = 'var(--pf-t--color--gray--80)';
export const internalEntitiesBadgeColor = 'var(--pf-t--color--gray--70)';

export function DeploymentIcon(props: BadgeProps) {
    return (
        <Badge {...props} style={{ backgroundColor: deploymentBadgeColor }}>
            {deploymentBadgeText}
        </Badge>
    );
}

export function NamespaceIcon(props: BadgeProps) {
    return (
        <Badge {...props} style={{ backgroundColor: namespaceBadgeColor }}>
            {namespaceBadgeText}
        </Badge>
    );
}

export function ClusterIcon(props: BadgeProps) {
    return (
        <Badge {...props} style={{ backgroundColor: clusterBadgeColor }}>
            {clusterBadgeText}
        </Badge>
    );
}

export function CidrBlockIcon(props: BadgeProps) {
    return (
        <Badge {...props} style={{ backgroundColor: cidrBlockBadgeColor }}>
            {cidrBlockBadgeText}
        </Badge>
    );
}

export function ExternalEntitiesIcon(props: BadgeProps) {
    return (
        <Badge {...props} style={{ backgroundColor: externalEntitiesBadgeColor }}>
            {externalEntitiesBadgeText}
        </Badge>
    );
}

export function InternalEntitiesIcon(props: BadgeProps) {
    return (
        <Badge {...props} style={{ backgroundColor: internalEntitiesBadgeColor }}>
            {internalEntitiesBadgeText}
        </Badge>
    );
}
