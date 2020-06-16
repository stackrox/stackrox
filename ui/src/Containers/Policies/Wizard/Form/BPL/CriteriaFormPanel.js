import React from 'react';
import PropTypes from 'prop-types';

import { knownBackendFlags } from 'utils/featureFlags';
import Panel from 'Components/Panel';
import FeatureEnabled from 'Containers/FeatureEnabled';
import BooleanPolicySection from 'Containers/Policies/Wizard/Form/BooleanPolicySection';
import CriteriaFormButtons from './CriteriaFormButtons';
import FormMessages from '../FormMessages';

function CriteriaFormPanel({ header, onClose }) {
    return (
        <Panel
            header={header}
            headerComponents={<CriteriaFormButtons />}
            onClose={onClose}
            id="side-panel"
            className="w-1/2"
        >
            <div className="w-full h-full">
                <FormMessages />
                <form className="flex flex-col w-full overflow-auto h-full">
                    <FeatureEnabled featureFlag={knownBackendFlags.ROX_BOOLEAN_POLICY_LOGIC}>
                        {({ featureEnabled }) => {
                            return featureEnabled && <BooleanPolicySection />;
                        }}
                    </FeatureEnabled>
                </form>
            </div>
        </Panel>
    );
}

CriteriaFormPanel.propTypes = {
    header: PropTypes.string,
    onClose: PropTypes.func.isRequired,
};

CriteriaFormPanel.defaultProps = {
    header: '',
};

export default CriteriaFormPanel;
