import React, { useEffect, useState } from 'react';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';

import { fetchCIDRBlocks } from 'services/NetworkService';
import Panel from 'Components/Panel';
import Loader from 'Components/Loader';
import CIDRForm from './CIDRForm';

const CIDRPanel = ({ selectedClusterId, onClose }) => {
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
            {CIDRBlocks?.entities?.length >= 0 ? (
                <CIDRForm rows={CIDRBlocks} clusterId={selectedClusterId} onClose={onClose} />
            ) : (
                <Loader />
            )}
        </Panel>
    );
};

CIDRPanel.defaultProps = {
    selectedClusterId: '',
};

const mapStateToProps = createStructuredSelector({
    selectedClusterId: selectors.getSelectedNetworkClusterId,
});

export default connect(mapStateToProps)(CIDRPanel);
