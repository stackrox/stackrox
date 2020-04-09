import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { Edit, Check } from 'react-feather';

import Button from 'Components/Button';

function SectionHeaderInput({ header }) {
    const [isEditing, setIsEditing] = useState(false);
    function editHandler() {
        setIsEditing(!isEditing);
    }

    return (
        <div className="flex flex-1 justify-between items-center">
            {!isEditing && (
                <>
                    <span className="p-2 text-base-600 font-700">{header}</span>
                    <div className="hover:bg-base-400">
                        <Button
                            icon={<Edit className="w-5 h-5" />}
                            onClick={editHandler}
                            className="p-2"
                        />
                    </div>
                </>
            )}
            {isEditing && (
                <>
                    <input value={header} className="p-2 w-full bg-base-200" />
                    <div className="hover:bg-base-400">
                        <Button
                            icon={<Check className="w-5 h-5" />}
                            onClick={editHandler}
                            className="p-2"
                        />
                    </div>
                </>
            )}
        </div>
    );
}

SectionHeaderInput.propTypes = {
    header: PropTypes.string.isRequired
};

export default SectionHeaderInput;
