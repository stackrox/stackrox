import React from 'react';
import PropTypes from 'prop-types';

import CloseButton from 'Components/CloseButton';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';
import DetailsButtons from './DetailsButtons';
import PolicyDetails from './PolicyDetails';

function PolicyDetailsPanel({ header, onClose, policy }) {
    return (
        <PanelNew testid="side-panel">
            <PanelHead>
                <PanelTitle isUpperCase breakAll={false} testid="side-panel-header" text={header} />
                <PanelHeadEnd>
                    <DetailsButtons />
                    <CloseButton onClose={onClose} className="border-base-400 border-l" />
                </PanelHeadEnd>
            </PanelHead>
            <PanelBody>
                <PolicyDetails policy={policy} />
            </PanelBody>
        </PanelNew>
    );
}

PolicyDetailsPanel.propTypes = {
    header: PropTypes.string,
    onClose: PropTypes.func.isRequired,
    policy: PropTypes.shape({}).isRequired,
};

PolicyDetailsPanel.defaultProps = {
    header: '',
};

export default PolicyDetailsPanel;
