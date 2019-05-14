import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import Panel from 'Components/Panel';
import LoadingSection from 'Components/LoadingSection';
import { Details as EnforcementDetails } from 'Containers/Violations/Enforcement/Details';
import DeploymentDetails from '../Risk/DeploymentDetails';
import ViolationsDetails from './ViolationsDetails';
import { Panel as PolicyDetails } from '../Policies/Wizard/Details/Panel';

const ViolationsPanelTabs = ({ alert }) => {
    const riskPanelTabs = [
        { text: 'Violation' },
        { text: 'Enforcement' },
        { text: 'Deployment' },
        { text: 'Policy' }
    ];
    const isLoading = !alert; // TODO: poor-man loading check until a proper one in place
    const content = isLoading ? (
        <LoadingSection />
    ) : (
        <Tabs headers={riskPanelTabs}>
            <TabContent>
                <div className="flex flex-1 flex-col">
                    <ViolationsDetails
                        violations={alert.violations}
                        processViolation={alert.processViolation}
                    />
                </div>
            </TabContent>
            <TabContent>
                <div className="flex flex-1 flex-col">
                    <EnforcementDetails listAlert={alert} />
                </div>
            </TabContent>
            <TabContent>
                <div className="flex flex-1 flex-col">
                    <DeploymentDetails deployment={alert.deployment} />
                </div>
            </TabContent>
            <TabContent>
                <div className="flex flex-1 flex-col">
                    <PolicyDetails wizardPolicy={alert.policy} />
                </div>
            </TabContent>
        </Tabs>
    );
    return content;
};

const ViolationsPanel = ({ alert, onClose }) => {
    if (!alert || !alert.policy || !alert.deployment) return null; // TODO: show loading

    const header = `${alert.deployment.name} (${alert.deployment.id})`;
    return (
        <Panel
            header={header}
            className="bg-primary-200 z-10 w-full h-full absolute pin-r pin-t min-w-72 md:w-1/2 md:relative"
            onClose={onClose}
        >
            <ViolationsPanelTabs alert={alert} />
        </Panel>
    );
};

ViolationsPanel.propTypes = {
    alert: PropTypes.shape({
        id: PropTypes.string
    }),
    onClose: PropTypes.func.isRequired
};

ViolationsPanel.defaultProps = {
    alert: null
};

const getAlert = createSelector(
    [selectors.getFilteredAlerts, (state, props) => props.alertId],
    (alerts, alertId) => alerts.find(alert => alert.id === alertId)
);

const mapStateToProps = createStructuredSelector({
    alert: getAlert
});

export default connect(mapStateToProps)(ViolationsPanel);
