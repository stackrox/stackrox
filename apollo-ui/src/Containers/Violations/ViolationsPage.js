import React, { Component } from 'react';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import Table from 'Components/Table';
import Select from 'Components/Select';

import PolicyAlertsSidePanel from 'Containers/Violations/Policies/PolicyAlertsSidePanel';
import CompliancePage from 'Containers/Violations/Compliance/CompliancePage';

import axios from 'axios';
import emitter from 'emitter';
import queryString from 'query-string';

class ViolationsContainer extends Component {
    constructor(props) {
        super(props);

        this.params = {
            stale: false
        };

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
            tab: {
                headers: [{ text: 'Policies', disabled: false }, { text: 'Compliance', disabled: false }]
            },
            category: {
                options: ['All categories', 'Image Assurance', 'Configurations', 'Orchestrator Target', 'Denial of Policy', 'Privileges & Capabilities', 'Account Authorization']
            },
            time: {
                options: ['Last 24 Hours', 'Last Week', 'Last Month', 'Last Year']
            },
            pills: [{ text: 'Image Assurance', value: 'IMAGE_ASSURANCE', disabled: false }, { text: 'Configurations', value: 'CONFIGURATIONS', disabled: true }, { text: 'Orchestrator Target', value: 'ORCHESTRATOR_TARGET', disabled: true }, { text: 'Denial of Policy', value: 'DENIAL_OF_POLICY', disabled: true }, { text: 'Privileges & Capabilities', value: 'PRIVILEGES_AND_CAPABILITIES', disabled: true }, { text: 'Account Authorization', value: 'ACCOUNT_AUTHORIZATION', disabled: true }],
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
        const params = `?${queryString.stringify(this.params)}`;
        const { table } = this.state;
        return axios.get(`/v1/alerts/groups${params}`).then((response) => {
            if (!response.data.byCategory) return;
            let tableRows = [];
            response.data.byCategory.forEach((category) => {
                const rows = category.byPolicy.map((policy) => {
                    const result = {
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

    pollAlertGroups = () => {
        this.getAlertsGroups().then(() => {
            this.pollTimeoutId = setTimeout(this.pollAlertGroups, 5000);
        });
    }

    render() {
        return (
            <section className="flex flex-1 h-full">
                <Tabs headers={this.state.tab.headers}>
                    <TabContent>
                        <div className="flex mb-3 mx-3 flex-none">
                            <div className="flex flex-1 self-center justify-start">
                                <input
                                    className="border rounded w-full p-3  border-base-300"
                                    placeholder="Filter by registry, severity, deployment, or tag"
                                />
                            </div>
                            <div className="flex self-center justify-end">
                                <Select options={this.state.category.options} />
                            </div>
                            <div className="flex self-center justify-end">
                                <Select options={this.state.time.options} />
                            </div>
                        </div>
                        <div className="flex flex-1 border-t border-primary-300 bg-base-100">
                            <div className="w-full p-3 overflow-y-scroll bg-white rounded-sm shadow">
                                <Table columns={this.state.table.columns} rows={this.state.table.rows} onRowClick={this.onRowClick} />
                            </div>
                            <PolicyAlertsSidePanel />
                        </div>
                    </TabContent>
                    <TabContent>
                        <CompliancePage />
                    </TabContent>
                </Tabs>
            </section>
        );
    }
}

export default ViolationsContainer;
