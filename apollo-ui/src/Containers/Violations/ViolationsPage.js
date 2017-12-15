import React, { Component } from 'react';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import Table from 'Components/Table';
import Select from 'Components/Select';

import PolicyAlertsSidePanel from "Containers/Violations/Policies/PolicyAlertsSidePanel";
import CompliancePage from 'Containers/Violations/Compliance/CompliancePage';

import axios from 'axios';
import emitter from 'emitter';
import queryString from 'query-string';

class ViolationsContainer extends Component {
    constructor(props) {
        super(props);

        this.params = {};

        this.timeout = null;

        var setSeverityClass = (item) => { 
            switch(item) {
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
                    { key: 'severity', label: 'Severity', classFunc: setSeverityClass},
                    { key: 'numAlerts', label: 'Alerts', align: 'right' }
                ],
                rows: []
            }
        }

        this.pollAlertGroups = this.pollAlertGroups.bind(this);
    }

    componentDidMount() {
        this.pollAlertGroups();
    }

    pollAlertGroups() {
        var promise = this.getAlertsGroups();
        var func = this.pollAlertGroups;
        // eslint-disable-next-line
        var timeout = this.timeout;
        promise.then(function (result) {
            timeout = setTimeout(func, 5000);
        });
    }

    getAlertsGroups() {
        var promise = new Promise((resolve, reject) => {
            var params = "?" + queryString.stringify(this.params);
            const table = this.state.table;
            axios.get(`/v1/alerts/groups${params}`).then((response) => {
                if (!response.data.byCategory) return;
                response.data.byCategory.map((category) => {
                    return table.rows = category.byPolicy.map((policy) => {
                        var result = {
                            name: policy.policy.name,
                            description: policy.policy.imagePolicy.description,
                            category: category.category.replace('_', ' ').capitalizeFirstLetterOfWord(),
                            severity: policy.policy.severity.split('_')[0].capitalizeFirstLetterOfWord(),
                            numAlerts: policy.numAlerts
                        }
                        return result;
                    });
                });
                this.setState({ table: table });
                resolve({});
            }).catch((error) => {
                table.rows = [];
                this.setState({ table: table });
                resolve({});
            });
        });
        return promise;
    }

    onRowClick(row) {
        emitter.emit('PolicyAlertsTable:row-selected', row);
    }

    onActivePillsChange(active) {
        const params = this.params;
        params.category = Object.keys(active);
        this.getAlertsGroups();
    }

    render() {
        return (
            <section className="flex flex-1 h-full">
                <Tabs headers={this.state.tab.headers}>
                    <TabContent name={this.state.tab.headers[0]}>
                        <div className="flex mb-3 mx-3 flex-none">
                             <div className="flex flex-1 self-center justify-start">
                                <input className="border rounded w-full p-3  border-base-300"
                                    placeholder="Filter by registry, severity, deployment, or tag" />
                            </div>
                            <div className="flex self-center justify-end">
                                <Select options={this.state.category.options}></Select>
                            </div>
                            <div className="flex self-center justify-end">
                                <Select options={this.state.time.options}></Select>
                            </div>
                        </div>
                        <div className="flex flex-1 border-t border-primary-300 bg-base-100">
                            <div className="w-full p-3 overflow-y-scroll bg-white rounded-sm shadow">
                                <Table columns={this.state.table.columns} rows={this.state.table.rows} onRowClick={this.onRowClick.bind(this)}></Table>
                            </div>
                            <PolicyAlertsSidePanel></PolicyAlertsSidePanel>
                        </div>
                    </TabContent>
                    <TabContent name={this.state.tab.headers[1]}>
                        <CompliancePage></CompliancePage>
                    </TabContent>
                </Tabs>
            </section>
        );
    }

    componentWillUnmount() {
        if (this.timeout) this.timeout.cancel();
    }

}

export default ViolationsContainer;
