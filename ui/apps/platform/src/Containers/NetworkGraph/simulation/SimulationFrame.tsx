import React from 'react';
import type { ReactElement, ReactNode } from 'react';
import { ScreenIcon } from '@patternfly/react-icons';

import { Flex, FlexItem } from '@patternfly/react-core';

type SimulationFrameProps = {
    isSimulating: boolean;
    children: ReactNode;
};

function SimulationFrame({ isSimulating, children }: SimulationFrameProps): ReactElement {
    let style = {};
    if (isSimulating) {
        style = { position: 'relative', border: '5px solid var(--pf-v5-global--info-color--100' };
    } else {
        style = {};
    }
    // Simulation frame and rectangle at upper left have same colors as inline info alert:
    // border and icon have same color
    // background color
    // text has same (ordinary) color as title or body for sufficient color contrast
    return (
        <div className="pf-ri__topology-section" style={style}>
            {children}
            {isSimulating && (
                <Flex
                    className="pf-v5-u-p-sm pf-v5-u-background-color-info"
                    style={{
                        position: 'absolute',
                        left: '0',
                        top: '0',
                        zIndex: 100,
                    }}
                    alignItems={{ default: 'alignItemsCenter' }}
                    spaceItems={{ default: 'spaceItemsSm' }}
                >
                    <FlexItem>
                        <ScreenIcon className="pf-v5-u-info-color-100" />
                    </FlexItem>
                    <FlexItem>Simulated view</FlexItem>
                </Flex>
            )}
        </div>
    );
}

export default SimulationFrame;
