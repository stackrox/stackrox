import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import * as Icon from 'react-feather';
import { selectors } from 'reducers';
import { actions as dialogueActions } from 'reducers/network/dialogue';

import dialogueStages from 'Containers/Network/Dialogue/dialogueStages';

class Notify extends Component {
    static propTypes = {
        notifiers: PropTypes.arrayOf(PropTypes.shape({})),
        setDialogueStage: PropTypes.func.isRequired
    };

    static defaultProps = {
        notifiers: []
    };

    onClick = () => {
        this.props.setDialogueStage(dialogueStages.notification);
    };

    render() {
        return (
            <div className="ml-3">
                <button
                    type="button"
                    className="inline-block flex my-3 px-3 text-center bg-primary-600 font-700 rounded-sm text-base-100 h-9 hover:bg-primary-700"
                    onClick={this.onClick}
                    disabled={this.props.notifiers.length === 0}
                >
                    <Icon.Share2 className="h-4 w-4 mr-2" />
                    Share YAML
                </button>
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    notifiers: selectors.getNotifiers
});

const mapDispatchToProps = {
    setDialogueStage: dialogueActions.setNetworkDialogueStage
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Notify);
