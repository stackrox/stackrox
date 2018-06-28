import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Form as ReactForm } from 'react-form';
import * as Icon from 'react-feather';

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

        source: PropTypes.oneOf(['imageIntegrations', 'notifiers', 'authProviders', 'clusters'])
            .isRequired,
        type: PropTypes.string.isRequired,

        onCancel: PropTypes.func.isRequired,
        onSubmitRequest: PropTypes.func.isRequired,
        onSubmitSuccess: PropTypes.func.isRequired,
        onSubmitError: PropTypes.func.isRequired,
        onTestRequest: PropTypes.func.isRequired,
        onTestSuccess: PropTypes.func.isRequired,
        onTestError: PropTypes.func.isRequired
    };

    static defaultProps = {
        initialValues: null
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
        return (
            <form onSubmit={this.formApi.submitForm} className="w-full p-4">
                <div>
                    <FormFields
                        formApi={this.formApi}
                        source={this.props.source}
                        type={this.props.type}
                    />
                </div>
            </form>
        );
    };

    render() {
        if (!this.props.initialValues) return '';

        const header = this.props.initialValues.name || 'New Integration';
        const buttons = (
            <React.Fragment>
                <PanelButton
                    icon={<Icon.Save className="h-4 w-4" />}
                    text={this.props.initialValues.name ? 'Save' : 'Create'}
                    className="btn-success"
                    onClick={this.onSubmit}
                />
                {this.props.source !== 'authProviders' && (
                    <PanelButton
                        icon={<Icon.Check className="h-4 w-4" />}
                        text="Test"
                        className="btn-primary"
                        onClick={this.onTest}
                    />
                )}
            </React.Fragment>
        );

        const key = this.props.initialValues ? this.props.initialValues.name : 'new-integration';

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
