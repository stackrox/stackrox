import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';

import { actions as graphActions } from 'reducers/network/graph';
import { fetchCIDRBlocks } from 'services/NetworkService';
import Panel from 'Components/Panel';
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
        <Panel
            header="Segment External Entities by CIDR Blocks"
            onClose={onClose}
            bodyClassName="flex flex-col bg-base-100"
            id="network-cidr-form"
        >
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
        </Panel>
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
