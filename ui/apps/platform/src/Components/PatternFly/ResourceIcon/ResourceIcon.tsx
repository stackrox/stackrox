import React from 'react';

import './ResourceIcon.css';

type K8sResourceKind = 'Cluster' | 'Namespace' | 'Deployment';

export type ResourceIconProps = {
    className?: string;
    kind: K8sResourceKind;
};

const IconAttributes: Record<K8sResourceKind, { text: string; classNameSuffix: string }> = {
    Cluster: { text: 'CL', classNameSuffix: 'cluster' },
    Namespace: { text: 'NS', classNameSuffix: 'namespace' },
    Deployment: { text: 'D', classNameSuffix: 'deployment' },
} as const;

/**
 * A small badge-like icon used to represent a type of K8s resource. Note that the display
 * of these is modeled after those used on the OpenShift console front end.
 *
 * The much more detailed OS implementation can be found at
 * https://github.com/openshift/console/blob/7f866284272db0d5898384ff116cea18351c9957/frontend/public/components/utils/resource-icon.tsx
 *
 */
function ResourceIcon(props: ResourceIconProps) {
    const { text, classNameSuffix } = IconAttributes[props.kind];
    return (
        <span
            title={props.kind}
            className={`resource-icon resource-icon-${classNameSuffix} ${props.className ?? ''}`}
        >
            {text}
        </span>
    );
}

export default ResourceIcon;
