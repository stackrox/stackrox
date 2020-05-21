import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';

import Fields from 'Containers/Policies/Wizard/Details/Fields';
import ConfigurationFields from 'Containers/Policies/Wizard/Details/ConfigurationFields';
import BooleanPolicySection from 'Containers/Policies/Wizard/Form/BooleanPolicySection';
import { knownBackendFlags, isBackendFeatureFlagEnabled } from 'utils/featureFlags';

function PolicyDetails({ initialValues, featureFlags }) {
    if (!initialValues) return null;

    const BPLenabled = isBackendFeatureFlagEnabled(
        featureFlags,
        knownBackendFlags.ROX_BOOLEAN_POLICY_LOGIC,
        false
    );

    const PolicyConfigurationSection = BPLenabled ? (
        <BooleanPolicySection readOnly initialValues={initialValues} />
    ) : (
        <ConfigurationFields policy={initialValues} />
    );

    return (
        <div className="w-full h-full">
            <div className="flex flex-col w-full overflow-auto pb-5">
                <Fields policy={initialValues} />
                {PolicyConfigurationSection}
            </div>
        </div>
    );
}

PolicyDetails.propTypes = {
    initialValues: PropTypes.shape({
        name: PropTypes.string,
    }).isRequired,
    featureFlags: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
};

const mapStateToProps = createStructuredSelector({
    featureFlags: selectors.getFeatureFlags,
});

export default connect(mapStateToProps, null)(PolicyDetails);
