import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import * as Icon from 'react-feather';
import { actions as dialogueActions } from 'reducers/network/dialogue';

import dialogueStages from 'Containers/Network/Dialogue/dialogueStages';

class Apply extends Component {
    static propTypes = {
        setDialogueStage: PropTypes.func.isRequired
    };

    onClick = () => {
        this.props.setDialogueStage(dialogueStages.application);
    };

    render() {
        return (
            <div>
                <button
                    type="button"
                    className="inline-block flex my-3 px-3 text-center bg-primary-600 font-700 rounded-sm text-base-100 h-9 hover:bg-primary-700"
                    onClick={this.onClick}
                    disabled={false}
                >
                    <Icon.Save className="h-4 w-4 mr-2" />
                    Apply Network Policies
                </button>
            </div>
        );
    }
}

const mapDispatchToProps = {
    setDialogueStage: dialogueActions.setNetworkDialogueStage
};

export default connect(
    null,
    mapDispatchToProps
)(Apply);
