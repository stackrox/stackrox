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
        content = permissions.map((datum, i) => {
            const colorClass = getLabelColor(datum.key);
            const permissionKeyClass = `rounded bg-${colorClass}-200 text-${colorClass}-700 ${
                i !== permissions.length - 1 ? `border border-${colorClass}-300` : ''
            } px-2 py-1 self-center`;
            return (
                <div className="flex border-b border-base-300" key={datum.key}>
                    <div className="w-43 border-r border-base-300 px-3 text-sm flex">
                        <div className={permissionKeyClass}>
                            {datum.key === '*' ? (
                                '* (All verbs)'
                            ) : (
                                <span className="capitalize">{datum.key}</span>
                            )}
                            :
                        </div>
                    </div>
                    <div className="w-full font-500 p-3 text-primary-800 text-sm leading-normal">
                        {datum.values.includes('*') ? '* (All resources)' : datum.values.join(', ')}
                    </div>
                </div>
            );
        });
    }
    return content;
};

export default ScopedPermissions;
