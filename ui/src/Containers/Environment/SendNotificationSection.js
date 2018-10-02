import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { actions } from 'reducers/environment';
import { selectors } from 'reducers';
import { createSelector, createStructuredSelector } from 'reselect';
import { Link } from 'react-router-dom';

import Select from 'Components/ReactSelect';

const selectMenuOnTopStyles = {
    menu: base => ({
        ...base,
        position: 'absolute !important',
        top: 'auto !important',
        bottom: '100% !important',
        'border-radius': '5px 5px 0px 0px !important'
    })
};

class SendNotificationSection extends Component {
    static propTypes = {
        notifiers: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
        sendYAMLNotification: PropTypes.func.isRequired
    };

    state = {
        selectedNotifierId: null
    };

    onClick = () => {
        this.props.sendYAMLNotification(this.state.selectedNotifierId);
    };

    selectNotifier = selectedNotifierId => this.setState({ selectedNotifierId });

    renderDropdown() {
        const { notifiers } = this.props;
        const { selectedNotifierId } = this.state;
        if (!notifiers.length) return null;
        return (
            <div>
                <span className="uppercase text-primary-500">Send network policy yaml to team</span>
                <div className="flex items-center mt-2">
                    <Select
                        options={notifiers}
                        placeholder="Select a notifier"
                        value={selectedNotifierId}
                        onChange={this.selectNotifier}
                        className="w-3/4"
                        styles={selectMenuOnTopStyles}
                    />
                    <button
                        type="button"
                        className="p-3 ml-2 bg-primary-500 rounded-sm text-center text-base-100 w-1/4 h-9 hover:bg-primary-400 hover:text-base-100"
                        onClick={this.onClick}
                        disabled={!selectedNotifierId}
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
                <span>There are no notifiers integrated.</span>
                <Link to="/main/integrations">
                    <button
                        type="button"
                        className="pl-3 pr-3 bg-primary-500 rounded-sm text-center text-base-100 h-9 hover:bg-primary-600 hover:text-base-100"
                    >
                        Add Notification Integrations
                    </button>
                </Link>
            </div>
        );
    }

    render() {
        return (
            <div className="bg-base-200 border-t-2 border-base-100 p-3">
                {this.renderMessage()}
                {this.renderDropdown()}
            </div>
        );
    }
}

const getFormattedNotifiers = createSelector([selectors.getNotifiers], notifiers =>
    notifiers.map(notifier => ({
        label: notifier.name,
        value: notifier.id
    }))
);

const mapStateToProps = createStructuredSelector({
    notifiers: getFormattedNotifiers
});

const mapDispatchToProps = {
    sendYAMLNotification: actions.sendYAMLNotification
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(SendNotificationSection);
