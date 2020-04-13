import React from 'react';
import PropTypes from 'prop-types';
import ReduxToggleField from 'Components/forms/ReduxToggleField';
import CollapsibleText from 'Components/CollapsibleText';

const ConfigTelemetryDetailWidget = ({ config, editable }) => {
    return (
        <div className="px-3 w-full h-full" data-testid="view-telemetry-config">
            <aside className="bg-base-100 border-base-200 shadow h-full">
                <div className="py-2 px-4 border-b border-base-300 text-base-600 font-700 text-lg capitalize flex justify-between items-center h-10">
                    Online Telemetry Data Collection {getTextOrToggle(config, editable)}
                </div>

                <div className="flex flex-col pt-2 pb-4 px-4 w-full">
                    <div className="w-full pr-4 whitespace-pre-wrap leading-normal">
                        <p className="py-2 text-base-600">
                            Online telemetry data collection allows StackRox to better utilize
                            anonymized information to enhance your user experience.
                        </p>
                        <CollapsibleText
                            expandText="Learn more..."
                            collapseText="Show less"
                            initiallyExpanded={false}
                        >
                            <p className="py-2 text-base-600 font-600">
                                By consenting to online data collection, you allow StackRox to store
                                and perform analytics on data that arises from the usage and
                                operation of the StackRox Kubernetes Security Platform. This data
                                may contain both operational metrics of the platform itself, as well
                                as information about the environment(s) in which it is being used.
                                While the data is associated with your account, we do not collect
                                any information pertaining to the purpose of these environments; in
                                particular, we will never collect the names of nodes, workloads or
                                non-default namespaces.
                            </p>
                            <p className="py-2 text-base-600 font-600">
                                You can revoke your consent to online telemetry data collection at
                                any time. If you wish to request the deletion of already collected
                                data, please contact our Customer Success team.
                            </p>
                        </CollapsibleText>
                    </div>
                </div>
            </aside>
        </div>
    );
};

function getTextOrToggle(config, editable) {
    if (editable) {
        return <ReduxToggleField name="telemetryConfig.enabled" />;
    }
    return (
        <div data-testid="telemetry-state">{config && config.enabled ? 'enabled' : 'disabled'}</div>
    );
}

ConfigTelemetryDetailWidget.propTypes = {
    config: PropTypes.shape({
        enabled: PropTypes.bool
    }).isRequired,
    editable: PropTypes.bool.isRequired
};

export default ConfigTelemetryDetailWidget;
