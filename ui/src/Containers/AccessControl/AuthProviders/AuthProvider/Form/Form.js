import React, { Component } from 'react';
import PropTypes from 'prop-types';
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
        onDelete: PropTypes.func.isRequired,
        initialValues: PropTypes.shape({})
    };

    static defaultProps = {
        initialValues: null
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
        const { handleSubmit, initialValues, onSubmit, onDelete } = this.props;
        const fields = formDescriptor[initialValues.type];
        if (!fields) return null;
        return (
            <>
                <form
                    className="w-full justify-between overflow-auto"
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
                            <FieldArray
                                name="groups"
                                component={this.renderRuleGroupsComponent}
                                onDelete={onDelete}
                                id={initialValues.id}
                            />
                        </CollapsibleCard>
                    </div>
                </form>
                {this.renderCreateRoleModal()}
            </>
        );
    }
}

export default reduxForm({
    // a unique name for the form
    form: 'auth-provider-form'
})(Form);
