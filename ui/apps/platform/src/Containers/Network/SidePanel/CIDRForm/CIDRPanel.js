import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';

import { actions as graphActions } from 'reducers/network/graph';
import { fetchCIDRBlocks } from 'services/NetworkService';
import CloseButton from 'Components/CloseButton';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';
import Loader from 'Components/Loader';
import DefaultCIDRToggle from './DefaultCIDRToggle';
import CIDRForm from './CIDRForm';

const CIDRPanel = ({ selectedClusterId, updateNetworkNodes, onClose }) => {
    const [CIDRBlocks, setCIDRBlocks] = useState();

    useEffect(() => {
        fetchCIDRBlocks(selectedClusterId).then(({ response }) => {
            const entities = response.entities.map(({ info }) => {
                const { externalSource, id } = info;
                const { name, cidr } = externalSource;
                return {
                    entity: {
                        cidr,
                        name,
                        id,
                    },
                };
            });
            setCIDRBlocks({ entities });
        });
    }, [selectedClusterId]);

    return (
        <PanelNew testid="network-cidr-form">
            <PanelHead>
                <PanelTitle
                    testid="network-cidr-form-header"
                    text="Segment External Entities by CIDR Blocks"
                />
                <PanelHeadEnd>
                    <CloseButton onClose={onClose} className="border-base-400 border-l" />
                </PanelHeadEnd>
            </PanelHead>
            <PanelBody>
                <div className="flex flex-col h-full">
                    <DefaultCIDRToggle />
                    {CIDRBlocks?.entities?.length >= 0 ? (
                        <CIDRForm
                            rows={CIDRBlocks}
                            clusterId={selectedClusterId}
                            onClose={onClose}
                            updateNetworkNodes={updateNetworkNodes}
                        />
                    ) : (
                        <Loader />
                    )}
                </div>
            </PanelBody>
        </PanelNew>
    );
};

CIDRPanel.propTypes = {
    selectedClusterId: PropTypes.string,
    onClose: PropTypes.func.isRequired,
    updateNetworkNodes: PropTypes.func.isRequired,
};

CIDRPanel.defaultProps = {
    selectedClusterId: '',
};

const mapStateToProps = createStructuredSelector({
    selectedClusterId: selectors.getSelectedNetworkClusterId,
});

const mapDispatchToProps = {
    updateNetworkNodes: graphActions.updateNetworkNodes,
};

export default connect(mapStateToProps, mapDispatchToProps)(CIDRPanel);
