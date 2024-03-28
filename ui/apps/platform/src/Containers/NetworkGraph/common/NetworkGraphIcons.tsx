import React from 'react';
import { Badge, BadgeProps } from '@patternfly/react-core';

export const clusterBadgeText = 'CL';
export const namespaceBadgeText = 'NS';
export const deploymentBadgeText = 'D';
export const cidrBlockBadgeText = 'CB';
export const externalEntitiesBadgeText = 'E';
export const internalEntitiesBadgeText = 'IE';

export const clusterBadgeColor = 'var(--pf-v5-global--palette--purple-500)';
export const namespaceBadgeColor = 'var(--pf-v5-global--palette--green-500)';
export const deploymentBadgeColor = 'var(--pf-v5-global--palette--blue-500)';
export const cidrBlockBadgeColor = 'var(--pf-v5-global--palette--light-blue-600)';
export const externalEntitiesBadgeColor = 'var(--pf-v5-global--palette--black-850)';
export const internalEntitiesBadgeColor = 'var(--pf-v5-global--palette--black-700)';

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
