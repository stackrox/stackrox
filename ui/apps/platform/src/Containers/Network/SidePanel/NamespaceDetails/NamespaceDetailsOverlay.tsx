import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import DetailsOverlay from 'Components/DetailsOverlay';
import NamespaceDeployments from './NamespaceDeployments';

type NamespaceDetailsOverlayProps = {
    selectedNamespace: {
        id: string;
        deployments: [];
    };
};

function NamespaceDetailsOverlay({
    selectedNamespace,
}: NamespaceDetailsOverlayProps): ReactElement {
    return (
        <DetailsOverlay
            headerText={selectedNamespace.id}
            subHeaderText="Deployments in this namespace"
            dataTestId="namespace-details-overlay"
        >
            <div className="flex flex-1 bg-base-100 rounded">
                <NamespaceDeployments deployments={selectedNamespace.deployments} />
            </div>
        </DetailsOverlay>
    );
}

const mapStateToProps = createStructuredSelector({
    selectedNamespace: selectors.getSelectedNamespace,
});

export default connect(mapStateToProps, null)(NamespaceDetailsOverlay);
