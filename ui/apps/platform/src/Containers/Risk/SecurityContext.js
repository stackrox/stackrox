import React from 'react';

import KeyValuePairs from 'Components/KeyValuePairs';
import CollapsibleCard from 'Components/CollapsibleCard';

const containerSecurityContextMap = {
    privileged: { label: 'Privileged' },
    add_capabilities: { label: 'Add Capabilities' },
    drop_capabilities: { label: 'Drop Capabilities' },
};

const getSecurityContext = (container) => {
    if (!container.securityContext) {
        return null;
    }
    const { privileged, add_capabilities, drop_capabilities } = container.securityContext; // eslint-disable-line
    return { privileged, add_capabilities, drop_capabilities };
};

const SecurityContext = ({ deployment }) => {
    let containers = [];
    if (deployment.containers) {
        containers = deployment.containers
            .filter((container) => container.securityContext)
            .map((container) => {
                const data = getSecurityContext(container);
                if (data === {}) {
                    return null;
                }
                return (
                    <div key={container.id}>
                        {data && (
                            <KeyValuePairs data={data} keyValueMap={containerSecurityContextMap} />
                        )}
                    </div>
                );
            });
        if (!containers.length) {
            containers = <span className="py-3 font-600 italic">None</span>;
        }
    } else {
        containers = <span className="py-3 font-600 italic">None</span>;
    }
    return (
        <div className="px-3 pt-5">
            <div className="bg-base-100 text-primary-600 tracking-wide">
                <CollapsibleCard title="Security Context">
                    <div className="flex h-full px-3">{containers}</div>
                </CollapsibleCard>
            </div>
        </div>
    );
};

export default SecurityContext;
