import React from 'react';

import './ResourceIcon.css';

type K8sResourceKind =
    | 'Cluster'
    | 'ConfigMap'
    | 'ClusterRoles'
    | 'ClusterRoleBindings'
    | 'NetworkPolicies'
    | 'SecurityContextConstraints'
    | 'EgressFirewalls'
    | 'Deployment'
    | 'Namespace'
    | 'Secret'
    | 'Unknown';

export type ResourceIconProps = {
    className?: string;
    kind: K8sResourceKind;
};

const IconAttributes: Record<K8sResourceKind, { text: string; backgroundColor: string }> = {
    Cluster: { text: 'CL', backgroundColor: 'var(--pf-global--palette--purple-500)' },
    ConfigMap: { text: 'CM', backgroundColor: 'var(--pf-global--palette--purple-600)' },
    ClusterRoles: { text: 'CR', backgroundColor: 'var(--pf-global--palette--purple-600)' },
    ClusterRoleBindings: { text: 'CRB', backgroundColor: 'var(--pf-global--palette--purple-600)' },
    NetworkPolicies: { text: 'NP', backgroundColor: 'var(--pf-global--palette--purple-600)' },
    SecurityContextConstraints: {
        text: 'SCC',
        backgroundColor: 'var(--pf-global--palette--purple-600)',
    },
    EgressFirewalls: { text: 'EF', backgroundColor: 'var(--pf-global--palette--purple-600)' },
    Deployment: { text: 'D', backgroundColor: 'var(--pf-global--palette--blue-500)' },
    Namespace: { text: 'NS', backgroundColor: 'var(--pf-global--palette--green-500)' },
    Secret: { text: 'S', backgroundColor: 'var(--pf-global--palette--orange-600)' },
    Unknown: { text: '?', backgroundColor: 'var(--pf-global--palette--black-700)' },
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
    const { text, backgroundColor } = IconAttributes[props.kind];
    return (
        <span
            title={props.kind}
            className={`resource-icon ${props.className ?? ''}`}
            style={{ backgroundColor }}
        >
            {text}
        </span>
    );
}

export default ResourceIcon;
