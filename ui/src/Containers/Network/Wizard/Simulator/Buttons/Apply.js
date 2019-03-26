import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import * as Icon from 'react-feather';
import { selectors } from 'reducers';
import { actions as dialogueActions } from 'reducers/network/dialogue';

import dialogueStages from 'Containers/Network/Dialogue/dialogueStages';

class Apply extends Component {
    static propTypes = {
        applicationState: PropTypes.string.isRequired,
        setDialogueStage: PropTypes.func.isRequired
    };

    onClick = () => {
        this.props.setDialogueStage(dialogueStages.application);
    };

    render() {
        const { applicationState } = this.props;

        return (
            <div>
                <button
                    type="button"
                    className="inline-block flex my-3 px-3 text-center bg-primary-600 font-700 rounded-sm text-base-100 h-9 hover:bg-primary-700"
                    onClick={this.onClick}
                    disabled={applicationState === 'REQUEST'}
                >
                    <Icon.Save className="h-4 w-4 mr-2" />
                    Apply Network Policies
                </button>
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    applicationState: selectors.getNetworkPolicyApplicationState
});

const mapDispatchToProps = {
    setDialogueStage: dialogueActions.setNetworkDialogueStage
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Apply);
