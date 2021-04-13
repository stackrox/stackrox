import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import * as Icon from 'react-feather';
import { selectors } from 'reducers';
import { actions as dialogueActions } from 'reducers/network/dialogue';

import dialogueStages from 'Containers/Network/Dialogue/dialogueStages';
import Button from './Button';

class Notify extends Component {
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
        notifiers: PropTypes.arrayOf(PropTypes.shape({})),
        setDialogueStage: PropTypes.func.isRequired,
    };

    static defaultProps = {
        notifiers: [],
    };

    onClick = () => {
        this.props.setDialogueStage(dialogueStages.notification);
    };

    render() {
        const { notifiers, modification } = this.props;
        const { noNotifiers } = notifiers.length === 0;

        const { applyYaml, toDelete } = modification;
        const noModification = applyYaml === '' && (!toDelete || toDelete.length === 0);
        return (
            <div className="ml-3">
                <Button
                    onClick={this.onClick}
                    disabled={noNotifiers || noModification}
                    icon={<Icon.Share2 className="h-4 w-4 mr-2" />}
                    text="Share YAML"
                />
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    modification: selectors.getNetworkPolicyModification,
    notifiers: selectors.getNotifiers,
});

const mapDispatchToProps = {
    setDialogueStage: dialogueActions.setNetworkDialogueStage,
};

export default connect(mapStateToProps, mapDispatchToProps)(Notify);
