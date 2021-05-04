import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
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
        <div className="flex flex-1 flex-col text-sm max-h-minus-buttons min-w-0">
            <div className="bg-primary-800 flex items-center m-2 min-w-108 p-3 rounded-lg shadow text-primary-100">
                <div className="flex flex-1 flex-col">
                    <div>{selectedNamespace.id}</div>
                    <div className="italic text-primary-200 text-xs capitalize">
                        Deployments in this namespace
                    </div>
                </div>
            </div>
            <div className="flex flex-1 m-2 pb-1 overflow-auto rounded bg-base-100">
                <NamespaceDeployments deployments={selectedNamespace.deployments} />
            </div>
        </div>
    );
}

const mapStateToProps = createStructuredSelector({
    selectedNamespace: selectors.getSelectedNamespace,
});

export default connect(mapStateToProps, null)(NamespaceDetailsOverlay);
