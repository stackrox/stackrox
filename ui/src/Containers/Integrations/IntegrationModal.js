import React, { Component } from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import PropTypes from 'prop-types';
import { Form, Text, Select } from 'react-form';
import { withRouter } from 'react-router-dom';
import MultiSelect from 'react-select';
import * as Icon from 'react-feather';

import Modal from 'Components/Modal';
import Table from 'Components/Table';
import Panel from 'Components/Panel';
import tableColumnDescriptor from 'Containers/Integrations/tableColumnDescriptor';
import AuthService from 'services/AuthService';
import { saveIntegration, testIntegration, deleteIntegration } from 'services/IntegrationsService';

import { clusterTypeLabels } from 'messages/common';

const sourceMap = {
    authProviders: {
        auth0: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Auth0'
            },
            {
                label: 'Domain',
                key: 'config.domain',
                type: 'text',
                placeholder: 'your-tenant.auth0.com'
            },
            {
                label: 'Client ID',
                key: 'config.client_id',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Audience',
                key: 'config.audience',
                type: 'text',
                placeholder: 'prevent.stackrox.io'
            }
        ]
    },
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
                placeholder: 'prevent-notifier@example.com'
            },
            {
                label: 'Recipient',
                key: 'config.recipient',
                type: 'text',
                placeholder: 'prevent-alerts@example.com'
            },
            {
                label: 'TLS',
                key: 'config.tls',
                type: 'select',
                options: [{ label: 'On', value: 'true' }, { label: 'Off', value: 'false' }],
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
    imageIntegrations: {
        tenable: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Tenable'
            },
            {
                label: 'Source Inputs',
                key: 'categories',
                type: 'multiselect',
                options: [
                    { value: 'REGISTRY', label: 'Registry' },
                    { value: 'SCANNER', label: 'Scanner' }
                ],
                placeholder: ''
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
        docker: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Docker Registry'
            },
            {
                label: 'Types',
                key: 'categories',
                type: 'multiselect',
                options: [{ value: 'REGISTRY', label: 'Registry', clearableValue: false }],
                placeholder: ''
            },
            {
                label: 'Endpoint',
                key: 'config.endpoint',
                type: 'text',
                placeholder: 'registry-1.docker.io'
            },
            {
                label: 'Username',
                key: 'config.username',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Password',
                key: 'config.password',
                type: 'password',
                placeholder: ''
            }
        ],
        dtr: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Prod Docker Trusted Registry'
            },
            {
                label: 'Types',
                key: 'categories',
                type: 'multiselect',
                options: [
                    { value: 'REGISTRY', label: 'Registry' },
                    { value: 'SCANNER', label: 'Scanner' }
                ],
                placeholder: ''
            },
            {
                label: 'Endpoint',
                key: 'config.endpoint',
                type: 'text',
                placeholder: 'dtr.example.com'
            },
            {
                label: 'Username',
                key: 'config.username',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Password',
                key: 'config.password',
                type: 'password',
                placeholder: ''
            }
        ],
        artifactory: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Artifactory'
            },
            {
                label: 'Types',
                key: 'categories',
                type: 'multiselect',
                options: [{ value: 'REGISTRY', label: 'Registry', clearableValue: false }],
                placeholder: ''
            },
            {
                label: 'Endpoint',
                key: 'config.endpoint',
                type: 'text',
                placeholder: 'artifactory.example.com'
            },
            {
                label: 'Username',
                key: 'config.username',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Password',
                key: 'config.password',
                type: 'password',
                placeholder: ''
            }
        ],
        quay: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Quay'
            },
            {
                label: 'Types',
                key: 'categories',
                type: 'multiselect',
                options: [
                    { value: 'REGISTRY', label: 'Registry' },
                    { value: 'SCANNER', label: 'Scanner' }
                ],
                placeholder: ''
            },
            {
                label: 'Endpoint',
                key: 'config.endpoint',
                type: 'text',
                placeholder: 'quay.io'
            },
            {
                label: 'OAuth Token',
                key: 'config.oauthToken',
                type: 'text',
                placeholder: ''
            }
        ],
        clair: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Clair'
            },
            {
                label: 'Types',
                key: 'categories',
                type: 'multiselect',
                options: [{ value: 'SCANNER', label: 'Scanner', clearableValue: false }],
                placeholder: ''
            },
            {
                label: 'Endpoint',
                key: 'config.endpoint',
                type: 'text',
                placeholder: 'https://clair.example.com'
            }
        ],
        google: [
            {
                label: 'Integration Name',
                key: 'name',
                type: 'text',
                placeholder: 'Google Registry and Scanner'
            },
            {
                label: 'Types',
                key: 'categories',
                type: 'multiselect',
                options: [
                    { value: 'REGISTRY', label: 'Registry' },
                    { value: 'SCANNER', label: 'Scanner' }
                ],
                placeholder: ''
            },
            {
                label: 'Registry Endpoint',
                key: 'config.endpoint',
                type: 'text',
                placeholder: 'gcr.io'
            },
            {
                label: 'Project',
                key: 'config.project',
                type: 'text',
                placeholder: ''
            },
            {
                label: 'Service Account',
                key: 'config.serviceAccount',
                type: 'text',
                placeholder: ''
            }
        ]
    }
};

const SOURCE_LABELS = Object.freeze({
    authProviders: 'authentication provider',
    imageIntegrations: 'image integrations',
    notifiers: 'plugin'
});

const reducer = (action, prevState, nextState) => {
    switch (action) {
        case 'EDIT_INTEGRATION':
            return { editIntegration: nextState.editIntegration };
        case 'CLEAR_MESSAGES':
            return { errorMessage: '', successMessage: '' };
        case 'ERROR_MESSAGE':
            return { errorMessage: nextState.errorMessage };
        case 'SUCCESS_MESSAGE':
            return { successMessage: nextState.successMessage };
        default:
            return prevState;
    }
};

class IntegrationModal extends Component {
    static propTypes = {
        integrations: PropTypes.arrayOf(
            PropTypes.shape({
                type: PropTypes.string.isRequired
            })
        ).isRequired,
        source: PropTypes.oneOf(['imageIntegrations', 'notifiers', 'authProviders', 'clusters'])
            .isRequired,
        type: PropTypes.string.isRequired,
        onRequestClose: PropTypes.func.isRequired,
        onIntegrationsUpdate: PropTypes.func.isRequired,
        history: ReactRouterPropTypes.history.isRequired
    };

    constructor(props) {
        super(props);

        this.state = {
            editIntegration: null,
            errorMessage: '',
            successMessage: ''
        };
    }

    onRequestClose = isSuccessful => {
        this.update('CLEAR_MESSAGES');
        this.props.onRequestClose(isSuccessful);
    };

    onTest = () => {
        this.update('CLEAR_MESSAGES');
        const data = this.addDefaultFormValues(this.formApi.values);
        testIntegration(this.props.source, data)
            .then(() => {
                this.update('SUCCESS_MESSAGE', {
                    successMessage: 'Integration test was successful'
                });
            })
            .catch(error => {
                this.update('ERROR_MESSAGE', { errorMessage: error.response.data.error });
            });
    };

    onSubmit = formData => {
        this.update('CLEAR_MESSAGES');
        const data = this.addDefaultFormValues(formData);
        if (this.props.source === 'authProviders') {
            AuthService.saveAuthProviders(data)
                .then(() => {
                    if (!this.props.integrations.length) {
                        AuthService.logout();
                        this.props.history.go('/login');
                        return;
                    }
                    this.props.onIntegrationsUpdate(this.props.source);
                    this.update('EDIT_INTEGRATION', { editIntegration: null });
                })
                .catch(error => {
                    this.update('ERROR_MESSAGE', { errorMessage: error.response.data.error });
                });
        } else {
            saveIntegration(this.props.source, data)
                .then(() => {
                    this.props.onIntegrationsUpdate(this.props.source);
                    this.update('EDIT_INTEGRATION', { editIntegration: null });
                })
                .catch(error => {
                    this.update('ERROR_MESSAGE', { errorMessage: error.response.data.error });
                });
        }
    };

    addDefaultFormValues = formData => {
        const data = formData;
        const { location } = window;
        data.uiEndpoint = this.props.source === 'authProviders' ? location.host : location.origin;
        data.type = this.props.type;
        data.enabled = true;
        return data;
    };

    addIntegration = () => {
        this.update('EDIT_INTEGRATION', { editIntegration: {} });
    };

    deleteIntegration = () => {
        const promises = [];
        this.integrationTable.getSelectedRows().forEach(data => {
            const promise =
                this.props.source === 'authProviders'
                    ? AuthService.deleteAuthProviders(data)
                    : deleteIntegration(this.props.source, data);
            promises.push(promise);
        });
        Promise.all(promises).then(() => {
            this.integrationTable.clearSelectedRows();
            this.props.onIntegrationsUpdate(this.props.source);
        });
    };

    update = (action, nextState) => {
        this.setState(prevState => reducer(action, prevState, nextState));
    };

    renderTable = () => {
        const header = `${
            this.props.source !== 'clusters' ? this.props.type : clusterTypeLabels[this.props.type]
        } Integrations`;
        const buttons =
            this.props.source !== 'clusters'
                ? [
                      {
                          renderIcon: () => <Icon.Trash2 className="h-4 w-4" />,
                          text: 'Delete',
                          className:
                              'flex py-1 px-2 rounded-sm text-danger-600 hover:text-white hover:bg-danger-400 uppercase text-center text-sm items-center ml-2 bg-white border-2 border-danger-400',
                          onClick: this.deleteIntegration,
                          disabled: this.state.editIntegration !== null
                      },
                      {
                          renderIcon: () => <Icon.Plus className="h-4 w-4" />,
                          text: 'Add Integration',
                          className:
                              'flex py-1 px-2 rounded-sm text-success-600 hover:text-white hover:bg-success-400 uppercase text-center text-sm items-center ml-2 bg-white border-2 border-success-400',
                          onClick: this.addIntegration,
                          disabled: this.state.editIntegration !== null
                      }
                  ]
                : [
                      {
                          renderIcon: () => <Icon.Trash2 className="h-4 w-4" />,
                          text: 'Delete',
                          className:
                              'flex py-1 px-2 rounded-sm text-danger-600 hover:text-white hover:bg-danger-400 uppercase text-center text-sm items-center ml-2 bg-white border-2 border-danger-400',
                          onClick: this.deleteIntegration,
                          disabled: this.state.editIntegration !== null
                      },
                      {
                          renderIcon: () => <Icon.Plus className="h-4 w-4" />,
                          text: 'Add Integration',
                          className:
                              'flex py-1 px-2 rounded-sm text-success-600 hover:text-white hover:bg-success-400 uppercase text-center text-sm items-center ml-2 bg-white border-2 border-success-400',
                          onClick: this.addIntegration,
                          disabled: this.state.editIntegration !== null
                      }
                  ];
        const columns =
            this.props.source !== 'clusters'
                ? tableColumnDescriptor[this.props.source][this.props.type]
                : tableColumnDescriptor[this.props.source];
        const rows = this.props.integrations;
        const onRowClickHandler = () => integration => {
            if (this.props.source !== 'clusters')
                this.update('EDIT_INTEGRATION', { editIntegration: integration });
        };

        return (
            <div className="flex flex-1">
                <Panel header={header} buttons={buttons}>
                    {rows.length !== 0 ? (
                        <Table
                            columns={columns}
                            rows={rows}
                            checkboxes={this.props.source !== 'clusters'}
                            onRowClick={onRowClickHandler()}
                            ref={table => {
                                this.integrationTable = table;
                            }}
                        />
                    ) : (
                        <div className="p3 w-full my-auto text-center capitalize">
                            {`No ${
                                this.props.source !== 'clusters'
                                    ? this.props.type
                                    : clusterTypeLabels[this.props.type]
                            } integrations`}
                        </div>
                    )}
                </Panel>
            </div>
        );
    };

    renderField = field => {
        const handleMultiSelectChange = () => newValue => {
            const values = newValue !== '' ? newValue.split(',') : [];
            this.formApi.setValue(field.key, values);
        };
        switch (field.type) {
            case 'text':
                return (
                    <Text
                        type="text"
                        className="border rounded w-full p-3 border-base-300"
                        field={field.key}
                        id={field.key}
                        placeholder={field.placeholder}
                    />
                );
            case 'password':
                return (
                    <Text
                        type="password"
                        className="border rounded w-full p-3 border-base-300"
                        field={field.key}
                        id={field.key}
                        placeholder={field.placeholder}
                    />
                );
            case 'select':
                return (
                    <Select
                        field={field.key}
                        id={field.key}
                        options={field.options}
                        placeholder={field.placeholder}
                        className="border rounded w-full p-3 border-base-300"
                    />
                );
            case 'multiselect':
                return (
                    <MultiSelect
                        key={field.key}
                        multi
                        onChange={handleMultiSelectChange()}
                        options={field.options}
                        placeholder={field.placeholder}
                        removeSelected
                        simpleValue
                        value={this.formApi.values[field.key]}
                        className="text-base-600 font-400 w-full"
                    />
                );
            default:
                return '';
        }
    };

    renderFields = () => {
        const fields = sourceMap[this.props.source][this.props.type];
        return fields.map(field => (
            <label className="flex mt-4" htmlFor={field.key} key={field.label}>
                <div className="mr-4 flex items-center w-2/3 capitalize">{field.label}</div>
                {this.renderField(field)}
            </label>
        ));
    };

    renderForm = () => {
        if (!this.state.editIntegration) return '';
        const header = this.state.editIntegration.name || 'New Integration';
        const buttons = [
            {
                renderIcon: () => <Icon.X className="h-4 w-4" />,
                text: 'Cancel',
                className:
                    'flex py-1 px-2 rounded-sm text-primary-600 hover:text-white hover:bg-primary-400 uppercase text-center text-sm items-center ml-2 bg-white border-2 border-primary-400',
                onClick: () => {
                    this.update('EDIT_INTEGRATION', { editIntegration: null });
                }
            },
            {
                renderIcon: () => <Icon.Save className="h-4 w-4" />,
                text: `${this.state.editIntegration.name ? 'Save' : 'Create'} Integration`,
                className:
                    'flex py-1 px-2 rounded-sm text-success-600 hover:text-white hover:bg-success-400 uppercase text-center text-sm items-center ml-2 bg-white border-2 border-success-400',
                onClick: () => {
                    this.formApi.submitForm();
                }
            }
        ];
        if (this.props.source !== 'authProviders') {
            const testButton = {
                renderIcon: () => <Icon.Check className="h-4 w-4" />,
                text: `Test Integration`,
                className:
                    'flex py-1 px-2 rounded-sm text-primary-600 hover:text-white hover:bg-primary-400 uppercase text-center text-sm items-center ml-2 bg-white border-2 border-primary-400',
                onClick: () => {
                    this.onTest();
                }
            };
            buttons.splice(1, 0, testButton);
        }
        const key = this.state.editIntegration
            ? this.state.editIntegration.name
            : 'new-integration';
        const FormContent = props => {
            this.formApi = props.formApi;
            return (
                <form onSubmit={props.formApi.submitForm} className="w-full p-4">
                    <div>{this.renderFields()}</div>
                </form>
            );
        };
        return (
            <div className="flex flex-1">
                <Panel header={header} buttons={buttons}>
                    <Form
                        onSubmit={this.onSubmit}
                        validateSuccess={this.validateSuccess}
                        defaultValues={this.state.editIntegration}
                        key={key}
                    >
                        <FormContent />
                    </Form>
                </Panel>
            </div>
        );
    };

    render() {
        const { source, type } = this.props;
        return (
            <Modal isOpen onRequestClose={this.onRequestClose} className="w-5/6 h-full">
                <header className="flex items-center w-full p-4 bg-primary-500 text-white uppercase">
                    <span className="flex flex-1">
                        {source !== 'clusters'
                            ? `Configure ${type} ${SOURCE_LABELS[source]}`
                            : `Configure ${clusterTypeLabels[type]}`}
                    </span>
                    <Icon.X className="h-4 w-4 cursor-pointer" onClick={this.onRequestClose} />
                </header>
                {this.state.errorMessage !== '' && (
                    <div className="px-4 py-2 bg-high-500 text-white">
                        {this.state.errorMessage}
                    </div>
                )}
                {this.state.successMessage !== '' && (
                    <div className="px-4 py-2 bg-success-500 text-white">
                        {this.state.successMessage}
                    </div>
                )}
                <div className="flex flex-1 w-full bg-white">
                    {this.renderTable()}
                    {this.renderForm()}
                </div>
            </Modal>
        );
    }
}

export default withRouter(IntegrationModal);
