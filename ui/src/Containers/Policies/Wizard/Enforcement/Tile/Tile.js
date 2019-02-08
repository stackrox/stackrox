import React from 'react';
import PropTypes from 'prop-types';

import OnOffSwitch from 'Containers/Policies/Wizard/Enforcement/Tile/OnOffSwitch';
import Descriptor from 'Containers/Policies/Wizard/Enforcement/Tile/Descriptor';
import Visual from 'Containers/Policies/Wizard/Enforcement/Tile/Visual';

const Tile = props => {
    const { enabled } = props;
    return (
        <div className={`p-3 w-full ${!enabled && 'opacity-50'}`}>
            <div className="flex w-full bg-primary-100 border-3 rounded border-primary-300">
                <div className="flex flex-col h-full border-r-3 border-primary-200">
                    <div className="px-5">
                        <Visual image={props.enforcement.image} label={props.enforcement.label} />
                    </div>
                    <div className="flex border-b border-primary-400" />
                    <div className="px-5">
                        <OnOffSwitch
                            enabled={props.enabled}
                            applied={props.applied}
                            onClick={props.onOffAction}
                        />
                    </div>
                </div>
                <Descriptor
                    header={props.enforcement.header}
                    description={props.enforcement.description}
                />
            </div>
        </div>
    );
};

Tile.propTypes = {
    enabled: PropTypes.bool.isRequired,
    applied: PropTypes.bool.isRequired,
    enforcement: PropTypes.shape({
        image: PropTypes.string.isRequired,
        label: PropTypes.string.isRequired,
        header: PropTypes.string.isRequired,
        description: PropTypes.string.isRequired
    }).isRequired,
    onOffAction: PropTypes.func.isRequired
};

export default Tile;
