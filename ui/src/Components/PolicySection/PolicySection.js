import React from 'react';
import PropTypes from 'prop-types';
import { useDrop } from 'react-dnd';
import { Trash2 } from 'react-feather';

import DRAG_DROP_TYPES from 'constants/dragDropTypes';
import Button from 'Components/Button';
import SectionHeaderInput from 'Components/SectionHeaderInput';

function PolicySection({ header, removeFieldHandler }) {
    const [, drop] = useDrop({
        accept: DRAG_DROP_TYPES.KEY,
        drop: () => {}
    });
    return (
        <div className="bg-base-300 border-2 border-base-100 rounded">
            <div className="flex justify-between items-center border-b-2 border-base-400">
                <SectionHeaderInput header={header} />
                <Button
                    onClick={removeFieldHandler}
                    icon={<Trash2 className="w-5 h-5" />}
                    className="p-2 border-l-2 border-base-400 hover:bg-base-400"
                />
            </div>
            <div className="p-2">
                <div>PolicySection content</div>
                <div
                    ref={drop}
                    className="bg-base-200 rounded border-2 border-base-300 border-dashed flex font-700 justify-center p-3 text-base-500 text-sm uppercase"
                >
                    Drop a policy field inside
                </div>
            </div>
        </div>
    );
}

PolicySection.propTypes = {
    header: PropTypes.string.isRequired,
    removeFieldHandler: PropTypes.func.isRequired
};

export default PolicySection;
