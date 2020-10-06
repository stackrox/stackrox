import React from 'react';
import PropTypes from 'prop-types';
import Panel from 'Components/Panel';

import DragAndDrop from './Tiles/DragAndDrop';
import Generate from './Tiles/Generate';
import ViewActive from './Buttons/ViewActive';

const Creator = ({ onClose }) => {
    const header = 'SELECT AN OPTION';
    return (
        <div data-testid="network-creator-panel" className="h-full w-full shadow-md bg-base-200">
            <Panel header={header} onClose={onClose} headerComponents={<ViewActive />}>
                <div className="flex h-full w-full flex-col p-4 pb-0">
                    <Generate />
                    <div className="w-full my-5 text-center flex items-center flex-shrink-0">
                        <div className="h-px bg-base-400 w-full" />
                        <span className="relative px-2 font-700">OR</span>
                        <div className="h-px bg-base-400 w-full" />
                    </div>
                    <DragAndDrop />
                </div>
            </Panel>
        </div>
    );
};

Creator.propTypes = {
    onClose: PropTypes.func.isRequired,
};

export default Creator;
