import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createSelector, createStructuredSelector } from 'reselect';
import { FieldArray, reduxForm } from 'redux-form';

import CollapsibleCard from 'Components/CollapsibleCard';
import Field from './Field';
import RuleGroups from './RuleGroups';
import CreateRoleModal from './CreateRoleModal';
import formDescriptor from './formDescriptor';

class Form extends Component {
    static propTypes = {
        handleSubmit: PropTypes.func.isRequired,
        onSubmit: PropTypes.func.isRequired,
        initialValues: PropTypes.shape({}),
        selectedAuthProvider: PropTypes.shape({}),
        roles: PropTypes.arrayOf(
            PropTypes.shape({
                name: PropTypes.string,
                globalAccess: PropTypes.string
            })
        ).isRequired
    };

    static defaultProps = {
        initialValues: null,
        selectedAuthProvider: null
    };

    constructor(props) {
        super(props);
        this.state = {
            modalOpen: false
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

    renderRuleGroupsComponent = props => (
        <RuleGroups toggleModal={this.toggleModal} roles={this.props.roles} {...props} />
    );

    render() {
        const { handleSubmit, initialValues, onSubmit, roles } = this.props;
        const fields = formDescriptor[initialValues.type];
        if (!fields) return null;
        return (
            <>
                <form
                    className="w-full justify-between overflow-auto h-full"
                    onSubmit={handleSubmit(onSubmit)}
                    initialvalues={initialValues}
                >
                    <CollapsibleCard title="1. Configuration">
                        <div className="w-full h-full p-4">
                            {fields.map((field, index) => (
                                <Field key={index} {...field} />
                            ))}
                        </div>
                    </CollapsibleCard>
                    <div className="mt-4">
                        <CollapsibleCard
                            title={`2. Assign StackRox Roles to your (${
                                initialValues.type
                            }) attributes`}
                        >
                            <div className="w-full p-2">
                                <Field
                                    label={`Default role for "${initialValues.name}"`}
                                    type="select"
                                    jsonPath="defaultRole"
                                    options={roles}
                                />
                            </div>
                            <FieldArray
                                name="groups"
                                component={this.renderRuleGroupsComponent}
                                initialValues={initialValues}
                            />
                        </CollapsibleCard>
                    </div>
                </form>
                {this.renderCreateRoleModal()}
            </>
        );
    }
}

const getRoleOptions = createSelector([selectors.getRoles], roles =>
    roles.map(role => ({ value: role.name, label: role.name }))
);

const mapStateToProps = createStructuredSelector({
    roles: getRoleOptions
});

export default reduxForm({
    form: 'auth-provider-form'
})(
    connect(
        mapStateToProps,
        null
    )(Form)
);
