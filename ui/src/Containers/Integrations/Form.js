import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Form as ReactForm } from 'react-form';
import * as Icon from 'react-feather';

import formDescriptors from 'Containers/Integrations/formDescriptors';
import FormFields from 'Containers/Integrations/FormFields';
import Panel from 'Components/Panel';
import PanelButton from 'Components/PanelButton';

import * as AuthService from 'services/AuthService';
import { saveIntegration, testIntegration } from 'services/IntegrationsService';

class Form extends Component {
    static propTypes = {
        initialValues: PropTypes.shape({
            name: PropTypes.string.isRequired
        }),

        source: PropTypes.oneOf([
            'imageIntegrations',
            'dnrIntegrations',
            'notifiers',
            'authProviders',
            'clusters'
        ]).isRequired,
        type: PropTypes.string.isRequired,

        clusters: PropTypes.arrayOf(
            PropTypes.shape({
                name: PropTypes.string.isRequired,
                id: PropTypes.string.isRequired
            })
        ),

        onCancel: PropTypes.func.isRequired,
        onSubmitRequest: PropTypes.func.isRequired,
        onSubmitSuccess: PropTypes.func.isRequired,
        onSubmitError: PropTypes.func.isRequired,
        onTestRequest: PropTypes.func.isRequired,
        onTestSuccess: PropTypes.func.isRequired,
        onTestError: PropTypes.func.isRequired
    };

    static defaultProps = {
        initialValues: {},
        clusters: []
    };

    onTest = () => {
        this.props.onTestRequest();
        const data = this.addDefaultFormValues(this.formApi.values);
        testIntegration(this.props.source, data)
            .then(() => {
                this.props.onTestSuccess();
            })
            .catch(error => {
                this.props.onTestError(error.response.data.error);
            });
    };

    onSubmit = () => {
        this.props.onSubmitRequest();
        const data = this.addDefaultFormValues(this.formApi.values);
        const promise =
            this.props.source === 'authProviders'
                ? AuthService.saveAuthProvider(data)
                : saveIntegration(this.props.source, data);
        promise
            .then(() => {
                this.props.onSubmitSuccess();
            })
            .catch(error => {
                this.props.onSubmitError(error.response.data.error);
            });
    };

    getDNRntegrationFormFields = () => {
        const options = this.props.clusters.map(({ id, name }) => ({ value: id, label: name }));
        return [
            {
                label: 'Clusters',
                key: 'clusterIds',
                type: 'multiselect',
                options,
                placeholder: 'Choose clusters...'
            },
            {
                label: 'Portal URL',
                key: 'portalUrl',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Auth Token',
                key: 'authToken',
                type: 'text',
                placeholder: ''
            }
        ];
    };

    getFormFields = () => {
        if (this.props.source === 'dnrIntegrations') {
            return this.getDNRntegrationFormFields();
        }
        return formDescriptors[this.props.source][this.props.type];
    };

    // isEditMode returns true if the form is editing an existing entity
    // and false if it's creating a new entity.
    isEditMode = () => !!this.props.initialValues.name;

    addDefaultFormValues = formData => {
        const data = formData;
        const { location } = window;
        data.uiEndpoint = this.props.source === 'authProviders' ? location.host : location.origin;
        data.type = this.props.type;
        data.enabled = true;
        return data;
    };

    renderFormContent = formApi => {
        this.formApi = formApi;
        const fields = this.getFormFields();
        return (
            <form onSubmit={this.formApi.submitForm} className="w-full p-4">
                <div>
                    <FormFields formApi={this.formApi} fields={fields} />
                </div>
            </form>
        );
    };

    render() {
        const header = this.isEditMode() ? this.props.initialValues.name : 'New Integration';
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

        const key = this.isEditMode() ? this.props.initialValues.name : 'new-integration';

        return (
            <div className="flex flex-1">
                <Panel header={header} onClose={this.props.onCancel} buttons={buttons}>
                    <ReactForm
                        onSubmit={this.onSubmit}
                        validateSuccess={this.validateSuccess}
                        defaultValues={this.props.initialValues}
                        key={key}
                    >
                        {this.renderFormContent}
                    </ReactForm>
                </Panel>
            </div>
        );
    }
}

export default Form;
