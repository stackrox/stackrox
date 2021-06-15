import React, { ReactElement } from 'react';

type EffectiveAccessScopeLabelsProps = {
    labels: Record<string, string>;
    isExpanded?: boolean;
};

function EffectiveAccessScopeLabels({
    labels,
    isExpanded,
}: EffectiveAccessScopeLabelsProps): ReactElement {
    const entries = isExpanded ? Object.entries(labels) : Object.entries(labels).slice(0, 1);

    return (
        <ul>
            {entries.map(([key, value]) => (
                <li key={key}>
                    <span className="pf-u-font-weight-bold">{key}</span>
                    <span>: {value}</span>
                </li>
            ))}
        </ul>
    );
}

export default EffectiveAccessScopeLabels;
