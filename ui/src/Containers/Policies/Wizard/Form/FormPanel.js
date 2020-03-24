import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import Panel from 'Components/Panel';
import FormButtons from 'Containers/Policies/Wizard/Form/FormButtons';
import FieldGroupCards from 'Containers/Policies/Wizard/Form/FieldGroupCards';

function FormPanel({ header, initialValues, fieldGroups, wizardPolicy, onClose }) {
    if (!wizardPolicy) return null;

    return (
        <Panel
            header={header}
            headerComponents={<FormButtons />}
            onClose={onClose}
            id="side-panel"
            className="w-1/2"
        >
            <div className="w-full h-full">
                <form className="flex flex-col w-full overflow-auto pb-5">
                    <FieldGroupCards initialValues={initialValues} fieldGroups={fieldGroups} />
                </form>
            </div>
        </Panel>
    );
}

FormPanel.propTypes = {
    header: PropTypes.string,
    initialValues: PropTypes.shape({}).isRequired,
    fieldGroups: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    wizardPolicy: PropTypes.shape({
        name: PropTypes.string
    }).isRequired,
    onClose: PropTypes.func.isRequired
};

FormPanel.defaultProps = {
    header: ''
};

const mapStateToProps = createStructuredSelector({
    wizardPolicy: selectors.getWizardPolicy
});

export default connect(mapStateToProps)(FormPanel);
