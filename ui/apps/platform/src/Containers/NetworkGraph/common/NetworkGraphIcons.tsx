import React from 'react';
import { Badge } from '@patternfly/react-core';

export function DeploymentIcon(props) {
    return (
        <Badge {...props} style={{ backgroundColor: 'rgb(0,102,205)' }}>
            D
        </Badge>
    );
}

export function NamespaceIcon(props) {
    return (
        <Badge {...props} style={{ backgroundColor: 'rgb(32,79,23)' }}>
            NS
        </Badge>
    );
}

export function ClusterIcon(props) {
    return (
        <Badge {...props} style={{ backgroundColor: 'rgb(132,118,209)' }}>
            CL
        </Badge>
    );
}

export function CidrBlockIcon() {
    return <Badge style={{ backgroundColor: 'rgb(9,143,177)' }}>CB</Badge>;
}

export function ExternalEntitiesIcon() {
    return <Badge style={{ backgroundColor: 'rgb(0,0,0)' }}>E</Badge>;
}
