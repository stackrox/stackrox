import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createSelector, createStructuredSelector } from 'reselect';
import { reduxForm, formValueSelector, change } from 'redux-form';

import flattenObject from 'utils/flattenObject';
import removeEmptyFields from 'utils/removeEmptyFields';
import { getPolicyFormDataKeys } from 'Containers/Policies/Wizard/Form/utils';
import policyFormFields from 'Containers/Policies/Wizard/Form/descriptors';

import FormField from 'Components/FormField';
import CustomSelect from 'Components/Select';
import Field from 'Containers/Policies/Wizard/Form/Field';

class FieldGroupCards extends Component {
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

    addFormField = option => {
        let fieldToAdd = {};
        Object.keys(this.props.policyFormFields).forEach(fieldGroup => {
            const field = this.props.policyFormFields[fieldGroup].descriptor.find(
                obj => obj.jsonpath === option.value
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

    renderFields = (formFields, formData) => {
        const filteredFields = formFields.filter(field => {
            const isAddedField =
                this.state.fields.length !== 0 && this.state.fields.find(o => o === field.jsonpath);
            return (
                !field.header &&
                (field.default ||
                    isAddedField ||
                    formData.find(jsonpath => jsonpath.includes(field.jsonpath)))
            );
        });

        if (!filteredFields.length) {
            if (this.isHeaderOnlyCard(formFields)) return '';

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
                            <Field field={field} />
                        </FormField>
                    );
                })}
            </div>
        );
    };

    renderFieldsDropdown = (formFields, formData) => {
        const availableFields = formFields
            .filter(
                field =>
                    !this.state.fields.find(jsonpath => jsonpath === field.jsonpath) &&
                    !field.default &&
                    !formData.find(jsonpath => jsonpath.includes(field.jsonpath))
            )
            .map(field => ({ label: field.label, value: field.jsonpath }));
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

    renderHeaderControl = formFields => {
        const headerField = formFields.find(field => field.header);
        if (!headerField) return '';

        return (
            <div className="header-control float-right flex">
                <span className="pr-1">{headerField.label}</span>
                <Field field={headerField} />
            </div>
        );
    };

    isHeaderOnlyCard = formFields =>
        formFields.length === 1 && formFields.find(field => field.header);

    render() {
        const fieldGroups = this.props.policyFormFields;
        const fieldGroupKeys = Object.keys(fieldGroups);
        const formData = Object.keys(flattenObject(removeEmptyFields(this.props.formData)));

        return fieldGroupKeys.map(fieldGroupKey => {
            const fieldGroupName = fieldGroups[fieldGroupKey].header;
            const formFields = fieldGroups[fieldGroupKey].descriptor;
            const headerControl = this.renderHeaderControl(formFields);
            const border = this.isHeaderOnlyCard(formFields) ? '' : 'border';
            const leading = headerControl ? 'leading-loose' : 'leading-normal';
            return (
                <div className="px-3 pt-5" data-test-id={fieldGroupKey} key={fieldGroupKey}>
                    <div className={`bg-base-100 ${border} border-base-200 shadow`}>
                        <div
                            className={`p-2 pb-2 border-b border-base-300 text-base-600 font-700 text-lg capitalize flex justify-between ${leading}`}
                        >
                            {fieldGroupName}
                            {headerControl}
                        </div>
                        {this.renderFields(formFields, formData)}
                        {this.renderFieldsDropdown(formFields, formData)}
                    </div>
                </div>
            );
        });
    }
}

const getPolicyFormFields = createSelector(
    [
        selectors.getNotifiers,
        selectors.getClusters,
        selectors.getDeployments,
        selectors.getImages,
        selectors.getPolicyCategories
    ],
    (notifiers, clusters, deployments, images, policyCategories) => {
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
                case 'deployments':
                    options = deployments.map(deployment => ({
                        label: deployment.name,
                        value: deployment.name
                    }));
                    break;
                case 'images':
                    options = images.map(image => ({
                        label: image.name,
                        value: image.name
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

const getFormData = createSelector(
    [formFields],
    formData => formData
);

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
    )(FieldGroupCards)
);
