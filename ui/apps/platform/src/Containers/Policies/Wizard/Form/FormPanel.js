import React from 'react';
import PropTypes from 'prop-types';

import CloseButton from 'Components/CloseButton';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';
import FormButtons from 'Containers/Policies/Wizard/Form/FormButtons';
import PolicyDetailsForm from 'Containers/Policies/Wizard/Form/PolicyDetailsForm';
import FormMessages from './FormMessages';

function FormPanel({ header, policyDetailsFormFields, onClose, initialValues }) {
    return (
        <PanelNew test id="side-panel">
            <PanelHead>
                <PanelTitle isUpperCase testid="side-panel-header" text={header} />
                <PanelHeadEnd>
                    <FormButtons />
                    <CloseButton onClose={onClose} className="border-base-400 border-l" />
                </PanelHeadEnd>
            </PanelHead>
            <PanelBody>
                <FormMessages />
                <PolicyDetailsForm
                    initialValues={initialValues}
                    formFields={policyDetailsFormFields}
                />
            </PanelBody>
        </PanelNew>
    );
}

FormPanel.propTypes = {
    header: PropTypes.string,
    policyDetailsFormFields: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    onClose: PropTypes.func.isRequired,
    initialValues: PropTypes.shape({}).isRequired,
};

FormPanel.defaultProps = {
    header: '',
};

export default FormPanel;
