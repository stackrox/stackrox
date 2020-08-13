import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import * as Icon from 'react-feather';
import { selectors } from 'reducers';
import { actions as dialogueActions } from 'reducers/network/dialogue';

import dialogueStages from 'Containers/Network/Dialogue/dialogueStages';
import Button from './Button';

class Apply extends Component {
    static propTypes = {
        modification: PropTypes.shape({
            applyYaml: PropTypes.string.isRequired,
            toDelete: PropTypes.arrayOf(
                PropTypes.shape({
                    namespace: PropTypes.string.isRequired,
                    name: PropTypes.string.isRequired,
                })
            ),
        }).isRequired,
        applicationState: PropTypes.string.isRequired,
        setDialogueStage: PropTypes.func.isRequired,
    };

    onClick = () => {
        this.props.setDialogueStage(dialogueStages.application);
    };

    render() {
        const { applicationState } = this.props;
        const inRequest = applicationState === 'REQUEST';
        const { applyYaml, toDelete } = this.props.modification;
        const noModification = applyYaml === '' && (!toDelete || toDelete.length === 0);
        return (
            <div>
                <Button
                    onClick={this.onClick}
                    disabled={inRequest || noModification}
                    icon={<Icon.Save className="h-4 w-4 mr-2" />}
                    text="Apply Network Policies"
                />
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    modification: selectors.getNetworkPolicyModification,
    applicationState: selectors.getNetworkPolicyApplicationState,
});

const mapDispatchToProps = {
    setDialogueStage: dialogueActions.setNetworkDialogueStage,
};

export default connect(mapStateToProps, mapDispatchToProps)(Apply);
