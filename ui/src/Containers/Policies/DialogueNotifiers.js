import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as dialogueActions } from 'reducers/policies/notifier';

import CheckboxTable from 'Components/CheckboxTable';
import uniq from 'lodash/uniq';
import policyBulkActions from './policyBulkActions';

// DialogueNotifiers displays notifiers for policies selected from the table.
class DialogueNotifiers extends Component {
    static propTypes = {
        policiesAction: PropTypes.string.isRequired,
        policies: PropTypes.arrayOf(PropTypes.object).isRequired,
        selectedPolicyIds: PropTypes.arrayOf(PropTypes.string).isRequired,

        notifiers: PropTypes.arrayOf(
            PropTypes.shape({
                name: PropTypes.string.isRequired,
                id: PropTypes.string.isRequired,
            })
        ).isRequired,
        selectedNotifiers: PropTypes.arrayOf(PropTypes.string),
        setPolicyNotifiers: PropTypes.func.isRequired,
    };

    static defaultProps = {
        selectedNotifiers: [],
    };

    toggleRow = (id) => {
        const { selectedNotifiers } = this.props;
        if (selectedNotifiers.indexOf(id) > -1) {
            this.props.setPolicyNotifiers(
                selectedNotifiers.filter((notifierId) => notifierId !== id)
            );
        } else if (selectedNotifiers.length === 0) {
            this.props.setPolicyNotifiers([id]);
        } else {
            this.props.setPolicyNotifiers(selectedNotifiers.concat([id]));
        }
    };

    toggleSelectAll = () => {
        const { notifiers, selectedNotifiers } = this.props;
        if (notifiers.length > selectedNotifiers.length) {
            this.props.setPolicyNotifiers(notifiers.map((notifier) => notifier.id));
        } else {
            this.props.setPolicyNotifiers([]);
        }
    };

    selectedPolicyNotifiers = () => {
        const { notifiers } = this.props;
        if (this.props.policiesAction === policyBulkActions.enableNotification) {
            return notifiers;
        }

        const policyNotifiers = uniq(
            this.props.policies
                .filter(
                    (policy) =>
                        this.props.selectedPolicyIds.find((id) => id === policy.id) &&
                        policy.notifiers.length > 0
                )
                .flatMap((policy) => policy.notifiers)
        );

        return notifiers.filter((notifier) => policyNotifiers.find((o) => o === notifier.id));
    };

    render() {
        const notifiers = this.selectedPolicyNotifiers();

        if (notifiers.length < 1) {
            return null;
        }

        if (notifiers.length === 1) {
            return (
                <div className="p-4 border-b border-base-300 bg-base-100 text-base-900">
                    Selected Notifier: {notifiers[0].name}
                </div>
            );
        }

        const columns = [
            {
                accessor: 'name',
                Header: 'Select Notifiers',
            },
        ];

        const { selectedNotifiers } = this.props;
        return (
            <div>
                <CheckboxTable
                    rows={notifiers}
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
    policies: selectors.getFilteredPolicies,
    policiesAction: selectors.getPoliciesAction,
    selectedPolicyIds: selectors.getSelectedPolicyIds,

    notifiers: selectors.getNotifiers,
    selectedNotifiers: selectors.getPolicyNotifiers,
});

const mapDispatchToProps = {
    setPolicyNotifiers: dialogueActions.setPolicyNotifiers,
};

export default connect(mapStateToProps, mapDispatchToProps)(DialogueNotifiers);
