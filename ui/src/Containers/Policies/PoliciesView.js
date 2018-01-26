import React, { Component } from 'react';
import PropTypes from 'prop-types';

import flatten from 'flat';
import omitBy from 'lodash/omitBy';
import difference from 'lodash/difference';
import pick from 'lodash/pick';

const categoriesMap = {
    IMAGE_ASSURANCE: 'Image Assurance',
    CONTAINER_CONFIGURATION: 'Container Configuration',
    PRIVILEGES_CAPABILITIES: 'Privileges and Capabilities'
};

const categoryGroupsMap = {
    imagePolicy: 'Image Assurance',
    configurationPolicy: 'Container Configuration',
    privilegePolicy: 'Privileges and Capabilities'
};

const cvssMap = {
    mathOp: {
        MAX: 'Max score',
        AVG: 'AVG score',
        MIN: 'Min score',
    },
    op: {
        GREATER_THAN: 'Is greater than',
        GREATER_THAN_OR_EQUALS: 'Is greater than or equal to',
        EQUALS: 'Is equal to',
        LESS_THAN_OR_EQUALS: 'Is less than or equal to',
        LESS_THAN: 'Is less than',
    }
};

const categories = ['imagePolicy', 'configurationPolicy', 'privilegePolicy'];

const fieldsMap = {
    id: {
        label: 'Id',
        formatValue: d => d
    },
    name: {
        label: 'Name',
        formatValue: d => d
    },
    severity: {
        label: 'Severity',
        formatValue: (d) => {
            switch (d) {
                case 'CRITICAL_SEVERITY':
                    return 'Critical';
                case 'HIGH_SEVERITY':
                    return 'High';
                case 'MEDIUM_SEVERITY':
                    return 'Medium';
                case 'LOW_SEVERITY':
                    return 'Low';
                default:
                    return '';
            }
        }
    },
    description: {
        label: 'Description',
        formatValue: d => d
    },
    notifiers: {
        label: 'Notifications',
        formatValue: d => d.join(', ')
    },
    scope: {
        label: 'Scope',
        formatValue: d => d.join(', ')
    },
    enforce: {
        label: 'Enforce',
        formatValue: d => ((d === true) ? 'Yes' : 'No')
    },
    disabled: {
        label: 'Disabled',
        formatValue: d => ((d === true) ? 'Yes' : 'No')
    },
    categories: {
        label: 'Categories',
        formatValue: d => d.map(obj => categoriesMap[obj]).join(', ')
    },
    imageName: {
        label: 'Image',
        formatValue: (d) => {
            const namespace = (d.namespace) ? d.namespace : 'any';
            const repo = (d.repo) ? d.repo : 'any';
            const tag = (d.tag) ? d.tag : 'any';
            const registry = (d.registry) ? d.registry : 'any';
            return `Alert on ${namespace} namespace${(d.namespace) ? '' : 's'} using ${repo} repo${(d.repo) ? '' : 's'} using ${tag} tag from ${registry} registry`;
        }
    },
    imageAgeDays: {
        label: 'Image Created',
        formatValue: d => ((d !== '0') ? `${Number(d)} Days ago` : '')
    },
    scanExists: {
        label: 'Scan Does Not Exist',
        formatValue: () => 'Verify that the image is scanned'
    },
    scanAgeDays: {
        label: 'Image Last Scanned',
        formatValue: d => ((d !== '0') ? `${Number(d)} Days ago` : '')
    },
    lineRule: {
        label: 'Line Rule',
        formatValue: d => `${d.instruction} ${d.value}`
    },
    cvss: {
        label: 'CVSS',
        formatValue: d => `${cvssMap.mathOp[d.mathOp]} ${cvssMap.op[d.op]} ${d.value}`
    },
    cve: {
        label: 'CVE',
        formatValue: d => d
    },
    component: {
        label: 'Component',
        formatValue: (d) => {
            const name = (d.name) ? `${d.name}` : '';
            const version = (d.version) ? d.version : '';
            return `'${name}' with version '${version}'`;
        }
    },
    env: {
        label: 'Environment',
        formatValue: (d) => {
            const key = (d.key) ? `${d.key}` : '';
            const value = (d.value) ? d.value : '';
            return `${key}=${value}`;
        }
    },
    command: {
        label: 'Command',
        formatValue: d => d
    },
    arguments: {
        label: 'Arguments',
        formatValue: d => d
    },
    directory: {
        label: 'Directory',
        formatValue: d => d
    },
    user: {
        label: 'User',
        formatValue: d => d
    },
    volumePolicy: {
        label: 'Volume Policy',
        formatValue: (d) => {
            const type = (d.type) ? `${d.type} ` : '';
            const path = (d.path) ? d.path : '';
            return `${type}${path}`;
        }
    },
    portPolicy: {
        label: 'Port',
        formatValue: (d) => {
            const protocol = (d.protocol) ? `${d.protocol} ` : '';
            const port = (d.port) ? d.port : '';
            return `${protocol}${port}`;
        }
    },
    dropCapabilities: {
        label: 'Drop Capabilities',
        formatValue: d => d
    },
    addCapabilities: {
        label: 'Add Capabilities',
        formatValue: d => d
    },
    privileged: {
        label: 'Privileged',
        formatValue: d => ((d === true) ? 'Yes' : 'No')
    },
};

class PolicyView extends Component {
    static propTypes = {
        policy: PropTypes.shape({}).isRequired
    }

    constructor(props) {
        super(props);

        this.state = {};
    }

    removeEmptyFields = (obj) => {
        const flattenedObj = flatten(obj);
        const omittedObj = omitBy(flattenedObj, value => value === null || value === undefined || value === '' || value === []);
        const newObj = flatten.unflatten(omittedObj);
        return newObj;
    }

    renderFields = () => {
        const policy = this.removeEmptyFields(this.props.policy);
        const fields = Object.keys(policy);
        const policyDetails = difference(fields, categories);
        if (!policyDetails) return '';
        return (
            <div className="px-3 py-4 border-b border-base-300">
                <div className="bg-white border border-base-200 shadow">
                    <div className="p-3 border-b border-base-300 text-primary-600 uppercase tracking-wide">Policy Details</div>
                    <div className="h-full p-3">
                        {
                            policyDetails.map((field) => {
                                if (!fieldsMap[field]) return '';
                                const { label } = fieldsMap[field];
                                const value = fieldsMap[field].formatValue(policy[field]);
                                if (!value || (Array.isArray(value) && !value.length)) return '';
                                return (
                                    <div className="mb-4" key={field}>
                                        <div className="py-2 text-primary-500">{label}</div>
                                        <div className="flex">{value}</div>
                                    </div>
                                );
                            })
                        }
                    </div>
                </div>
            </div>
        );
    }

    renderFieldsByPolicyCategories = () => {
        const policy = this.removeEmptyFields(this.props.policy);
        const policyCategoryFields = Object.keys(pick(policy, categories));
        return policyCategoryFields.map((category) => {
            const policyCategoryLabel = categoryGroupsMap[category];
            const fields = Object.keys(policy[category]);
            if (!fields.length) return '';
            return (
                <div className="px-3 py-4 border-b border-base-300" key={category}>
                    <div className="bg-white border border-base-200 shadow">
                        <div className="p-3 border-b border-base-300 text-primary-600 uppercase tracking-wide">{policyCategoryLabel}</div>
                        <div className="h-full p-3">
                            {
                                fields.map((field) => {
                                    if (!fieldsMap[field]) return '';
                                    const { label } = fieldsMap[field];
                                    const value =
                                        fieldsMap[field].formatValue(policy[category][field]);
                                    if (!value || (Array.isArray(value) && !value.length)) return '';
                                    return (
                                        <div className="mb-4" key={field}>
                                            <div className="py-2 text-primary-500">{label}</div>
                                            <div className="flex">{value}</div>
                                        </div>
                                    );
                                })
                            }
                        </div>
                    </div>
                </div>
            );
        });
    }

    render() {
        return (
            <div className="flex flex-1 flex-col bg-base-100">
                {this.renderFields()}
                {this.renderFieldsByPolicyCategories()}
            </div>
        );
    }
}

export default PolicyView;
