import React from 'react';
import PropTypes from 'prop-types';
import { FieldArray, reduxForm } from 'redux-form';

import * as Icon from 'react-feather';
import CollapsibleCard from 'Components/CollapsibleCard';
import Field from './Field';
import formDescriptor from './formDescriptor';

const Groups = ({ fields, onDelete, id }) => {
    const { keyOptions, roleOptions } = formDescriptor.attrToRole;
    const addRule = () => fields.push({ props: { auth_provider_id: id } });
    const deleteRule = (group, idx) => () => {
        onDelete(group.get(idx));
        fields.remove(idx);
    };
    return (
        <div className="w-full p-2">
            {fields.map((group, idx, value) => (
                <div className="flex flex-row" key={idx}>
                    <div className="w-full">
                        <Field
                            jsonPath={`${group}.props.key`}
                            type="select"
                            label="Key"
                            options={keyOptions}
                        />
                    </div>
                    <div className="w-full">
                        <Field jsonPath={`${group}.props.value`} type="text" label="Value" />
                    </div>
                    <div className="flex items-center">
                        <Icon.ArrowRight className="h-4 w-4" />
                    </div>
                    <div className="w-full">
                        <Field
                            jsonPath={`${group}.roleName`}
                            type="select"
                            label="Role"
                            options={roleOptions}
                        />
                    </div>
                    <button className="pl-2 pr-2" type="button">
                        <Icon.Plus className="h-4 w-4" />
                    </button>
                    <button className="pl-2 pr-2" type="button">
                        <Icon.Trash2 className="h-4 w-4" onClick={deleteRule(value, idx)} />
                    </button>
                </div>
            ))}
            {/* eslint-disable-next-line */}
            <button className="btn btn-primary" type="button" onClick={addRule}>
                Add New Rule
            </button>
        </div>
    );
};

const Form = props => {
    const { handleSubmit, initialValues, onSubmit, onDelete } = props;
    const fields = formDescriptor[initialValues.type];
    if (!fields) return null;
    return (
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
                    title={`2. Assign StackRox Roles to your (${initialValues.type}) attributes`}
                >
                    <FieldArray
                        name="groups"
                        component={Groups}
                        onDelete={onDelete}
                        id={initialValues.id}
                    />
                </CollapsibleCard>
            </div>
        </form>
    );
};

Groups.propTypes = {
    fields: PropTypes.shape({}).isRequired,
    onDelete: PropTypes.func.isRequired,
    id: PropTypes.string
};

Groups.defaultProps = {
    id: ''
};

Form.propTypes = {
    handleSubmit: PropTypes.func.isRequired,
    onSubmit: PropTypes.func.isRequired,
    onDelete: PropTypes.func.isRequired,
    initialValues: PropTypes.shape({})
};

Form.defaultProps = {
    initialValues: null
};

export default reduxForm({
    // a unique name for the form
    form: 'auth-provider-form'
})(Form);
