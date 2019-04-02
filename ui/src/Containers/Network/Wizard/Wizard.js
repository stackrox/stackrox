import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';

import Details from './Details/Details';
import Creator from './Creator/Creator';
import Simulator from './Simulator/Simulator';
import NamespaceDetails from './NamespaceDetails/NamespaceDetails';
import NodesUpdateSection from '../Graph/Overlays/NodesUpdateSection';
import ZoomButtons from '../Graph/Overlays/ZoomButtons';

function Wizard(props) {
    const width = props.wizardOpen ? 'w-1/3' : 'w-0';

    return (
        <div className={`${width} h-full absolute pin-r bg-primary-200 shadow-lg`}>
            <NodesUpdateSection />
            <Details />
            <Creator />
            <Simulator />
            <NamespaceDetails />
            <ZoomButtons />
        </div>
    );
}

Wizard.propTypes = {
    wizardOpen: PropTypes.bool.isRequired
};

const mapStateToProps = createStructuredSelector({
    wizardOpen: selectors.getNetworkWizardOpen
});

export default connect(mapStateToProps)(Wizard);
