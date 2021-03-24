import React from 'react';
import PropTypes from 'prop-types';

import CloseButton from 'Components/CloseButton';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';
import BooleanPolicySection from 'Containers/Policies/Wizard/Form/BooleanPolicySection';
import CriteriaFormButtons from './CriteriaFormButtons';
import FormMessages from '../FormMessages';

function CriteriaFormPanel({ header, onClose }) {
    return (
        <PanelNew testid="side-panel">
            <PanelHead>
                <PanelTitle isUpperCase testid="side-panel" text={header} />
                <PanelHeadEnd>
                    <CriteriaFormButtons />
                    <CloseButton onClose={onClose} className="border-base-400 border-l" />
                </PanelHeadEnd>
            </PanelHead>
            <PanelBody>
                <FormMessages />
                <form className="flex flex-col w-full overflow-auto h-full">
                    <BooleanPolicySection />
                </form>
            </PanelBody>
        </PanelNew>
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
