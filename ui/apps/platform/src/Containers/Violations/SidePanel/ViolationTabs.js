import React from 'react';
import PropTypes from 'prop-types';

import Tabs from 'Components/Tabs';
import Tab from 'Components/Tab';
import EnforcementDetails from 'Containers/Violations/Enforcement/Details';
import { preFormatPolicyFields } from 'Containers/Policies/Wizard/Form/utils';
import DeploymentDetails from '../../Risk/DeploymentDetails';
import ViolationsDetails from './ViolationsDetails';
import PolicyDetails from '../../Policies/Wizard/Details/PolicyDetails';

function ViolationTabs({ alert }) {
    const initialValuesForPolicy = preFormatPolicyFields(alert.policy);
    const tabHeaders = [{ text: 'Violation' }];
    if (alert.enforcement) {
        tabHeaders.push({ text: 'Enforcement' });
    }
    if (alert.deployment) {
        tabHeaders.push({ text: 'Deployment' });
    }
    tabHeaders.push({ text: 'Policy' });
    return (
        <Tabs headers={tabHeaders}>
            <Tab extraClasses="bg-base-0">
                <div className="flex flex-1 flex-col">
                    <ViolationsDetails
                        violationId={alert.id}
                        violations={alert.violations}
                        processViolation={alert.processViolation}
                        lifecycleStage={alert.lifecycleStage}
                    />
                </div>
            </Tab>
            {alert.enforcement && (
                <Tab extraClasses="bg-base-0">
                    <div className="flex flex-1 flex-col">
                        <EnforcementDetails alert={alert} />
                    </div>
                </Tab>
            )}
            {alert.deployment && (
                <Tab extraClasses="bg-base-0">
                    <div className="flex flex-1 flex-col">
                        <DeploymentDetails deployment={alert.deployment} />
                    </div>
                </Tab>
            )}
            <Tab extraClasses="bg-base-0">
                <div className="flex flex-1 flex-col">
                    <PolicyDetails policy={initialValuesForPolicy} />
                </div>
            </Tab>
        </Tabs>
    );
}

ViolationTabs.propTypes = {
    alert: PropTypes.shape({
        id: PropTypes.string.isRequired,
        lifecycleStage: PropTypes.string.isRequired,
        violations: PropTypes.arrayOf(PropTypes.object),
        processViolation: PropTypes.shape({}),
        deployment: PropTypes.shape({}),
        enforcement: PropTypes.shape({}),
        policy: PropTypes.shape({}),
    }).isRequired,
};

export default React.memo(ViolationTabs);
