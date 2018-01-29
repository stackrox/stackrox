import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Modal from 'Components/Modal';
import { Form, Text, Select } from 'react-form';

import axios from 'axios';
import * as Icon from 'react-feather';

const sourceMap = {
    notifiers: {
        jira: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Jira Integration'
            },
            {
                label: 'Username',
                key: 'config.username',
                type: 'text',
                placeholder: 'user@example.com'
            },
            {
                label: 'Password',
                key: 'config.password',
                type: 'password',
                placeholder: ''
            },
            {
                label: 'Project Key',
                key: 'config.project',
                type: 'text',
                placeholder: 'PROJ'
            },
            {
                label: 'Issue Type',
                key: 'config.issue_type',
                type: 'text',
                placeholder: 'Task, Sub-task, Story, Bug, or Epic'
            },
            {
                label: 'Jira URL',
                key: 'config.url',
                type: 'text',
                placeholder: 'https://example.atlassian.net'
            }
        ],
        email: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Email Integration'
            },
            {
                label: 'Email Server',
                key: 'config.server',
                type: 'text',
                placeholder: 'smtp.example.com:465'
            },
            {
                label: 'Username',
                key: 'config.username',
                type: 'text',
                placeholder: 'postmaster@example.com'
            },
            {
                label: 'Password',
                key: 'config.password',
                type: 'password'
            },
            {
                label: 'Sender',
                key: 'config.sender',
                type: 'text',
                placeholder: 'mitigate-notifier@example.com'
            },
            {
                label: 'Recipient',
                key: 'config.recipient',
                type: 'text',
                placeholder: 'mitigate-alerts@example.com'
            },
            {
                label: 'TLS',
                key: 'config.tls',
                type: 'select',
                options: [
                    { label: 'On', value: 'true' },
                    { label: 'Off', value: 'false' }
                ],
                placeholder: 'Enable TLS?'
            }
        ],
        slack: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Slack Integration'
            },
            {
                label: 'Slack Webhook',
                key: 'config.webhook',
                type: 'text',
                placeholder: 'https://hooks.slack.com/services/EXAMPLE'
            },
            {
                label: 'Slack Channel',
                key: 'config.channel',
                type: 'text',
                placeholder: '#slack-channel'
            }
        ]
    },
    scanners: {
        tenable: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Tenable Scanner'
            },
            {
                label: 'Scanner Endpoint',
                key: 'endpoint',
                type: 'text',
                placeholder: 'https://cloud.tenable.com'
            },
            {
                label: 'Remote Endpoint',
                key: 'remote',
                type: 'text',
                placeholder: 'registry.cloud.tenable.com'
            },
            {
                label: 'Access Key',
                key: 'config.accessKey',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Secret Key',
                key: 'config.secretKey',
                type: 'text',
                placeholder: ''
            }
        ],
        dtr: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'DTR Scanner'
            },
            {
                label: 'Scanner Endpoint',
                key: 'endpoint',
                type: 'text',
                placeholder: 'example-dtr.rox.systems'
            },
            {
                label: 'Remote Endpoint',
                key: 'remote',
                type: 'text',
                placeholder: 'example-dtr.rox.systems'
            },
            {
                label: 'Docker Username',
                key: 'config.username',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Docker Password',
                key: 'config.password',
                type: 'password',
                placeholder: ''
            }
        ],
    },
    registries: {
        docker: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Docker Registry'
            },
            {
                label: 'Scanner Endpoint',
                key: 'endpoint',
                type: 'text',
                placeholder: 'registry-1.docker.io'
            },
            {
                label: 'Remote Endpoint',
                key: 'remote',
                type: 'text',
                placeholder: 'docker.io'
            },
            {
                label: 'Docker Username',
                key: 'config.username',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Docker Password',
                key: 'config.password',
                type: 'password',
                placeholder: ''
            }
        ],
        tenable: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Tenable Registry'
            },
            {
                label: 'Scanner Endpoint',
                key: 'endpoint',
                type: 'text',
                placeholder: 'registry.cloud.tenable.com'
            },
            {
                label: 'Remote Endpoint',
                key: 'remote',
                type: 'text',
                placeholder: 'registry.cloud.tenable.com'
            },
            {
                label: 'Access Key',
                key: 'config.accessKey',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Secret Key',
                key: 'config.secretKey',
                type: 'text',
                placeholder: ''
            }
        ],
        dtr: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'DTR Registry'
            },
            {
                label: 'Scanner Endpoint',
                key: 'endpoint',
                type: 'text',
                placeholder: 'example-dtr.rox.systems'
            },
            {
                label: 'Remote Endpoint',
                key: 'remote',
                type: 'text',
                placeholder: 'example-dtr.rox.systems'
            },
            {
                label: 'Docker Username',
                key: 'config.username',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Docker Password',
                key: 'config.password',
                type: 'password',
                placeholder: ''
            }
        ],
    }
};

const SOURCE_LABELS = Object.freeze({
    registries: 'registry',
    scanners: 'scanner',
    notifiers: 'plugin'
});

const api = {
    registries: data => (data.id !== '' ? axios.put(`/v1/registries/${data.id}`, data) : axios.post('/v1/registries', data)),
    scanners: data => (data.id !== '' ? axios.put(`/v1/scanners/${data.id}`, data) : axios.post('/v1/scanners', data)),
    notifiers: data => (data.id !== '' ? axios.put(`/v1/notifiers/${data.id}`, data) : axios.post('/v1/notifiers', data))
};

const reducer = (action, prevState, nextState) => {
    switch (action) {
        case 'ERROR_MESSAGE':
            return { message: nextState.message };
        case 'CLEAR_ERROR_MESSAGE':
            return { message: '' };
        default:
            return prevState;
    }
};

class IntegrationModal extends Component {
    static propTypes = {
        integration: PropTypes.shape({
            type: PropTypes.string.isRequired
        }).isRequired,
        source: PropTypes.oneOf(['registries', 'scanners', 'notifiers']).isRequired,
        onRequestClose: PropTypes.func.isRequired
    }

    constructor(props) {
        super(props);

        this.state = {
            message: ''
        };
    }

    onRequestClose = (isSuccessful) => {
        this.update('CLEAR_ERROR_MESSAGE');
        this.props.onRequestClose(isSuccessful);
    }

    onSubmit = (formData) => {
        const data = this.addDefaultFormValues(formData);
        api[this.props.source](data).then(() => {
            const isSuccessful = true;
            this.onRequestClose(isSuccessful);
        }).catch((error) => {
            this.update('ERROR_MESSAGE', { message: error.response.data.error });
        });
    }

    addDefaultFormValues = (formData) => {
        const data = formData;
        data.uiEndpoint = window.location.origin;
        data.type = this.props.integration.type;
        data.enabled = true;
        return data;
    }

    update = (action, nextState) => {
        this.setState(prevState => reducer(action, prevState, nextState));
    }

    renderField = (field) => {
        switch (field.type) {
            case 'text':
                return (
                    <Text type="text" className="border rounded w-full p-3 border-base-300" field={field.key} id={field.key} placeholder={field.placeholder} />
                );
            case 'password':
                return (
                    <Text type="password" className="border rounded w-full p-3 border-base-300" field={field.key} id={field.key} placeholder={field.placeholder} />
                );
            case 'select':
                return (
                    <Select field={field.key} id={field.key} options={field.options} placeholder={field.placeholder} className="border rounded w-full p-3 border-base-300" />
                );
            default:
                return '';
        }
    }

    renderFields = () => {
        if (!this.props.integration) return '';
        const fields = sourceMap[this.props.source][this.props.integration.type];
        return fields.map(field => (
            <label className="flex mt-4" htmlFor={field.key} key={field.label}>
                <div className="mr-4 flex items-center w-2/3 capitalize">{field.label}</div>
                {this.renderField(field)}
            </label>
        ));
    };

    render() {
        const FormContent = props => (
            <form onSubmit={props.formApi.submitForm}>
                <div>
                    {this.renderFields()}
                </div>
                <div className="flex items-center justify-end mt-4">
                    <button
                        className="p-3 rounded-sm bg-success-500 text-white hover:bg-success-600 uppercase"
                        type="submit"
                    >
                        Integrate
                    </button>
                </div>
            </form>
        );
        const type = (this.props.integration) ? this.props.integration.type : '';
        const { source } = this.props;
        return (
            <Modal isOpen onRequestClose={this.onRequestClose}>
                <header className="flex items-center w-full p-4 bg-primary-500 text-white uppercase">
                    <span className="flex flex-1">Configure {type} {SOURCE_LABELS[source]}</span>
                    <Icon.X className="h-4 w-4 cursor-pointer" onClick={this.onRequestClose} />
                </header>
                {(this.state.message !== '') ? <div className="px-4 py-2 bg-high-500 text-white">{this.state.message}</div> : ''}
                <div className="p-4">
                    <Form onSubmit={this.onSubmit} validateSuccess={this.validateSuccess} defaultValues={this.props.integration}>
                        <FormContent />
                    </Form>
                </div>
            </Modal>
        );
    }
}

export default IntegrationModal;
