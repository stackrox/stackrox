import React, { Component } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import Tooltip from 'Components/Tooltip';
import TooltipOverlay from 'Components/TooltipOverlay';
import { selectors } from 'reducers';
import { actions as wizardActions } from 'reducers/network/wizard';

class Undo extends Component {
    static propTypes = {
        applicationState: PropTypes.string.isRequired,
        undoModification: PropTypes.func.isRequired,
    };

    onClick = () => {
        this.props.undoModification();
    };

    render() {
        const { applicationState } = this.props;
        return (
            <Tooltip content={<TooltipOverlay>Revert most recently applied YAML</TooltipOverlay>}>
                <button
                    type="button"
                    disabled={applicationState === 'REQUEST'}
                    className="inline-block px-2 py-2 border-l border-r border-base-300 cursor-pointer"
                    onClick={this.onClick}
                >
                    <Icon.RotateCcw className="h-4 w-4 text-base-500 hover:bg-base-200" />
                </button>
            </Tooltip>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    applicationState: selectors.getNetworkPolicyApplicationState,
});

const mapDispatchToProps = {
    undoModification: wizardActions.loadUndoNetworkPolicyModification,
};

export default connect(mapStateToProps, mapDispatchToProps)(Undo);
