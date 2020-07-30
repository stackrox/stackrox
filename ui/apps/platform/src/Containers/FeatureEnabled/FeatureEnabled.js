import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';

import { isBackendFeatureFlagEnabled } from 'utils/featureFlags';

const FeatureEnabled = ({ featureFlags, featureFlag, children }) => {
    const featureEnabled = isBackendFeatureFlagEnabled(featureFlags, featureFlag, false);

    return children({ featureEnabled });
};

const mapStateToProps = createStructuredSelector({
    featureFlags: selectors.getFeatureFlags,
});

export default connect(mapStateToProps, null)(FeatureEnabled);
