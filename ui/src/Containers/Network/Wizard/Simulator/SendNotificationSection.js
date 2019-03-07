import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { Link } from 'react-router-dom';
import { selectors } from 'reducers';
import { actions as dialogueActions } from 'reducers/network/dialogue';

import Select, { selectMenuOnTopStyles } from 'Components/ReactSelect';

class SendNotificationSection extends Component {
    static propTypes = {
        notifiers: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
        selectedNetworkNotifiers: PropTypes.arrayOf(PropTypes.string),
        setNetworkNotifierIds: PropTypes.func.isRequired,
        notifyNetworkPolicyModification: PropTypes.func.isRequired
    };

    static defaultProps = {
        selectedNetworkNotifiers: null
    };

    onClick = () => {
        this.props.notifyNetworkPolicyModification();
    };

    selectNotifier = selectedNotifierId => this.props.setNetworkNotifierIds([selectedNotifierId]);

    renderDropdown() {
        const { notifiers } = this.props;
        if (!notifiers.length) return null;

        // Selected notifiers is now an array so we can send multiple at once with the dialogue.
        // Until we use the dialogue, adapt the array to a single value.
        const { selectedNetworkNotifiers } = this.props;
        const selectedNotifier =
            selectedNetworkNotifiers && selectedNetworkNotifiers.length === 1
                ? selectedNetworkNotifiers[0]
                : null;

        return (
            <div>
                <span className="uppercase text-primary-500">Send network policy yaml to team</span>
                <div className="flex items-center mt-2">
                    <Select
                        options={notifiers}
                        placeholder="Select a notifier"
                        value={selectedNotifier}
                        onChange={this.selectNotifier}
                        className="w-3/4"
                        styles={selectMenuOnTopStyles}
                    />
                    <button
                        type="button"
                        className="p-3 ml-2 bg-primary-600 font-700 rounded-sm text-center text-base-100 w-1/4 h-9 hover:bg-primary-700"
                        onClick={this.onClick}
                        disabled={!selectedNotifier}
                    >
                        Send
                    </button>
                </div>
            </div>
        );
    }

    renderMessage() {
        if (this.props.notifiers.length) return null;
        return (
            <div className="flex items-center justify-between">
                <span className="text-primary-800">
                    There are currently no notifiers integrated.
                </span>
                <Link to="/main/integrations">
                    <button
                        type="button"
                        className="pl-3 pr-3 bg-primary-600 font-700 rounded-sm text-center text-base-100 h-9 hover:bg-primary-700"
                    >
                        Add Notification Integrations
                    </button>
                </Link>
            </div>
        );
    }

    render() {
        return (
            <div className="bg-primary-200 border-t-2 border-base-100 p-3">
                {this.renderMessage()}
                {this.renderDropdown()}
            </div>
        );
    }
}

const getFormattedNotifiers = createSelector(
    [selectors.getNotifiers],
    notifiers =>
        notifiers.map(notifier => ({
            label: notifier.name,
            value: notifier.id
        }))
);

const mapStateToProps = createStructuredSelector({
    notifiers: getFormattedNotifiers,
    selectedNetworkNotifiers: selectors.getNetworkNotifiers
});

const mapDispatchToProps = {
    setNetworkNotifierIds: dialogueActions.setNetworkNotifiers,
    notifyNetworkPolicyModification: dialogueActions.notifyNetworkPolicyModification
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(SendNotificationSection);
