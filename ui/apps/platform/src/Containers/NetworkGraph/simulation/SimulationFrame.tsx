import React from 'react';
import { ScreenIcon } from '@patternfly/react-icons';

import { Flex, FlexItem } from '@patternfly/react-core';
import { NetworkPolicySimulator } from '../hooks/useNetworkPolicySimulator';

type SimulationFrameProps = {
    simulator: NetworkPolicySimulator;
    children: React.ReactNode;
};

function SimulationFrame({ simulator, children }: SimulationFrameProps) {
    const isSimulating =
        simulator.state === 'GENERATED' ||
        simulator.state === 'UNDO' ||
        simulator.state === 'UPLOAD';

    let style = {};
    if (isSimulating) {
        style = { position: 'relative', border: '5px solid rgb(115,188,247)' };
    } else {
        style = {};
    }
    return (
        <div className="pf-ri__topology-demo" style={style}>
            {children}
            {isSimulating && (
                <Flex
                    className="pf-u-p-sm"
                    style={{
                        backgroundColor: 'rgb(224,233,242)',
                        position: 'absolute',
                        left: '0',
                        top: '0',
                        zIndex: 100,
                    }}
                    alignItems={{ default: 'alignItemsCenter' }}
                >
                    <FlexItem>
                        <ScreenIcon className="pf-u-info-color-100" />
                    </FlexItem>
                    <FlexItem>
                        <div className="pf-u-info-color-100">Simulated view</div>
                    </FlexItem>
                </Flex>
            )}
        </div>
    );
}

export default SimulationFrame;
