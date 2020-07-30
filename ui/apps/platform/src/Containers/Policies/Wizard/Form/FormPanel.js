import React from 'react';
import PropTypes from 'prop-types';

import Panel from 'Components/Panel';
import FormButtons from 'Containers/Policies/Wizard/Form/FormButtons';
import FieldGroupCards from 'Containers/Policies/Wizard/Form/FieldGroupCards';
import FormMessages from './FormMessages';

function FormPanel({ header, fieldGroups, onClose, initialValues }) {
    return (
        <Panel
            header={header}
            headerComponents={<FormButtons />}
            onClose={onClose}
            id="side-panel"
            className="w-1/2"
        >
            <div className="w-full h-full">
                <FormMessages />
                <form className="flex flex-col w-full overflow-auto pb-5">
                    <FieldGroupCards initialValues={initialValues} fieldGroups={fieldGroups} />
                </form>
            </div>
        </Panel>
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
