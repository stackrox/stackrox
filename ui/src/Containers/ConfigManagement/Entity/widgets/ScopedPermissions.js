import React from 'react';

const colors = ['primary', 'secondary', 'tertiary'];

const getLabelColor = key => {
    let color = '';
    switch (key) {
        case 'create':
        case 'update':
            color = 'success';
            break;
        case 'delete':
            color = 'alert';
            break;
        default:
            color = colors[Math.floor(Math.random() * colors.length)];
            break;
    }
    return color;
};

const ScopedPermissions = ({ permissions }) => {
    let content = [];
    const { length } = permissions;
    if (length) {
        content = permissions.map(datum => {
            const colorClass = getLabelColor(datum.key);
            const permissionKeyClass = `rounded bg-${colorClass}-200 text-${colorClass}-700 border border-${colorClass}-300 px-2`;
            return (
                <div className="flex border-b border-base-300" key={datum.key}>
                    <div className="min-w-48 border-r border-base-300 p-4 text-sm capitalize">
                        <span className={permissionKeyClass}>{datum.key}:</span>
                    </div>
                    <div className="font-500 p-4 text-primary-800 text-sm">
                        {datum.values.join(', ')}
                    </div>
                </div>
            );
        });
    }
    return content;
};

export default ScopedPermissions;
