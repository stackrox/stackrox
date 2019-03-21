import React, { Component } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import Tooltip from 'rc-tooltip';
import { actions as wizardActions } from 'reducers/network/wizard';

import generate from 'images/generate.svg';

class Generate extends Component {
    static propTypes = {
        generatePolicyModification: PropTypes.func.isRequired
    };

    onClick = () => {
        this.props.generatePolicyModification();
    };

    render() {
        return (
            <button
                type="button"
                className="inline-block px-2 py-2 border-r border-base-300 cursor-pointer"
                onClick={this.onClick}
            >
                <Tooltip
                    placement="top"
                    overlay={<div>Generate a new YAML</div>}
                    mouseEnterDelay={0.5}
                    mouseLeaveDelay={0}
                >
                    <img
                        className="text-primary-700 h-4 w-4 hover:bg-base-200"
                        alt=""
                        src={generate}
                    />
                </Tooltip>
            </button>
        );
    }
}

const mapDispatchToProps = {
    generatePolicyModification: wizardActions.generateNetworkPolicyModification
};

export default connect(
    null,
    mapDispatchToProps
)(Generate);
