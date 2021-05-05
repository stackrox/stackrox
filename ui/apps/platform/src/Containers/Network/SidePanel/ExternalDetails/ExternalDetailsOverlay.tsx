import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import useNavigateToEntity from 'Containers/Network/SidePanel/useNavigateToEntity';
import { selectors } from 'reducers';
import DetailsOverlay from 'Components/DetailsOverlay';
import NetworkFlows from '../Details/NetworkFlows';

function ExternalDetailsOverlay({ selectedNode }): ReactElement {
    const onNavigateToEntity = useNavigateToEntity();

    const { edges, cidr, name } = selectedNode;
    // TODO remove type casts when selectedNode prop has a type.
    const headerName = cidr ? `${name as string} | ${cidr as string}` : name;

    return (
        <DetailsOverlay
            headerText={headerName}
            subHeaderText="Connected entities outside your cluster"
            dataTestId="external-details-overlay"
        >
            <div className="flex flex-1 bg-base-100 rounded">
                <NetworkFlows
                    edges={edges}
                    filterState={1}
                    onNavigateToDeploymentById={onNavigateToEntity}
                />
            </div>
        </DetailsOverlay>
    );
}

const mapStateToProps = createStructuredSelector({
    selectedNode: selectors.getSelectedNode,
});

export default connect(mapStateToProps, null)(ExternalDetailsOverlay);
