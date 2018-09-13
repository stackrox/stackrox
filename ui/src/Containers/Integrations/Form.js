import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { actions } from 'reducers/integrations';
import { createSelector, createStructuredSelector } from 'reselect';
import { reduxForm, formValueSelector } from 'redux-form';
import * as Icon from 'react-feather';

import Panel from 'Components/Panel';
import PanelButton from 'Components/PanelButton';
import ReduxSelectField from 'Components/forms/ReduxSelectField';
import ReduxTextField from 'Components/forms/ReduxTextField';
import ReduxPasswordField from 'Components/forms/ReduxPasswordField';
import ReduxCheckboxField from 'Components/forms/ReduxCheckboxField';
import ReduxMultiSelectField from 'Components/forms/ReduxMultiSelectField';

import formDescriptors from 'Containers/Integrations/formDescriptors';

class Form extends Component {
    static propTypes = {
        initialValues: PropTypes.shape({
            name: PropTypes.string
        }),
        source: PropTypes.oneOf([
            'imageIntegrations',
            'dnrIntegrations',
            'notifiers',
            'authProviders',
            'clusters'
        ]).isRequired,
        type: PropTypes.string.isRequired,
        formFields: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
        formData: PropTypes.shape({
            name: PropTypes.string
        }).isRequired,
        onClose: PropTypes.func.isRequired,
        testIntegration: PropTypes.func.isRequired,
        saveIntegration: PropTypes.func.isRequired
    };

    static defaultProps = {
        initialValues: null
    };

    onTest = () => {
        const data = this.addDefaultFormValues();
        this.props.testIntegration(this.props.source, data);
    };

    onSubmit = () => {
        const { source, type } = this.props;
        const data = this.addDefaultFormValues();
        this.props.saveIntegration(source, type, data);
    };

    // isEditMode returns true if the form is editing an existing entity
    // and false if it's creating a new entity.
    isEditMode = () => (this.props.initialValues ? !!this.props.initialValues.name : false);

    addDefaultFormValues = () => {
        const { initialValues, formData } = this.props;
        const data = Object.assign({}, initialValues, formData);
        const { location } = window;
        data.uiEndpoint = this.props.source === 'authProviders' ? location.host : location.origin;
        data.type = this.props.type;
        data.enabled = true;
        data.categories = data.categories ? data.categories.map(category => category.value) : [];
        data.clusterIds = data.clusterIds ? data.clusterIds.map(id => id.value) : [];
        return data;
    };

    renderFormField = field => {
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
                return (
                    <ReduxCheckboxField
                        name={field.jsonpath}
                        disabled={field.disabled}
                        placeholder={field.placeholder}
                    />
                );
            case 'select':
                return (
                    <ReduxSelectField
                        key={field.jsonpath}
                        name={field.jsonpath}
                        options={field.options}
                    />
                );

            case 'password':
                return (
                    <ReduxPasswordField
                        name={field.jsonpath}
                        key={field.jsonpath}
                        placeholder={field.placeholder}
                    />
                );
            case 'multiselect':
                return <ReduxMultiSelectField name={field.jsonpath} options={field.options} />;
            default:
                throw new Error(`Unknown field type: ${field.type}`);
        }
    };

    renderForm = () => {
        const { formFields } = this.props;
        return (
            <form id="integrations-form" className="w-full p-4">
                <div>
                    {formFields.map(field => (
                        <label className="flex mt-4" htmlFor={field.key} key={field.label}>
                            <div className="mr-4 flex items-center w-2/3 capitalize">
                                {field.label}
                            </div>
                            {this.renderFormField(field)}
                        </label>
                    ))}
                </div>
            </form>
        );
    };

    render() {
        const header = this.isEditMode() ? this.props.formData.name : 'New Integration';
        const buttons = (
            <React.Fragment>
                <PanelButton
                    icon={<Icon.Save className="h-4 w-4" />}
                    text={this.isEditMode() ? 'Save' : 'Create'}
                    className="btn btn-success"
                    onClick={this.onSubmit}
                />
                {this.props.source !== 'authProviders' && (
                    <PanelButton
                        icon={<Icon.Check className="h-4 w-4" />}
                        text="Test"
                        className="btn btn-primary"
                        onClick={this.onTest}
                    />
                )}
            </React.Fragment>
        );

        return (
            <div className="flex flex-1">
                <Panel header={header} onClose={this.props.onClose} buttons={buttons}>
                    {this.renderForm()}
                </Panel>
            </div>
        );
    }
}

const getFormFields = createSelector(
    [selectors.getClusters, (state, props) => props],
    (clusters, props) => {
        if (props.source === 'dnrIntegrations') {
            const options = clusters.map(({ id, name }) => ({ value: id, label: name }));
            return [
                {
                    label: 'Clusters',
                    jsonpath: 'clusterIds',
                    type: 'multiselect',
                    options,
                    placeholder: 'Choose clusters...'
                },
                {
                    label: 'Portal URL',
                    jsonpath: 'portalUrl',
                    type: 'text',
                    placeholder: ''
                },
                {
                    label: 'Auth Token',
                    jsonpath: 'authToken',
                    type: 'text',
                    placeholder: ''
                }
            ];
        }
        return formDescriptors[props.source][props.type];
    }
);
const getFormFieldKeys = (source, type) => {
    if (source === 'dnrIntegrations') return ['clusterIds', 'portalUrl', 'authToken'];
    return formDescriptors[source] ? formDescriptors[source][type].map(obj => obj.jsonpath) : '';
};

const formFieldKeys = (state, props) =>
    formValueSelector('integrationForm')(state, ...getFormFieldKeys(props.source, props.type));
const getFormData = createSelector([formFieldKeys], formData => formData);

const mapStateToProps = createStructuredSelector({
    formFields: getFormFields,
    formData: getFormData
});

const mapDispatchToProps = dispatch => ({
    saveIntegration: (source, sourceType, integration) =>
        dispatch(actions.saveIntegration.request({ source, sourceType, integration })),
    testIntegration: (source, integration) => dispatch(actions.testIntegration(source, integration))
});

export default reduxForm({ form: 'integrationForm' })(
    connect(mapStateToProps, mapDispatchToProps)(Form)
);
