import React, { ReactNode } from 'react';
import { Panel, PanelHeader, PanelMain, PanelMainBody, Divider } from '@patternfly/react-core';

type ClusterHealthPanelProps = {
    children: ReactNode;
    header: ReactNode;
};

function ClusterHealthPanel({ children, header }: ClusterHealthPanelProps) {
    return (
        <Panel variant="bordered" className="pf-v5-u-h-100">
            <PanelHeader>{header}</PanelHeader>
            <Divider />
            <PanelMain>
                <PanelMainBody>{children}</PanelMainBody>
            </PanelMain>
        </Panel>
    );
}

export default ClusterHealthPanel;
