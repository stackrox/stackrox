import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createSelector, createStructuredSelector } from 'reselect';
import { reduxForm, formValueSelector, change } from 'redux-form';
import flattenObject from 'utils/flattenObject';

import FormField from 'Components/FormField';
import CustomSelect from 'Components/Select';
import removeEmptyFields from 'utils/removeEmptyFields';
import { getPolicyFormDataKeys } from 'Containers/Policies/policyFormUtils';
import policyFormFields from 'Containers/Policies/policyCreationFormDescriptor';

import ReduxSelectField from 'Components/forms/ReduxSelectField';
import ReduxTextField from 'Components/forms/ReduxTextField';
import ReduxTextAreaField from 'Components/forms/ReduxTextAreaField';
import ReduxCheckboxField from 'Components/forms/ReduxCheckboxField';
import ReduxMultiSelectField from 'Components/forms/ReduxMultiSelectField';
import ReduxMultiSelectCreatableField from 'Components/forms/ReduxMultiSelectCreatableField';
import ReduxNumericInputField from 'Components/forms/ReduxNumericInputField';

class PolicyCreationForm extends Component {
    static propTypes = {
        policyFormFields: PropTypes.shape({}).isRequired,
        formData: PropTypes.shape({}).isRequired,
        change: PropTypes.func.isRequired
    };

    constructor(props) {
        super(props);

        this.state = {
            fields: []
        };
    }

    getReduxFormField = field => {
        switch (field.type) {
            case 'text':
                return (
                    <ReduxTextField
                        key={field.jsonpath}
                        name={field.jsonpath}
                        disabled={field.disabled}
                        placeholder={field.placeholder}
                    />
                );
            case 'checkbox':
                return <ReduxCheckboxField name={field.jsonpath} disabled={field.disabled} />;
            case 'select':
                return (
                    <ReduxSelectField
                        key={field.jsonpath}
                        name={field.jsonpath}
                        options={field.options}
                        placeholder={field.placeholder}
                        disabled={field.disabled}
                        defaultValue={field.default}
                    />
                );
            case 'multiselect':
                return <ReduxMultiSelectField name={field.jsonpath} options={field.options} />;
            case 'multiselect-creatable':
                return (
                    <ReduxMultiSelectCreatableField name={field.jsonpath} options={field.options} />
                );
            case 'textarea':
                return (
                    <ReduxTextAreaField
                        name={field.jsonpath}
                        disabled={field.disabled}
                        placeholder={field.placeholder}
                    />
                );
            case 'number':
                return (
                    <ReduxNumericInputField
                        key={field.jsonpath}
                        name={field.jsonpath}
                        min={field.min}
                        max={field.max}
                        step={field.step}
                        placeholder={field.placeholder}
                    />
                );
            case 'group':
                return field.jsonpaths.map(input => this.getReduxFormField(input));
            default:
                throw new Error(`Unknown field type: ${field.type}`);
        }
    };

    addFormField = jsonpath => {
        let fieldToAdd = {};
        Object.keys(this.props.policyFormFields).forEach(fieldGroup => {
            const field = this.props.policyFormFields[fieldGroup].descriptor.find(
                obj => obj.jsonpath === jsonpath
            );
            if (field) fieldToAdd = field;
        });
        this.setState(prevState => ({ fields: prevState.fields.concat(fieldToAdd.jsonpath) }));
    };

    removeField = jsonpath => {
        let fieldToRemove = {};
        Object.keys(this.props.policyFormFields).forEach(fieldGroup => {
            const field = this.props.policyFormFields[fieldGroup].descriptor.find(
                obj => obj.jsonpath === jsonpath
            );

            if (field) fieldToRemove = field;
        });
        this.setState(prevState => ({
            fields: prevState.fields.filter(fieldPath => fieldPath !== fieldToRemove.jsonpath)
        }));
        this.props.change(fieldToRemove.jsonpath, null);
    };

    renderFieldsDropdown = (formFields, formData) => {
        const availableFields = formFields.filter(
            field =>
                !this.state.fields.find(jsonpath => jsonpath === field.jsonpath) &&
                !field.default &&
                !formData.find(jsonpath => jsonpath.includes(field.jsonpath))
        );
        const placeholder = 'Add a field';
        if (!availableFields.length) return '';
        return (
            <div className="flex p-3 border-t border-base-200 bg-success-100">
                <span className="w-full">
                    <CustomSelect
                        className="border bg-base-100 border-success-500 text-success-600 p-3 pr-8 rounded cursor-pointer w-full font-400"
                        placeholder={placeholder}
                        options={availableFields}
                        value=""
                        onChange={this.addFormField}
                    />
                </span>
            </div>
        );
    };

    renderFields = (formFields, formData) => {
        const filteredFields = formFields.filter(field => {
            const isAddedField =
                this.state.fields.length !== 0 && this.state.fields.find(o => o === field.jsonpath);
            return (
                field.default ||
                isAddedField ||
                formData.find(jsonpath => jsonpath.includes(field.jsonpath))
            );
        });
        if (!filteredFields.length) {
            return <div className="p-3 text-base-500 font-500">No Fields Added</div>;
        }
        return (
            <div className="h-full p-3">
                {filteredFields.map(field => {
                    const removeField = !field.default ? this.removeField : null;
                    return (
                        <FormField
                            key={field.jsonpath}
                            label={field.label}
                            name={field.jsonpath}
                            required={field.required}
                            onRemove={removeField}
                        >
                            {this.getReduxFormField(field)}
                        </FormField>
                    );
                })}
            </div>
        );
    };

    renderFieldGroupCards = () => {
        const fieldGroups = Object.keys(this.props.policyFormFields);
        const formData = Object.keys(flattenObject(removeEmptyFields(this.props.formData)));
        return fieldGroups.map(fieldGroup => {
            const fieldGroupName = fieldGroup.replace(/([A-Z])/g, ' $1');
            const fields = this.props.policyFormFields[fieldGroup].descriptor;
            return (
                <div className="px-3 pt-5" data-test-id={fieldGroup} key={fieldGroup}>
                    <div className="bg-base-100 border border-base-200 shadow">
                        <div className="p-3 pb-2 border-b border-base-300 text-base-600 font-700 text-lg leading-normal capitalize">
                            {fieldGroupName}
                        </div>
                        {this.renderFields(fields, formData)}
                        {this.renderFieldsDropdown(fields, formData)}
                    </div>
                </div>
            );
        });
    };

    render() {
        return (
            <div className="flex flex-1 flex-col">
                <form id="dynamic-form" className="bg-base-200 pb-5">
                    {this.renderFieldGroupCards()}
                </form>
            </div>
        );
    }
}

const getPolicyFormFields = createSelector(
    [
        selectors.getNotifiers,
        selectors.getClusters,
        selectors.getDeployments,
        selectors.getPolicyCategories
    ],
    (notifiers, clusters, deployments, policyCategories) => {
        const { descriptor } = policyFormFields.policyDetails;
        const policyDetailsFormFields = descriptor.map(field => {
            const newField = Object.assign({}, field);
            let { options } = newField;
            switch (field.jsonpath) {
                case 'categories':
                    options = policyCategories.map(category => ({
                        label: category,
                        value: category
                    }));
                    break;
                case 'scope':
                    options = clusters.map(cluster => ({
                        label: cluster.name,
                        value: cluster.id
                    }));
                    break;
                case 'deployments':
                    options = deployments.map(deployment => ({
                        label: deployment.name,
                        value: deployment.name
                    }));
                    break;
                case 'notifiers':
                    options = notifiers.map(notifier => ({
                        label: notifier.name,
                        value: notifier.id
                    }));
                    break;
                default:
                    break;
            }
            newField.options = options;
            return newField;
        });
        policyFormFields.policyDetails.descriptor = policyDetailsFormFields;
        return policyFormFields;
    }
);

const formFields = state =>
    formValueSelector('policyCreationForm')(state, ...getPolicyFormDataKeys());
const getFormData = createSelector([formFields], formData => formData);

const mapStateToProps = createStructuredSelector({
    policyFormFields: getPolicyFormFields,
    formData: getFormData
});

const mapDispatchToProps = dispatch => ({
    change: (field, value) => dispatch(change('policyCreationForm', field, value))
});

export default reduxForm({ form: 'policyCreationForm' })(
    connect(
        mapStateToProps,
        mapDispatchToProps
    )(PolicyCreationForm)
);
