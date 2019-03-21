import React, { Component } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import Tooltip from 'rc-tooltip';
import * as Icon from 'react-feather';
import { actions as wizardActions } from 'reducers/network/wizard';

class Undo extends Component {
    static propTypes = {
        undoModification: PropTypes.func.isRequired
    };

    onClick = () => {
        this.props.undoModification();
    };

    render() {
        return (
            <button
                type="button"
                className="inline-block px-2 py-2 border-l border-r border-base-300 cursor-pointer"
                onClick={this.onClick}
            >
                <Tooltip
                    placement="top"
                    overlay={<div>Revert most recently applied YAML</div>}
                    mouseEnterDelay={0.5}
                    mouseLeaveDelay={0}
                >
                    <Icon.RotateCcw className="h-4 w-4 text-base-500 hover:bg-base-200" />
                </Tooltip>
            </button>
        );
    }
}

const mapDispatchToProps = {
    undoModification: wizardActions.loadUndoNetworkPolicyModification
};

export default connect(
    null,
    mapDispatchToProps
)(Undo);
