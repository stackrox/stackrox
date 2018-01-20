import React, { Component } from 'react';
import Table from 'Components/Table';
import MultiSelect from 'react-select';

import PolicyAlertsSidePanel from 'Containers/Violations/PolicyAlertsSidePanel';

import axios from 'axios';
import emitter from 'emitter';
import queryString from 'query-string';

class PoliciesPage extends Component {
    constructor(props) {
        super(props);

        this.pollTimeoutId = null;

        const setSeverityClass = (item) => {
            switch (item) {
                case 'Low':
                    return 'text-low-500';
                case 'Medium':
                    return 'text-medium-500';
                case 'High':
                    return 'text-high-500';
                case 'Critical':
                    return 'text-critical-500';
                default:
                    return '';
            }
        };

        this.state = {
            params: {
                stale: false,
                category: []
            },
            category: {
                options: [{ label: 'Image Assurance', value: 'IMAGE_ASSURANCE' }, { label: 'Container Configuration', value: 'CONTAINER_CONFIGURATION' }, { label: 'Privileges & Capabilities', value: 'PRIVILEGES_CAPABILITIES' }]
            },
            table: {
                columns: [
                    { key: 'name', label: 'Name' },
                    { key: 'description', label: 'Description' },
                    { key: 'category', label: 'Category' },
                    { key: 'severity', label: 'Severity', classFunc: setSeverityClass },
                    { key: 'numAlerts', label: 'Alerts', align: 'right' }
                ],
                rows: []
            }
        };
    }

    componentDidMount() {
        this.pollAlertGroups();
    }

    componentWillUnmount() {
        if (this.pollTimeoutId) {
            clearTimeout(this.pollTimeoutId);
            this.pollTimeoutId = null;
        }
    }

    onRowClick = (row) => {
        emitter.emit('PolicyAlertsTable:row-selected', row);
    }

    onActivePillsChange(active) {
        const { params } = this;
        params.category = Object.keys(active);
        this.getAlertsGroups();
    }

    getAlertsGroups = () => {
        const params = this.getParams();
        const { table } = this.state;
        return axios.get(`/v1/alerts/groups${params}`).then((response) => {
            if (!response.data.byCategory) return;
            let tableRows = [];
            response.data.byCategory.forEach((category) => {
                const rows = category.byPolicy.map((policy) => {
                    const result = {
                        id: policy.policy.id,
                        name: policy.policy.name,
                        description: policy.policy.description,
                        category: category.category.replace('_', ' ').capitalizeFirstLetterOfWord(),
                        severity: policy.policy.severity.split('_')[0].capitalizeFirstLetterOfWord(),
                        numAlerts: policy.numAlerts
                    };
                    return result;
                });
                tableRows = tableRows.concat(rows);
            });
            table.rows = tableRows;
            this.setState({ table });
        }).catch(() => {
            table.rows = [];
            this.setState({ table });
        });
    }

    getParams = () => {
        let { params } = this.state;
        params = `?${queryString.stringify(params)}`;
        return params;
    };

    changeCategoriesParam = (categories) => {
        const { params } = this.state;
        const selectedCategories = (categories !== '') ? categories.split(',') : [];
        params.category = selectedCategories;
        this.setState({ params });
        this.getAlertsGroups();
    }

    pollAlertGroups = () => {
        this.getAlertsGroups().then(() => {
            this.pollTimeoutId = setTimeout(this.pollAlertGroups, 5000);
        });
    }

    render() {
        return (
            <section className="flex flex-1 h-full">
                <div className="flex flex-1 mt-3 flex-col">
                    <div className="flex mb-3 mx-3 flex-none">
                        <div className="flex flex-1 self-center justify-start">
                            <input
                                className="border rounded w-full p-3  border-base-300"
                                placeholder="Filter by registry, severity, deployment, or tag"
                            />
                        </div>
                        <div className="flex self-center justify-end ml-3">
                            <MultiSelect
                                multi
                                onChange={this.changeCategoriesParam}
                                options={this.state.category.options}
                                placeholder="Select categories"
                                removeSelected
                                simpleValue
                                value={this.state.params.category}
                                className="text-base-600 font-400 min-w-64"
                            />
                        </div>
                    </div>
                    <div className="flex flex-1 border-t border-primary-300 bg-base-100">
                        <div className="w-full p-3 overflow-y-scroll bg-white rounded-sm shadow">
                            <Table columns={this.state.table.columns} rows={this.state.table.rows} onRowClick={this.onRowClick} />
                        </div>
                        <PolicyAlertsSidePanel />
                    </div>
                </div>
            </section>
        );
    }
}

export default PoliciesPage;
