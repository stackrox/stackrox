import type { ReactNode } from 'react';
import CollapsibleCard from 'Components/CollapsibleCard';

import type { Container, Deployment } from 'types/deployment.proto';

import KeyValuePairs from './KeyValuePairs';

const containerSecurityContextMap = {
    privileged: { label: 'Privileged' },
    add_capabilities: { label: 'Add Capabilities' },
    drop_capabilities: { label: 'Drop Capabilities' },
};

const getSecurityContext = (container: Container) => {
    if (!container.securityContext) {
        return null;
    }
    // @ts-expect-error TODO: add_capabilities and drop_capabilities are not typed in the proto file
    // TODO: Do we need to update the proto file or the code here? Should be camelCase?
    const { privileged, add_capabilities, drop_capabilities } = container.securityContext;
    return { privileged, add_capabilities, drop_capabilities };
};

type SecurityContextProps = {
    deployment: Deployment;
};

function SecurityContext({ deployment }: SecurityContextProps) {
    let containers: ReactNode | ReactNode[] = [];
    if (deployment.containers) {
        containers = deployment.containers
            .filter((container) => container.securityContext)
            .map((container) => {
                const data = getSecurityContext(container);
                if (data === null || !Object.values(data).some((value) => !!value)) {
                    return null;
                }
                return (
                    <div key={container.id}>
                        <KeyValuePairs data={data} keyValueMap={containerSecurityContextMap} />
                    </div>
                );
            });
        if (!Array.isArray(containers) || containers.length === 0) {
            containers = <span className="py-3">None</span>;
        }
    } else {
        containers = <span className="py-3">None</span>;
    }
    return (
        <div className="px-3 pt-5">
            <div className="bg-base-100 text-primary-600">
                <CollapsibleCard title="Security Context">
                    <div className="flex h-full px-3">{containers}</div>
                </CollapsibleCard>
            </div>
        </div>
    );
}

export default SecurityContext;
