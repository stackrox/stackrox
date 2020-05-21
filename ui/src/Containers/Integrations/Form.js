import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { actions } from 'reducers/integrations';
import { createSelector, createStructuredSelector } from 'reselect';
import { reduxForm, formValueSelector, FieldArray } from 'redux-form';
import * as Icon from 'react-feather';
import set from 'lodash/set';

import Panel from 'Components/Panel';
import PanelButton from 'Components/PanelButton';
import ReduxSelectField from 'Components/forms/ReduxSelectField';
import ReduxTextField from 'Components/forms/ReduxTextField';
import ReduxTextAreaField from 'Components/forms/ReduxTextAreaField';
import ReduxPasswordField from 'Components/forms/ReduxPasswordField';
import ReduxToggleField from 'Components/forms/ReduxToggleField';
import ReduxMultiSelectField from 'Components/forms/ReduxMultiSelectField';
import ReduxNumericInputField from 'Components/forms/ReduxNumericInputField';
import HelpIcon from 'Components/forms/HelpIcon';
import formDescriptors from 'Containers/Integrations/formDescriptors';
import { setFormSubmissionOptions } from './integrationFormUtils';
import Schedule from './Schedule';

class Form extends Component {
    static propTypes = {
        initialValues: PropTypes.shape({
            id: PropTypes.string,
            name: PropTypes.string,
        }),
        isNewIntegration: PropTypes.bool.isRequired,
        source: PropTypes.oneOf([
            'imageIntegrations',
            'notifiers',
            'authProviders',
            'clusters',
            'backups',
            'authPlugins',
        ]).isRequired,
        type: PropTypes.string.isRequired,
        formFields: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
        formData: PropTypes.shape({
            name: PropTypes.string,
        }).isRequired,
        onClose: PropTypes.func.isRequired,
        testIntegration: PropTypes.func.isRequired,
        saveIntegration: PropTypes.func.isRequired,
        triggerBackup: PropTypes.func.isRequired,
    };

    static defaultProps = {
        initialValues: null,
    };

    onTest = () => {
        const { source, type, isNewIntegration } = this.props;
        const data = this.addDefaultFormValues();
        const options = setFormSubmissionOptions(source, type, data, { isNewIntegration });
        this.props.testIntegration(source, data, options);
    };

    onBackup = () => {
        this.props.triggerBackup(this.props.initialValues.id);
    };

    onSubmit = () => {
        const { source, type, isNewIntegration } = this.props;
        const data = this.addDefaultFormValues();
        const options = setFormSubmissionOptions(source, type, data, { isNewIntegration });
        this.props.saveIntegration(source, type, data, options);
        this.props.onClose();
    };

    // isEditMode returns true if the form is editing an existing entity
    // and false if it's creating a new entity.
    isEditMode = () => (this.props.initialValues ? !!this.props.initialValues.name : false);

    addDefaultFormValues = () => {
        const { initialValues, formData } = this.props;
        const data = { ...initialValues, ...formData };
        const { location } = window;
        data.uiEndpoint = this.props.source === 'authProviders' ? location.host : location.origin;
        data.type = this.props.type;
        // Set a default value of true for everything but auth plugins (they have their own toggle for that).
        if (this.props.source !== 'authPlugins') {
            data.enabled = true;
        }
        data.categories = data.categories || [];
        data.clusterIds = data.clusterIds || [];
        return data;
    };

    renderHiddenField = (field) => (
        <input type="hidden" name={field.jsonpath} value={field.value} />
    );

    renderFormField = (field, initialValues) => {
        const disabled = field.disabled || (this.isEditMode() && field.immutable);
        const placeholder = field.placeholderFunction
            ? field.placeholderFunction(initialValues)
            : field.placeholder;
        switch (field.type) {
            case 'text':
                return (
                    <ReduxTextField
                        key={field.jsonpath}
                        name={field.jsonpath}
                        disabled={disabled}
                        placeholder={placeholder}
                        value={field.default}
                    />
                );
            case 'textarea':
                return (
                    <ReduxTextAreaField
                        key={field.jsonpath}
                        name={field.jsonpath}
                        disabled={disabled}
                        placeholder={placeholder}
                        value={field.default}
                    />
                );
            case 'toggle':
                return (
                    <ReduxToggleField
                        name={field.jsonpath}
                        disabled={disabled}
                        placeholder={placeholder}
                        value={field.default}
                    />
                );
            case 'select':
                return (
                    <ReduxSelectField
                        key={field.jsonpath}
                        name={field.jsonpath}
                        options={field.options}
                        disabled={disabled}
                        value={field.default}
                    />
                );

            case 'password':
                return (
                    <ReduxPasswordField
                        name={field.jsonpath}
                        key={field.jsonpath}
                        placeholder={placeholder}
                        disabled={disabled}
                    />
                );
            case 'multiselect':
                return (
                    <ReduxMultiSelectField
                        name={field.jsonpath}
                        options={field.options}
                        disabled={disabled}
                    />
                );
            case 'number':
                return (
                    <ReduxNumericInputField
                        name={field.jsonpath}
                        options={field.options}
                        min={0}
                        disabled={disabled}
                    />
                );
            case 'list':
                return <FieldArray name={field.jsonpath} component={field.listRender} />;
            case 'schedule':
                return <Schedule data={field.jsonpath} />;
            default:
                throw new Error(`Unknown field type: ${field.type}`);
        }
    };

    renderForm = () => {
        const { formFields, initialValues } = this.props;
        return (
            <form id="integrations-form" className="w-full p-4">
                <div>
                    {formFields
                        .filter((field) => {
                            return field.hiddenFunction
                                ? !field.hiddenFunction(initialValues)
                                : !field.hidden;
                        })
                        .map((field) => {
                            if (field.type === 'html') {
                                return field.html;
                            }
                            const width = field.type === 'toggle' ? 'w-full' : 'w-2/3';
                            const align = field.type !== 'list' ? 'items-center' : 'pt-2';
                            const helpIconDescription = field.helpFunction
                                ? field.helpFunction(initialValues)
                                : field.help;
                            return (
                                // eslint-disable-next-line jsx-a11y/label-has-for
                                <div className="flex mt-4" htmlFor={field.key} key={field.label}>
                                    <div className={`mr-4 flex ${width} capitalize ${align}`}>
                                        {field.label}
                                        {helpIconDescription && (
                                            <div className="ml-2">
                                                <HelpIcon description={helpIconDescription} />
                                            </div>
                                        )}
                                    </div>
                                    {this.renderFormField(field, initialValues)}
                                </div>
                            );
                        })}
                    {formFields
                        .filter((field) => {
                            return field.hiddenFunction
                                ? field.hiddenFunction(initialValues)
                                : field.hidden;
                        })
                        .map(this.renderHiddenField)}
                </div>
            </form>
        );
    };

    render() {
        const header = this.isEditMode() ? this.props.formData.name : 'New Integration';
        const buttons = (
            <>
                <PanelButton
                    icon={<Icon.Save className="h-4 w-4" />}
                    className="btn btn-success mx-1"
                    onClick={this.onSubmit}
                    tooltip={this.isEditMode() ? 'Save' : 'Create'}
                >
                    {this.isEditMode() ? 'Save' : 'Create'}
                </PanelButton>
                {this.props.source === 'backups' &&
                    this.props.initialValues &&
                    this.props.initialValues.id && (
                        <PanelButton
                            icon={<Icon.Check className="h-4 w-4" />}
                            className="btn btn-base mx-1"
                            onClick={this.onBackup}
                            tooltip="Trigger Backup"
                        >
                            Trigger Backup
                        </PanelButton>
                    )}
                {this.props.source !== 'authProviders' && (
                    <PanelButton
                        icon={<Icon.Check className="h-4 w-4" />}
                        className="btn btn-base mx-1"
                        onClick={this.onTest}
                        tooltip="Test"
                    >
                        Test
                    </PanelButton>
                )}
            </>
        );

        return (
            <div className="flex flex-1">
                <Panel header={header} onClose={this.props.onClose} headerComponents={buttons}>
                    {this.renderForm()}
                </Panel>
            </div>
        );
    }
}

const getFormFields = createSelector(
    [selectors.getClusters, (state, props) => props],
    (clusters, props) => formDescriptors[props.source][props.type]
);

const getFormFieldKeys = (source, type) =>
    formDescriptors[source] ? formDescriptors[source][type].map((obj) => obj.jsonpath) : '';

const getFormDefaultValues = (source, type) => {
    const defaultValues = {};
    if (formDescriptors[source] && formDescriptors[source][type]) {
        formDescriptors[source][type].forEach((field) => {
            if (field.default) {
                set(defaultValues, field.jsonpath, field.default);
            }
        });
    }
    return defaultValues;
};

const formFieldKeys = (state, props) => {
    const values = formValueSelector('integrationForm')(
        state,
        ...getFormFieldKeys(props.source, props.type)
    );
    const defaultValues = getFormDefaultValues(props.source, props.type);
    const initialValues = { ...defaultValues, ...values };
    return initialValues;
};

const getFormData = createSelector([formFieldKeys], (formData) => formData);

const mapStateToProps = createStructuredSelector({
    formFields: getFormFields,
    formData: getFormData,
});

const mapDispatchToProps = (dispatch) => ({
    saveIntegration: (source, sourceType, integration, options) =>
        dispatch(actions.saveIntegration.request({ source, sourceType, integration, options })),
    testIntegration: (source, integration, options) =>
        dispatch(actions.testIntegration(source, integration, options)),
    triggerBackup: (source, id) => dispatch(actions.triggerBackup(source, id)),
});

export default reduxForm({ form: 'integrationForm' })(
    connect(mapStateToProps, mapDispatchToProps)(Form)
);
