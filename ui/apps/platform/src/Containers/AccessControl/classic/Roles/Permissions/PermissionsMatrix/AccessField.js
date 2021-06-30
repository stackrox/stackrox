import React from 'react';
import PropTypes from 'prop-types';

import { NO_ACCESS, READ_ACCESS, READ_WRITE_ACCESS } from 'constants/accessControl';
import { accessControl } from 'messages/common';

import ReduxSelectField from 'Components/forms/ReduxSelectField';
import { ReadAccessIcon, WriteAccessIcon } from '../../../../PermissionSets/AccessIcons';

const AccessField = ({ input, resourceToAccess, resourceName, isEditing }) => {
    const options = [
        { label: accessControl.NO_ACCESS, value: NO_ACCESS },
        { label: accessControl.READ_ACCESS, value: READ_ACCESS },
        { label: accessControl.READ_WRITE_ACCESS, value: READ_WRITE_ACCESS },
    ];
    const accessType = input ? input.value : resourceToAccess[resourceName];
    const name = input ? input.name : '';
    return (
        <tr>
            <td className="border-r border-base-300 text-left font-600 border-b border-base-300">
                <span className="p-2">{resourceName}</span>
            </td>
            <td className="p-2 text-center border-b border-base-300">
                {accessType.read && (accessType.read.join(', ') || 'None')}
                {!accessType.read && <ReadAccessIcon accessLevel={accessType} />}
            </td>
            <td className="p-2 text-center border-b border-base-300">
                {accessType.write && (accessType.write.join(', ') || 'None')}
                {!accessType.write && <WriteAccessIcon accessLevel={accessType} />}
            </td>
            {isEditing && (
                <td className="p-2 border-b border-base-300">
                    <ReduxSelectField name={name} options={options} />
                </td>
            )}
        </tr>
    );
};

AccessField.propTypes = {
    input: PropTypes.shape({
        value: PropTypes.oneOfType([PropTypes.arrayOf(PropTypes.string), PropTypes.string]),
        name: PropTypes.string,
        onChange: PropTypes.func,
    }),
    resourceToAccess: PropTypes.shape({}).isRequired,
    resourceName: PropTypes.string.isRequired,
    isEditing: PropTypes.bool.isRequired,
};

AccessField.defaultProps = {
    input: null,
};

export default AccessField;
