import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';

import Fields from 'Containers/Policies/Wizard/Details/Fields';
import ConfigurationFields from 'Containers/Policies/Wizard/Details/ConfigurationFields';

export function Panel({ wizardPolicy }) {
    if (!wizardPolicy) return null;
    return (
        <div className="flex flex-col w-full bg-primary-100 overflow-auto pb-5">
            <Fields policy={wizardPolicy} />
            <ConfigurationFields policy={wizardPolicy} />
        </div>
    );
}

Panel.propTypes = {
    wizardPolicy: PropTypes.shape({}).isRequired
};

const mapStateToProps = createStructuredSelector({
    wizardPolicy: selectors.getWizardPolicy
});

export default connect(mapStateToProps)(Panel);
