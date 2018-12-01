import React from 'react';
import PropTypes from 'prop-types';

import { NO_ACCESS, READ_ACCESS, READ_WRITE_ACCESS } from 'constants/accessControl';
import { accessControl } from 'messages/common';

import ReduxSelectField from 'Components/forms/ReduxSelectField';
import ReadWriteIcon from './ReadWriteIcon';

const AccessField = ({ input, resourceToAccess, resourceName, isEditing }) => {
    const options = [
        { label: accessControl.NO_ACCESS, value: NO_ACCESS },
        { label: accessControl.READ_ACCESS, value: READ_ACCESS },
        { label: accessControl.READ_WRITE_ACCESS, value: READ_WRITE_ACCESS }
    ];
    const value = input ? input.value : resourceToAccess[resourceName];
    const name = input ? input.name : '';
    return (
        <tr>
            <th className="p-4 border-r border-base-300">{resourceName}</th>
            <th className="p-4">
                <ReadWriteIcon value={value} type="READ" />
            </th>
            <th className="p-4">
                <ReadWriteIcon value={value} type="WRITE" />
            </th>
            {isEditing && (
                <th className="p-4">
                    <ReduxSelectField name={name} options={options} />
                </th>
            )}
        </tr>
    );
};

AccessField.propTypes = {
    input: PropTypes.shape({
        value: PropTypes.oneOfType([PropTypes.array, PropTypes.string]),
        name: PropTypes.string,
        onChange: PropTypes.func
    }),
    resourceToAccess: PropTypes.shape({}).isRequired,
    resourceName: PropTypes.string.isRequired,
    isEditing: PropTypes.bool.isRequired
};

AccessField.defaultProps = {
    input: null
};

export default AccessField;
