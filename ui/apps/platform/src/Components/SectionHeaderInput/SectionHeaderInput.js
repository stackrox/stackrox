import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { Edit, Check } from 'react-feather';

import Button from 'Components/Button';

function SectionHeaderInput({ input, readOnly }) {
    const [isEditing, setIsEditing] = useState(false);
    function editHandler() {
        setIsEditing(!isEditing);
    }
    const { value, onChange } = input;

    return (
        <div
            className="flex flex-1 justify-between items-center capitalize"
            data-testid="section-header"
        >
            {!isEditing && (
                <>
                    <span className="p-2 text-base-600 font-700">{value}</span>
                    {!readOnly && (
                        <div className="hover:bg-base-400">
                            <Button
                                icon={<Edit className="w-5 h-5" />}
                                onClick={editHandler}
                                className="p-2"
                                dataTestId="section-header-edit-btn"
                            />
                        </div>
                    )}
                </>
            )}
            {isEditing && (
                <>
                    <input
                        value={value}
                        onChange={onChange}
                        className="p-2 w-full bg-base-200"
                        aria-label="Policy Section Header Input"
                    />
                    <div className="hover:bg-base-400">
                        <Button
                            icon={<Check className="w-5 h-5" />}
                            onClick={editHandler}
                            className="p-2"
                            dataTestId="section-header-confirm-btn"
                        />
                    </div>
                </>
            )}
        </div>
    );
}

SectionHeaderInput.propTypes = {
    input: PropTypes.shape({
        value: PropTypes.string,
        onChange: PropTypes.func.isRequired,
    }).isRequired,
    readOnly: PropTypes.bool,
};

SectionHeaderInput.defaultProps = {
    readOnly: false,
};

export default SectionHeaderInput;
