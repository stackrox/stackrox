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

    renderRuleGroupsComponent = props => <RuleGroups toggleModal={this.toggleModal} {...props} />;

    render() {
        const { handleSubmit, initialValues, onSubmit, roles } = this.props;
        const fields = formDescriptor[initialValues.type];
        if (!fields) return null;
        return (
            <>
                <form
                    className="w-full justify-between overflow-auto h-full p-4"
                    onSubmit={handleSubmit(onSubmit)}
                    initialvalues={initialValues}
                >
                    <CollapsibleCard
                        title="1. Configuration"
                        titleClassName="border-b px-1 border-warning-300 leading-normal cursor-pointer flex justify-between items-center bg-warning-200 hover:border-warning-400"
                    >
                        <div className="w-full h-full px-4 py-3 pb-0">
                            {fields.map((field, index) => (
                                <Field key={index} {...field} />
                            ))}
                        </div>
                    </CollapsibleCard>
                    <div className="mt-4">
                        <CollapsibleCard
                            title={`2. Assign StackRox Roles to your ${initialValues.name} users`}
                            titleClassName="border-b px-1 border-warning-300 leading-normal cursor-pointer flex justify-between items-center bg-warning-200 hover:border-warning-400"
                        >
                            <div className="p-2">
                                <div className="w-full p-2">
                                    <Field
                                        label="Default Role"
                                        type="select"
                                        jsonPath="defaultRole"
                                        options={roles}
                                    />
                                    <p className="pb-2">
                                        The default role is granted when a user signs in with{' '}
                                        {initialValues.name}, but doesn&lsquo;t match any rules.
                                    </p>
                                    <p className="pb-2">
                                        To give users different roles, add rules.
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

const getRoleOptions = createSelector(
    [selectors.getRoles],
    roles => roles.map(role => ({ value: role.name, label: role.name }))
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
