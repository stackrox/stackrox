import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as dialogueActions } from 'reducers/network/dialogue';

import CheckboxTable from 'Components/CheckboxTable';

// ConfirmationDialogue is the pop-up that displays when deleting policies from the table.
class NotifyMany extends Component {
    static propTypes = {
        notifiers: PropTypes.arrayOf(
            PropTypes.shape({
                name: PropTypes.string.isRequired,
                id: PropTypes.string.isRequired
            })
        ).isRequired,
        selectedNotifiers: PropTypes.arrayOf(PropTypes.string),

        setNetworkNotifiers: PropTypes.func.isRequired
    };

    static defaultProps = {
        selectedNotifiers: []
    };

    toggleRow = id => {
        const { selectedNotifiers } = this.props;
        if (selectedNotifiers.indexOf(id) > -1) {
            this.props.setNetworkNotifiers(
                selectedNotifiers.filter(notifierId => notifierId !== id)
            );
        } else if (selectedNotifiers.length === 0) {
            this.props.setNetworkNotifiers([id]);
        } else {
            this.props.setNetworkNotifiers(selectedNotifiers.concat([id]));
        }
    };

    toggleSelectAll = () => {
        const { notifiers, selectedNotifiers } = this.props;
        if (notifiers.length > selectedNotifiers.length) {
            this.props.setNetworkNotifiers(notifiers.map(notifier => notifier.id));
        } else {
            this.props.setNetworkNotifiers([]);
        }
    };

    render() {
        if (this.props.notifiers.length <= 1) {
            return null;
        }
        const columns = [
            {
                accessor: 'name',
                Header: 'Select Notifiers'
            }
        ];

        const { selectedNotifiers } = this.props;
        return (
            <div>
                <CheckboxTable
                    rows={this.props.notifiers}
                    columns={columns}
                    selection={selectedNotifiers}
                    toggleRow={this.toggleRow}
                    toggleSelectAll={this.toggleSelectAll}
                />
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    notifiers: selectors.getNotifiers,
    selectedNotifiers: selectors.getNetworkNotifiers
});

const mapDispatchToProps = {
    setNetworkNotifiers: dialogueActions.setNetworkNotifiers
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(NotifyMany);
