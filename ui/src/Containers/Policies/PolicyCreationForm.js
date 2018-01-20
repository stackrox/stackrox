import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Text, TextArea, Select } from 'react-form';
import MultiSelect from 'react-select';
import 'react-select/dist/react-select.css';
import FormField from 'Components/FormField';
import CustomSelect from 'Components/Select';

import flatten from 'flat';
import flattenObject from 'utils/flattenObject';
import differenceBy from 'lodash/differenceBy';
import intersection from 'lodash/intersection';
import omitBy from 'lodash/omitBy';
import axios from 'axios';

const categoriesMap = {
    imagePolicy: 'IMAGE_ASSURANCE',
    configurationPolicy: 'CONTAINER_CONFIGURATION',
    privilegePolicy: 'PRIVILEGES_CAPABILITIES'
};

const reducer = (action, prevState, nextState) => {
    switch (action) {
        case 'UPDATE_POLICY_FIELDS':
            return { policyFields: nextState.policyFields };
        default:
            return prevState;
    }
};

class PolicyCreationForm extends Component {
    static propTypes = {
        policy: PropTypes.shape({}).isRequired,
        formApi: PropTypes.shape({
            submitForm: PropTypes.func,
            setValue: PropTypes.func,
            setAllValues: PropTypes.func,
            clearAll: PropTypes.func,
            values: PropTypes.shape({})
        }).isRequired
    }

    constructor(props) {
        super(props);

        this.state = {
            policyFields: {
                policyDetails: [
                    {
                        label: 'Name',
                        value: 'name',
                        type: 'text',
                        required: true
                    },
                    {
                        label: 'Severity',
                        value: 'severity',
                        type: 'select',
                        options: [
                            { label: 'Critical', value: 'CRITICAL_SEVERITY' },
                            { label: 'High', value: 'HIGH_SEVERITY' },
                            { label: 'Medium', value: 'MEDIUM_SEVERITY' },
                            { label: 'Low', value: 'LOW_SEVERITY' },
                        ],
                        placeholder: 'Select a severity level',
                        required: true
                    },
                    {
                        label: 'Description',
                        value: 'description',
                        type: 'textarea',
                        required: true
                    },
                    {
                        label: 'Notifications',
                        value: 'notifiers',
                        exclude: false,
                        type: 'multiselect',
                        options: [],
                        required: true

                    },
                    {
                        label: 'Enforce',
                        value: 'enforce',
                        exclude: false,
                        type: 'select',
                        options: [
                            { label: 'Yes', value: true },
                            { label: 'No', value: false }
                        ],
                        required: true
                    }
                ],
                imagePolicy: [
                    {
                        label: 'Image Registry',
                        value: 'imagePolicy.imageName.registry',
                        type: 'text',
                        placeholder: 'docker.io',
                        required: false
                    },
                    {
                        label: 'Image Namespace',
                        value: 'imagePolicy.imageName.namespace',
                        type: 'text',
                        required: false
                    },
                    {
                        label: 'Image Repository',
                        value: 'imagePolicy.imageName.repo',
                        type: 'text',
                        placeholder: 'nginx',
                        required: false
                    },
                    {
                        label: 'Image Tag',
                        value: 'imagePolicy.imageName.tag',
                        type: 'text',
                        placeholder: 'latest',
                        required: false
                    },
                    {
                        label: 'Days since Image created',
                        value: 'imagePolicy.imageAgeDays',
                        type: 'number',
                        placeholder: '1 Day Ago',
                        required: false
                    },
                    {
                        label: 'Days since Image scanned',
                        value: 'imagePolicy.imageScanDays',
                        type: 'number',
                        placeholder: '1 Day Ago',
                        required: false
                    },
                    {
                        label: 'Line Rule',
                        value: 'imagePolicy.lineRule',
                        type: 'group',
                        values: [
                            {
                                value: 'imagePolicy.lineRule.instruction',
                                type: 'select',
                                options: [
                                    { label: 'FROM', value: 'FROM' },
                                    { label: 'LABEL', value: 'LABEL' },
                                    { label: 'RUN', value: 'RUN' },
                                    { label: 'CMD', value: 'CMD' },
                                    { label: 'EXPOSE', value: 'EXPOSE' },
                                    { label: 'ENV', value: 'ENV' },
                                    { label: 'ADD', value: 'ADD' },
                                    { label: 'COPY', value: 'COPY' },
                                    { label: 'ENTRYPOINT', value: 'ENTRYPOINT' },
                                    { label: 'VOLUME', value: 'VOLUME' },
                                    { label: 'USER', value: 'USER' },
                                    { label: 'WORKDIR', value: 'WORKDIR' },
                                    { label: 'ONBUILD', value: 'ONBUILD' }
                                ]
                            },
                            {
                                value: 'imagePolicy.lineRule.value',
                                type: 'text',
                                placeholder: '.*example.*'
                            }
                        ],
                        required: false
                    },
                    {
                        label: 'CVSS',
                        value: 'imagePolicy.cvss',
                        type: 'group',
                        values: [
                            {
                                value: 'imagePolicy.cvss.mathOp',
                                type: 'select',
                                options: [
                                    { label: 'Max score', value: 'MAX' },
                                    { label: 'Average score', value: 'AVG' },
                                    { label: 'Min score', value: 'MIN' }
                                ]
                            },
                            {
                                value: 'imagePolicy.cvss.op',
                                type: 'select',
                                options: [
                                    { label: 'Is greater than', value: 'GREATER_THAN' },
                                    { label: 'Is greater than or equal to', value: 'GREATER_THAN_OR_EQUALS' },
                                    { label: 'Is equal to', value: 'EQUALS' },
                                    { label: 'Is less than or equal to', value: 'LESS_THAN_OR_EQUALS' },
                                    { label: 'Is less than', value: 'LESS_THAN' }
                                ]
                            },
                            {
                                value: 'imagePolicy.cvss.value',
                                type: 'number',
                                placeholder: '0-10',
                                max: 10,
                                min: 0
                            },
                        ],
                        required: false
                    },
                    {
                        label: 'CVE',
                        value: 'imagePolicy.cve',
                        type: 'text',
                        placeholder: 'CVE-2017-11882',
                        required: false
                    },
                    {
                        label: 'Component',
                        value: 'imagePolicy.component',
                        type: 'text',
                        placeholder: '^example*',
                        required: false
                    }
                ],
                configurationPolicy: [
                    {
                        label: 'Environment',
                        value: 'configurationPolicy.env',
                        type: 'group',
                        values: [
                            {
                                value: 'configurationPolicy.env.key',
                                type: 'text',
                                placeholder: 'Key'
                            },
                            {
                                value: 'configurationPolicy.env.value',
                                type: 'text',
                                placeholder: 'Value'
                            }
                        ],
                        required: false
                    },
                    {
                        label: 'Command',
                        value: 'configurationPolicy.command',
                        type: 'text',
                        required: false
                    },
                    {
                        label: 'Arguments',
                        value: 'configurationPolicy.args',
                        type: 'text',
                        required: false
                    },
                    {
                        label: 'Directory',
                        value: 'configurationPolicy.directory',
                        type: 'text',
                        required: false
                    },
                    {
                        label: 'User',
                        value: 'configurationPolicy.user',
                        type: 'text',
                        required: false
                    },
                    {
                        label: 'Volume Name',
                        value: 'configurationPolicy.volumePolicy.name',
                        type: 'text',
                        placeholder: '/var/run/docker.sock',
                        required: false
                    },
                    {
                        label: 'Volume Path',
                        value: 'configurationPolicy.volumePolicy.path',
                        type: 'text',
                        placeholder: '^/var/run/docker.sock$',
                        required: false
                    },
                    {
                        label: 'Volume Type',
                        value: 'configurationPolicy.volumePolicy.type',
                        type: 'text',
                        placeholder: 'bind, secret',
                        required: false
                    },
                    {
                        label: 'Protocol',
                        value: 'configurationPolicy.portPolicy.protocol',
                        type: 'text',
                        required: false
                    },
                    {
                        label: 'Port',
                        value: 'configurationPolicy.portPolicy.port',
                        type: 'number',
                        required: false
                    }
                ],
                privilegePolicy: [
                    {
                        label: 'Privileged',
                        value: 'privilegePolicy.privileged',
                        type: 'select',
                        options: [
                            { label: 'Yes', value: true },
                            { label: 'No', value: false }
                        ],
                        required: false
                    },
                    {
                        label: 'SELinux User',
                        value: 'privilegePolicy.selinux.user',
                        type: 'text',
                        required: false
                    },
                    {
                        label: 'SELinux Role',
                        value: 'privilegePolicy.selinux.role',
                        type: 'text',
                        required: false
                    },
                    {
                        label: 'SELinux Type',
                        value: 'privilegePolicy.selinux.type',
                        type: 'text',
                        required: false
                    },
                    {
                        label: 'SELinux Level',
                        value: 'privilegePolicy.selinux.level',
                        type: 'text',
                        required: false
                    },
                    {
                        label: 'Drop Capabilities',
                        value: 'privilegePolicy.dropCapabilities',
                        type: 'multiselect',
                        options: [],
                        required: false
                    },
                    {
                        label: 'Add Capabilities',
                        value: 'privilegePolicy.addCapabilities',
                        type: 'multiselect',
                        options: [],
                        required: false
                    }
                ]
            }
        };
    }

    componentDidMount() {
        this.setFormFields();
        this.setNotifierFieldOptions();
    }

    componentWillReceiveProps() {
        this.forceUpdate();
    }

    getNotifiers = () => axios.get('/v1/notifiers');

    setNotifierFieldOptions = () => {
        this.getNotifiers().then((response) => {
            if (!response.data.notifiers) return;
            const { notifiers } = response.data;
            const { policyFields } = this.state;
            policyFields.policyDetails = policyFields.policyDetails.map((field) => {
                const newField = field;
                if (field.value === 'notifiers') newField.options = notifiers.map(notifier => ({ label: notifier.name, value: notifier.name }));
                return newField;
            });
            this.update('UPDATE_POLICY_FIELDS', { policyFields });
        });
    }

    setFormFields = () => {
        const filteredPolicy = this.removeEmptyFields(this.props.policy);
        this.props.formApi.setAllValues(filteredPolicy);
    }

    setCategories = (obj) => {
        const newObj = obj;
        newObj.categories = intersection(Object.keys(this.state.policyFields).filter(o => o !== 'policyDetails'), Object.keys(obj)).map(o => categoriesMap[o]);
        return obj;
    };

    preSubmit = (policy) => {
        let newPolicy = this.removeEmptyFields(policy);
        newPolicy = this.convertNumberValues(newPolicy);
        newPolicy = this.setCategories(newPolicy);
        return newPolicy;
    }

    convertNumberValues = (obj) => {
        const flattenedObj = flatten(obj);
        Object.keys(flattenedObj).filter(o => o !== 'id').forEach((key) => {
            if (typeof flattenedObj[key] === 'string' && !Number.isNaN(parseInt(flattenedObj[key], 10))) {
                flattenedObj[key] = parseInt(flattenedObj[key], 10);
            }
        });
        const newObj = flatten.unflatten(flattenedObj);
        return newObj;
    }

    removeEmptyFields = (obj) => {
        const flattenedObj = flatten(obj);
        const omittedObj = omitBy(flattenedObj, value => value === null || value === undefined || value === '' || value === []);
        const newObj = flatten.unflatten(omittedObj);
        return newObj;
    }

    clearAll = () => {
        this.props.formApi.clearAll();
    }

    submitForm = () => {
        this.props.formApi.submitForm();
    }

    addFormField = (fieldValue) => {
        let fieldToAdd = {};
        Object.keys(this.state.policyFields).forEach((fieldGroup) => {
            const field = this.state.policyFields[fieldGroup].find(obj => obj.value === fieldValue);
            if (field) fieldToAdd = field;
        });
        if (fieldToAdd.type === 'group') {
            fieldToAdd.values.forEach((field) => {
                this.props.formApi.setValue(field.value, '');
            });
        } else this.props.formApi.setValue(fieldToAdd.value, '');
    }

    removeField = (fieldValue) => {
        let fieldToRemove = {};
        Object.keys(this.state.policyFields).forEach((fieldGroup) => {
            const field = this.state.policyFields[fieldGroup].find(obj => obj.value === fieldValue);

            if (field) fieldToRemove = field;
        });
        if (fieldToRemove.type === 'group') {
            fieldToRemove.values.forEach((field) => {
                this.props.formApi.setValue(field.value, null);
            });
        } else this.props.formApi.setValue(fieldToRemove.value, null);
    }

    update = (action, nextState) => {
        this.setState(PrevState => reducer(action, PrevState, nextState));
    }

    renderFieldInput = (field, value) => {
        let handleMultiSelectChange = () => { };
        switch (field.type) {
            case 'text':
                return (<Text type="text" key={field.value} field={field.value} id={field.value} placeholder={field.placeholder} className="border rounded-l p-3 border-base-300 w-full font-400" />);
            case 'number':
                return (<Text type="number" max={field.max} min={field.min} key={field.value} field={field.value} id={field.value} placeholder={field.placeholder} className="border rounded-l p-3 border-base-300 w-full font-400" />);
            case 'textarea':
                return (
                    <TextArea
                        key={field.value}
                        className="border rounded-l p-3 border-base-300 text-base-600 w-full font-400"
                        field={field.value}
                        id={field.value}
                        rows="4"
                    />
                );
            case 'select':
                return (
                    <Select
                        key={field.value}
                        className="border bg-white border-base-300 text-base-600 p-3 pr-8 rounded-r-sm cursor-pointer z-1 focus:border-base-300 w-full font-400"
                        field={field.value}
                        id={field.value}
                        options={field.options}
                        placeholder={field.placeholder}
                    />
                );
            case 'multiselect':
                handleMultiSelectChange = (newValue) => {
                    const values = (newValue !== '') ? newValue.split(',') : [];
                    this.props.formApi.setValue(field.value, values);
                };

                return (
                    <MultiSelect
                        key={field.value}
                        multi
                        onChange={handleMultiSelectChange}
                        options={field.options}
                        placeholder="Select options"
                        removeSelected
                        simpleValue
                        value={value}
                        className="text-base-600 font-400 w-full"
                    />
                );
            case 'group':
                return field.values.map(input => this.renderFieldInput(input, input.value));
            default:
                return '';
        }
    }

    renderFields = (possibleFields) => {
        const formFields = Object.keys(flattenObject(this.props.formApi.values));
        const filteredFields = possibleFields.filter((obj) => {
            if (obj.type === 'group') return formFields.find(o => o.includes(obj.value));
            return formFields.indexOf(obj.value) !== -1 || obj.required === true;
        });
        if (!filteredFields.length) {
            return (
                <div className="p-3 text-base-500 font-500">
                    No Fields Added
                </div>
            );
        }
        return (
            <div className="h-full p-3">
                {
                    filteredFields.map((field) => {
                        const value = this.props.formApi.values[field.value];
                        const removeField = (!field.required) ? this.removeField : null;
                        return (
                            <FormField key={field.value} label={field.label} value={field.value} onRemove={removeField}>
                                {this.renderFieldInput(field, value)}
                            </FormField>
                        );
                    })
                }
            </div>
        );
    }

    renderFormFieldsBuilder = (possibleFields) => {
        const formFields = Object.keys(flattenObject(this.props.formApi.values))
            .map(d => ({ value: d }));
        const availableFields = differenceBy(possibleFields, formFields, 'value').filter((obj) => {
            if (obj.type === 'group') return !formFields.find(o => o.value.includes(obj.value));
            return !obj.required;
        });
        const placeholder = 'Add a field';
        if (!availableFields.length) return '';
        return (
            <div className="flex p-3 border-t border-base-200 bg-success-100">
                <span className="w-full">
                    <CustomSelect
                        className="border bg-white border-success-500 text-success-600 p-3 pr-8 rounded cursor-pointer w-full font-400"
                        placeholder={placeholder}
                        options={availableFields}
                        value=""
                        onChange={this.addFormField}
                    />
                </span>
            </div>
        );
    }

    renderGroupedFields = () => {
        const fieldGroups = Object.keys(this.state.policyFields);
        return fieldGroups.map((fieldGroup) => {
            const fieldGroupName = fieldGroup.replace(/([A-Z])/g, ' $1').replace(/^./, str => str.toUpperCase());
            const possibleFields = this.state.policyFields[fieldGroup];
            return (
                <div className="px-3 py-4 bg-base-100 border-b border-base-300" key={fieldGroup}>
                    <div className="bg-white border border-base-200 shadow">
                        <div className="p-3 border-b border-base-300 text-primary-600 uppercase tracking-wide">{fieldGroupName}</div>
                        {this.renderFields(possibleFields)}
                        {this.renderFormFieldsBuilder(possibleFields)}
                    </div>
                </div>
            );
        });
    }

    render() {
        return (
            <div className="flex flex-1 flex-col">
                <form id="dynamic-form" className="flex flex-1 flex-col">
                    {this.renderGroupedFields()}
                </form>
            </div>
        );
    }
}

export default PolicyCreationForm;
