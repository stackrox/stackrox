import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createSelector, createStructuredSelector } from 'reselect';
import { FieldArray, reduxForm } from 'redux-form';

import Labeled from 'Components/Labeled';
import ReduxSelectField from 'Components/forms/ReduxSelectField';
import CollapsibleCard from 'Components/CollapsibleCard';
import { getAuthProviderLabelByValue } from 'constants/accessControl';
import RuleGroups from './RuleGroups';
import CreateRoleModal from './CreateRoleModal';
import ConfigurationFormFields from './ConfigurationFormFields';

class Form extends Component {
    static propTypes = {
        handleSubmit: PropTypes.func.isRequired,
        onSubmit: PropTypes.func.isRequired,
        initialValues: PropTypes.shape({
            type: PropTypes.string,
            active: PropTypes.bool,
        }).isRequired,
        change: PropTypes.func.isRequired,
        roles: PropTypes.arrayOf(
            PropTypes.shape({
                name: PropTypes.string,
                globalAccess: PropTypes.string,
            })
        ).isRequired,
    };

    constructor(props) {
        super(props);
        this.state = {
            modalOpen: false,
        };
    }

    toggleModal = () => {
        const { modalOpen } = this.state;
        this.setState({ modalOpen: !modalOpen });
    };

    renderCreateRoleModal = () => {
        const { modalOpen } = this.state;
        if (!modalOpen) return null;
        return <CreateRoleModal onClose={this.toggleModal} />;
    };

    renderRuleGroupsComponent = (props) => <RuleGroups toggleModal={this.toggleModal} {...props} />;

    render() {
        const { handleSubmit, onSubmit, initialValues, roles, change } = this.props;
        return (
            <>
                <form
                    className="w-full justify-between overflow-auto h-full p-4"
                    onSubmit={handleSubmit(onSubmit)}
                >
                    <CollapsibleCard
                        title="1. Configuration"
                        titleClassName="border-b px-1 border-warning-300 leading-normal cursor-pointer flex justify-between items-center bg-warning-200 hover:border-warning-400"
                        open={!initialValues.active}
                    >
                        <div className="w-full h-full px-4 py-3">
                            <ConfigurationFormFields
                                providerType={initialValues.type}
                                disabled={!!initialValues.active}
                                change={change}
                            />
                        </div>
                    </CollapsibleCard>
                    <div className="mt-4">
                        <CollapsibleCard
                            // Use the "type" here because the user usually hasn't typed a c yet.
                            title={`2. Assign StackRox roles to your ${getAuthProviderLabelByValue(
                                initialValues.type
                            )} users`}
                            titleClassName="border-b px-1 border-warning-300 leading-normal cursor-pointer flex justify-between items-center bg-warning-200 hover:border-warning-400"
                        >
                            <div className="p-2">
                                <div className="w-full p-2">
                                    <Labeled label="Minimum access role">
                                        <ReduxSelectField name="defaultRole" options={roles} />
                                    </Labeled>
                                    <p className="pb-2">
                                        The minimum access role is granted to all users who sign in
                                        with this authentication provider.
                                    </p>
                                    <p className="pb-2">
                                        To give users different roles, add rules. Users are granted
                                        all matching roles.
                                    </p>
                                    <p className="pb-2">
                                        Set the minimum access role to <em>None</em> if you want to
                                        define permissions completely using specific rules below.
                                    </p>
                                </div>
                                <FieldArray
                                    name="groups"
                                    component={this.renderRuleGroupsComponent}
                                    initialValues={initialValues}
                                />
                            </div>
                        </CollapsibleCard>
                    </div>
                </form>
                {this.renderCreateRoleModal()}
            </>
        );
    }
}

const getRoleOptions = createSelector([selectors.getRoles], (roles) =>
    roles.map((role) => ({ value: role.name, label: role.name }))
);

const mapStateToProps = createStructuredSelector({
    roles: getRoleOptions,
});

export default reduxForm({
    form: 'auth-provider-form',
})(connect(mapStateToProps, null)(Form));
