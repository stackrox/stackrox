import type { ReactElement, ReactNode } from 'react';
import { Divider, Panel, PanelHeader, PanelMain, PanelMainBody } from '@patternfly/react-core';

type ClusterHealthPanelProps = {
    children: ReactNode;
    header: ReactNode;
};

function ClusterHealthPanel({ children, header }: ClusterHealthPanelProps): ReactElement {
    return (
        <Panel variant="bordered" className="pf-v6-u-h-100">
            <PanelHeader>{header}</PanelHeader>
            <Divider />
            <PanelMain>
                <PanelMainBody>{children}</PanelMainBody>
            </PanelMain>
        </Panel>
    );
}

export default ClusterHealthPanel;
