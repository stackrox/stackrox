import React, { Component } from 'react';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import Table from 'Components/Table';
import Select from 'Components/Select';
// import Pills from 'Components/Pills';

import CompliancePage from 'Containers/Violations/Compliance/CompliancePage';

import axios from 'axios';
import emitter from 'emitter';
import queryString from 'query-string';

class ViolationsContainer extends Component {
    constructor(props) {
        super(props);

        this.params = {};

        this.state = {
            tab: {
                headers: ['Policies', 'Compliance']
            },
            category: {
                options: ['All categories', 'Image Assurance', 'Configurations', 'Orchestrator Target', 'Denial of Policy', 'Privileges & Capabilities', 'Account Authorization']
            },
            time: {
                options: ['Last 24 Hours', 'Last Week', 'Last Month', 'Last Year']
            },
            pills: [{ text: 'Image Assurance', value: 'IMAGE_ASSURANCE' }, { text: 'Configurations', value: 'CONFIGURATIONS' }, { text: 'Orchestrator Target', value: 'ORCHESTRATOR_TARGET' }, { text: 'Denial of Policy', value: 'DENIAL_OF_POLICY' }, { text: 'Privileges & Capabilities', value: 'PRIVILEGES_AND_CAPABILITIES' }, { text: 'Account Authorization', value: 'ACCOUNT_AUTHORIZATION' }],
            table: {
                columns: [
                    { key: 'name', label: 'Name' },
                    { key: 'description', label: 'Description' },
                    { key: 'category', label: 'Category' },
                    { key: 'severity', label: 'Severity' },
                    { key: 'numAlerts', label: 'Alerts' }
                ],
                rows: []
            }
        }
    }

    componentDidMount() {
        this.getAlertsGroups();
    }

    onRowClick(row) {
        emitter.emit('Table:row-selected', row);
    }

    onActivePillsChange(active) {
        const params = this.params;
        params.category = Object.keys(active);
        this.getAlertsGroups();
    }

    getAlertsGroups() {
        var params = "?" + queryString.stringify(this.params);
        const table = this.state.table;
        axios.get(`/v1/alerts/groups${params}`).then((response) => {
            if (!response.data.byCategory) return;
            response.data.byCategory.map((category) => {
                return table.rows = category.byPolicy.map((policy) => {
                    var result = {
                        name: policy.policy.name,
                        description: policy.policy.imagePolicy.description,
                        category: category.category,
                        severity: policy.policy.severity,
                        numAlerts: policy.numAlerts
                    }
                    return result;
                });
            });
            this.setState({ table: table });
        }).catch((error) => {
            table.rows = [];
            this.setState({ table: table });
        });
    }

    render() {
        return (
            <section className="flex flex-1 p-3 h-full">
                <Tabs headers={this.state.tab.headers}>
                    <TabContent name={this.state.tab.headers[0]}>
                        <div className="flex flex-1 flex-row mb-3">
                            <div className="flex flex-1 self-center justify-start">
                                <input className="border rounded w-full p-3  border-base-300"
                                    placeholder="Filter by registry, severity, deployment, or tag" />
                            </div>
                            <div className="flex flex-row self-center justify-end">
                                <Select options={this.state.category.options}></Select>
                            </div>
                            <div className="flex flex-row self-center justify-end">
                                <Select options={this.state.time.options}></Select>
                            </div>
                        </div>
                        <div className="flex flex-1 flex-col pb-4">
                            <Table columns={this.state.table.columns} rows={this.state.table.rows} onRowClick={this.onRowClick.bind(this)}></Table>
                        </div>
                    </TabContent>
                    <TabContent name={this.state.tab.headers[1]}>
                        <CompliancePage></CompliancePage>
                    </TabContent>
                </Tabs>
            </section>
        );
    }

}

export default ViolationsContainer;
