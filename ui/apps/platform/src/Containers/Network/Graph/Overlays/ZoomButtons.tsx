import React, { ReactElement } from 'react';
import * as Icon from 'react-feather';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import Button from 'Components/Button';

type ZoomButtonsProps = {
    pinnedLeft?: boolean;
    networkGraphRef?: {
        zoomToFit: () => void;
        zoomIn: () => void;
        zoomOut: () => void;
    };
};

function ZoomButtons({
    pinnedLeft = false,
    networkGraphRef = undefined,
}: ZoomButtonsProps): ReactElement {
    function zoomToFit() {
        networkGraphRef?.zoomToFit();
    }

    function zoomIn() {
        networkGraphRef?.zoomIn();
    }

    function zoomOut() {
        networkGraphRef?.zoomOut();
    }

    return (
        <div
            className={`flex absolute bottom-0 ${
                pinnedLeft ? 'pin-network-zoom-buttons-left' : ''
            }`}
        >
            <div className="border-2 border-base-400 my-4">
                <Button
                    className="btn-icon btn-base border-b border-base-300"
                    icon={<Icon.Maximize className="h-4 w-4" />}
                    onClick={zoomToFit}
                />
            </div>
            <div className="flex graph-zoom-buttons m-4 border-2 border-base-400">
                <Button
                    className="btn-icon btn-base border-b border-base-300"
                    icon={<Icon.Plus className="h-4 w-4" />}
                    onClick={zoomIn}
                />
                <Button
                    className="btn-icon btn-base shadow"
                    icon={<Icon.Minus className="h-4 w-4" />}
                    onClick={zoomOut}
                />
            </div>
        </div>
    );
}

const mapStateToProps = createStructuredSelector({
    networkGraphRef: selectors.getNetworkGraphRef,
});

export default connect(mapStateToProps)(ZoomButtons);
