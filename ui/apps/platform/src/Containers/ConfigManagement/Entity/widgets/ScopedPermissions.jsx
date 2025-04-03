import React from 'react';

const ScopedPermissions = ({ permissions }) => {
    return permissions.map((datum) => {
        return (
            <div className="flex border-b border-base-300" key={datum.key}>
                <div className="w-43 border-r border-base-300 px-2 py-3 text-sm flex">
                    {datum.key === '*' ? (
                        '* (All verbs)'
                    ) : (
                        <span className="capitalize">{datum.key}</span>
                    )}
                    :
                </div>
                <div className="w-full px-2 py-3 text-sm leading-normal">
                    {datum.values.includes('*') ? '* (All resources)' : datum.values.join(', ')}
                </div>
            </div>
        );
    });
};

export default ScopedPermissions;
