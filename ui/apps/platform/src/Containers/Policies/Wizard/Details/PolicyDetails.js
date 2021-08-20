import React from 'react';
import PropTypes from 'prop-types';

import Fields from 'Containers/Policies/Wizard/Details/Fields';
import ConfigurationFields from 'Containers/Policies/Wizard/Details/ConfigurationFields';
import BooleanPolicySection from 'Containers/Policies/Wizard/Form/BooleanPolicySection';
import {
    FormSection,
    FormSectionBody,
} from 'Containers/Policies/Wizard/Form/PolicyDetailsForm/FormSection';
import MitreAttackVectors from 'Containers/MitreAttackVectors';

function PolicyDetails({ policy }) {
    if (!policy) {
        return null;
    }

    // If the policy version is not set, that means this is a legacy policy.
    // Legacy policies are only displayed when we display old alerts.
    const isLegacyPolicy = !policy.policyVersion;

    return (
        <div className="w-full h-full">
            <div className="flex flex-col w-full overflow-auto pb-5">
                <Fields policy={policy} />
                {!isLegacyPolicy && <BooleanPolicySection readOnly initialValues={policy} />}
                {isLegacyPolicy && <ConfigurationFields policy={policy} />}
                <div className="p-4">
                    <FormSection dataTestId="mitreAttackVectorDetails" headerText="MITRE ATT&CK">
                        <FormSectionBody>
                            <MitreAttackVectors policyId={policy.id} />
                        </FormSectionBody>
                    </FormSection>
                </div>
            </div>
        </div>
    );
}

PolicyDetails.propTypes = {
    policy: PropTypes.shape({
        id: PropTypes.string,
        name: PropTypes.string,
        policyVersion: PropTypes.string,
    }).isRequired,
};

export default PolicyDetails;
