import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';

import { preFormatPolicyFields } from 'Containers/Policies/Wizard/Form/utils';
import FieldGroupCards from 'Containers/Policies/Wizard/Form/FieldGroupCards';

function Panel({ wizardPolicy }) {
    return (
        <div className="flex flex-1 flex-col bg-primary-100">
            <form id="dynamic-form">
                <FieldGroupCards initialValues={preFormatPolicyFields(wizardPolicy)} />
            </form>
        </div>
    );
}

Panel.propTypes = {
    wizardPolicy: PropTypes.shape({
        name: PropTypes.string
    }).isRequired
};

const mapStateToProps = createStructuredSelector({
    wizardPolicy: selectors.getWizardPolicy
});

export default connect(mapStateToProps)(Panel);
