import React, { Component } from 'react';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import Table from 'Components/Table';
import Select from 'Components/Select';
import Pills from 'Components/Pills';

import axios from 'axios';

class ViolationsContainer extends Component {
    constructor(props) {
        super(props);

        this.state = {
            tab: {
                headers: ['Policies', 'Compliance']
            },
            select: {
                options: ['Last 24 Hours', 'Last Week', 'Last Month', 'Last Year']
            },
            pills: ['All', 'Image Assurance', 'Configurations', 'Orchestrator Target', 'Denial of Policy', 'Privileges & Capabilities', 'Account Authorization'],
            table: {
                columns: [
                    { key: 'name', label: 'Name' },
                    { key: 'description', label: 'Description' },
                    { key: 'severity', label: 'Severity' }
                ],
                rows: []
            }
        }
    }

    componentDidMount() {
        axios.get('/v1/images/policies', {
            params: {}
        }).then((response) => {
            if(!response.data.policies) return;
            const table = this.state.table;
            table.rows = response.data.policies;
            this.setState({ table: table });
        }).catch((error) => {
            console.log(error);
        });
    }

    onActivePillsChange(active) {
        console.log(active);
    }

    render() {
        return (
            <section className="flex flex-1 p-3">
                <Tabs headers={this.state.tab.headers}>
                    <TabContent name={this.state.tab.headers[0]}>
                        <div className="flex flex-1 flex-row">
                            <div className="flex flex-1 self-center justify-start">
                                <input className="appearance-none border rounded w-full py-2 px-3 border-base-500"
                                    placeholder="Scope by resource type:Registry" />
                            </div>
                            <div className="flex flex-row self-center justify-end">
                                <Select options={this.state.select.options}></Select>
                            </div>
                        </div>
                        <div className="flex flex-1 flex-col p-4">
                            <Pills data={this.state.pills} onActivePillsChange={this.onActivePillsChange.bind(this)}></Pills>
                        </div>
                        <div className="flex flex-1 flex-col p-4">
                            <Table columns={this.state.table.columns} rows={this.state.table.rows}></Table>
                        </div>
                    </TabContent>
                    <TabContent name={this.state.tab.headers[1]}>
                    </TabContent>
                </Tabs>
            </section>
        );
    }

}

export default ViolationsContainer;
