import React, { ReactElement } from 'react';

import KeyValuePairs from 'Components/KeyValuePairs';

type ContainerSecurityContextMap = {
    privileged: { label: string };
    add_capabilities: { label: string };
    drop_capabilities: { label: string };
};

const containerSecurityContextMap: ContainerSecurityContextMap = {
    privileged: { label: 'Privileged' },
    add_capabilities: { label: 'Add Capabilities' },
    drop_capabilities: { label: 'Drop Capabilities' },
};

const getSecurityContext = (container): ContainerSecurityContextMap | null => {
    if (!container.securityContext) {
        return null;
    }
    const { privileged, add_capabilities, drop_capabilities } = container.securityContext; // eslint-disable-line
    return { privileged, add_capabilities, drop_capabilities };
};

const SecurityContext = ({ deployment }): ReactElement => {
    let containerResult: ReactElement | ReactElement[];
    if (deployment.containers) {
        const containers = deployment.containers
            .filter((container) => !!container.securityContext)
            .map((container) => {
                const securityContext = getSecurityContext(container);
                if (!securityContext || JSON.stringify(securityContext) === '{}') {
                    return null;
                }
                return (
                    <div key={container.toString()}>
                        {securityContext && (
                            <KeyValuePairs
                                data={securityContext}
                                keyValueMap={containerSecurityContextMap}
                            />
                        )}
                    </div>
                );
            });
        containerResult = containers.length
            ? containers
            : (containerResult = <span className="py-3 font-600">None</span>);
    } else {
        containerResult = <span className="py-3 font-600">None</span>;
    }
    return <div className="flex h-full px-3">{containerResult}</div>;
};

export default SecurityContext;
