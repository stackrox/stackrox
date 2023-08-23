import React from 'react';
import { Badge } from '@patternfly/react-core';

export const clusterBadgeText = 'CL';
export const namespaceBadgeText = 'NS';
export const deploymentBadgeText = 'D';
export const cidrBlockBadgeText = 'CB';
export const externalEntitiesBadgeText = 'E';

export const clusterBadgeColor = 'var(--pf-global--palette--purple-500)';
export const namespaceBadgeColor = 'var(--pf-global--palette--green-500)';
export const deploymentBadgeColor = 'var(--pf-global--palette--blue-500)';
export const cidrBlockBadgeColor = 'var(--pf-global--palette--light-blue-600)';
export const externalEntitiesBadgeColor = 'var(--pf-global--palette--black-850)';

export function DeploymentIcon(props) {
    return (
        <Badge {...props} style={{ backgroundColor: deploymentBadgeColor }}>
            {deploymentBadgeText}
        </Badge>
    );
}

export function NamespaceIcon(props) {
    return (
        <Badge {...props} style={{ backgroundColor: namespaceBadgeColor }}>
            {namespaceBadgeText}
        </Badge>
    );
}

export function ClusterIcon(props) {
    return (
        <Badge {...props} style={{ backgroundColor: clusterBadgeColor }}>
            {clusterBadgeText}
        </Badge>
    );
}

export function CidrBlockIcon(props) {
    return (
        <Badge {...props} style={{ backgroundColor: cidrBlockBadgeColor }}>
            {cidrBlockBadgeText}
        </Badge>
    );
}

export function ExternalEntitiesIcon(props) {
    return (
        <Badge {...props} style={{ backgroundColor: externalEntitiesBadgeColor }}>
            {externalEntitiesBadgeText}
        </Badge>
    );
}
