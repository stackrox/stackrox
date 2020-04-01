import React from 'react';
import PropTypes from 'prop-types';

import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import EnforcementDetails from 'Containers/Violations/Enforcement/Details';
import DeploymentDetails from '../../Risk/DeploymentDetails';
import ViolationsDetails from './ViolationsDetails';
import PolicyDetails from '../../Policies/Wizard/Details/PolicyDetails';

const riskPanelTabs = [
    { text: 'Violation' },
    { text: 'Enforcement' },
    { text: 'Deployment' },
    { text: 'Policy' }
];

function ViolationTabs({ alert }) {
    return (
        <Tabs headers={riskPanelTabs}>
            <TabContent extraClasses="bg-base-0">
                <div className="flex flex-1 flex-col">
                    <ViolationsDetails
                        violationId={alert.id}
                        violations={alert.violations}
                        processViolation={alert.processViolation}
                    />
                </div>
            </TabContent>
            <TabContent extraClasses="bg-base-0">
                <div className="flex flex-1 flex-col">
                    <EnforcementDetails alert={alert} />
                </div>
            </TabContent>
            <TabContent extraClasses="bg-base-0">
                <div className="flex flex-1 flex-col">
                    <DeploymentDetails deployment={alert.deployment} />
                </div>
            </TabContent>
            <TabContent extraClasses="bg-base-0">
                <div className="flex flex-1 flex-col">
                    <PolicyDetails policy={alert.policy} />
                </div>
            </TabContent>
        </Tabs>
    );
}

ViolationTabs.propTypes = {
    alert: PropTypes.shape({
        id: PropTypes.string.isRequired,
        violations: PropTypes.arrayOf(PropTypes.object),
        processViolation: PropTypes.shape({}),
        deployment: PropTypes.shape({}),
        policy: PropTypes.shape({})
    }).isRequired
};

export default React.memo(ViolationTabs);
