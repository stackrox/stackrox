import React from 'react';
import { render, screen } from '@testing-library/react';

import NodeTooltipOverlay from './NodeTooltipOverlay';

test('should show listening ports', async () => {
    const listenPorts = [
        { port: 8443, l4protocol: 'L4_PROTOCOL_TCP' },
        { port: 9090, l4protocol: 'L4_PROTOCOL_TCP' },
    ];
    render(
        <NodeTooltipOverlay
            deploymentName="Test"
            numIngressFlows={0}
            numEgressFlows={0}
            ingressPortsAndProtocols={[]}
            egressPortsAndProtocols={[]}
            listenPorts={listenPorts}
            showPortsAndProtocols
        />
    );
    expect(screen.getByText('Listening Ports: 2')).toBeInTheDocument();
    expect(screen.getByText('TCP:')).toBeInTheDocument();
    expect(screen.getByText('8443, 9090')).toBeInTheDocument();
});
