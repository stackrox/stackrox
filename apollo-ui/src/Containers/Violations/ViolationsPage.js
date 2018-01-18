import React, { Component } from 'react';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';

import CompliancePage from 'Containers/Violations/Compliance/CompliancePage';
import PoliciesPage from 'Containers/Violations/Policies/PoliciesPage';

class ViolationsPage extends Component {
    constructor(props) {
        super(props);

        this.state = {
            tab: {
                headers: [{ text: 'Policies', disabled: false }, { text: 'Compliance', disabled: false }]
            }
        };
    }

    render() {
        return (
            <section className="flex flex-1 h-full">
                <Tabs headers={this.state.tab.headers}>
                    <TabContent>
                        <PoliciesPage />
                    </TabContent>
                    <TabContent>
                        <CompliancePage />
                    </TabContent>
                </Tabs>
            </section>
        );
    }
}

export default ViolationsPage;
