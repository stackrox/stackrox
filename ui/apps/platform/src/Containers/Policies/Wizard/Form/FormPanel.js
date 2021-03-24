import React from 'react';
import PropTypes from 'prop-types';

import CloseButton from 'Components/CloseButton';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';
import FormButtons from 'Containers/Policies/Wizard/Form/FormButtons';
import FieldGroupCards from 'Containers/Policies/Wizard/Form/FieldGroupCards';
import FormMessages from './FormMessages';

function FormPanel({ header, fieldGroups, onClose, initialValues }) {
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
                <form className="flex flex-col w-full overflow-auto pb-5">
                    <FieldGroupCards initialValues={initialValues} fieldGroups={fieldGroups} />
                </form>
            </PanelBody>
        </PanelNew>
    );
}

FormPanel.propTypes = {
    header: PropTypes.string,
    fieldGroups: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    onClose: PropTypes.func.isRequired,
    initialValues: PropTypes.shape({}).isRequired,
};

FormPanel.defaultProps = {
    header: '',
};

export default FormPanel;
