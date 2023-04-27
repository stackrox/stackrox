import React from 'react';
import { Badge } from '@patternfly/react-core';

export const clusterBadgeText = 'CL';
export const namespaceBadgeText = 'NS';
export const deploymentBadgeText = 'D';
export const cidrBlockBadgeText = 'CB';
export const externalEntitiesBadgeText = 'E';

export const clusterBadgeColor = '#8476d1';
export const namespaceBadgeColor = '#35842C';
export const deploymentBadgeColor = '#0566CA';
export const cidrBlockBadgeColor = '#008DAB';
export const externalEntitiesBadgeColor = '#000000';

export function DeploymentIcon(props) {
    return (
        <Badge {...props} style={{ backgroundColor: deploymentBadgeColor, width: 15 }}>
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

export function CidrBlockIcon() {
    return <Badge style={{ backgroundColor: cidrBlockBadgeColor }}>{cidrBlockBadgeText}</Badge>;
}

export function ExternalEntitiesIcon() {
    return (
        <Badge style={{ backgroundColor: externalEntitiesBadgeColor }}>
            {externalEntitiesBadgeText}
        </Badge>
    );
}
