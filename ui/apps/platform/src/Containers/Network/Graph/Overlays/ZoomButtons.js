import React from 'react';
import PropTypes from 'prop-types';
import Button from 'Components/Button';
import * as Icon from 'react-feather';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';

const ZoomButtons = ({ pinnedLeft, networkGraphRef: graph }) => {
    function zoomToFit() {
        if (graph) {
            graph.zoomToFit();
        }
    }

    function zoomIn() {
        if (graph) {
            graph.zoomIn();
        }
    }

    function zoomOut() {
        if (graph) {
            graph.zoomOut();
        }
    }

    return (
        <div
            className={`flex theme-light absolute bottom-0 ${
                pinnedLeft && 'pin-network-zoom-buttons-left'
            } right-0`}
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
};

ZoomButtons.propTypes = {
    networkGraphRef: PropTypes.shape({
        zoomToFit: PropTypes.func,
        zoomIn: PropTypes.func,
        zoomOut: PropTypes.func,
    }),
    pinnedLeft: PropTypes.bool,
};

ZoomButtons.defaultProps = {
    networkGraphRef: null,
    pinnedLeft: false,
};

const mapStateToProps = createStructuredSelector({
    networkGraphRef: selectors.getNetworkGraphRef,
});

export default connect(mapStateToProps)(ZoomButtons);
