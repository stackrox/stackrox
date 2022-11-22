import React from 'react';
import { Badge } from '@patternfly/react-core';

export function DeploymentIcon() {
    return <Badge style={{ backgroundColor: 'rgb(0,102,205)' }}>D</Badge>;
}

export function NamespaceIcon() {
    return <Badge style={{ backgroundColor: 'rgb(32,79,23)' }}>NS</Badge>;
}
