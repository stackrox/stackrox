import React from 'react';
import { MinusCircleIcon, PlusCircleIcon } from '@patternfly/react-icons';

const getVerbIcon = (key) => {
    switch (key) {
        case 'create':
        case 'update':
            // Keep success color for backward compatibility, although it is questionable semantically.
            return <PlusCircleIcon color="var(--pf-global--success-color--100)" />;
        case 'delete':
            return <MinusCircleIcon color="var(--pf-global--danger-color--100)" />;
        default:
            return null;
    }
};

const ScopedPermissions = ({ permissions }) => {
    return permissions.map((datum) => {
        return (
            <div className="flex border-b border-base-300" key={datum.key}>
                <div className="w-43 border-r border-base-300 p-3 text-sm flex">
                    <span className="pf-u-display-inline-flex pf-u-align-items-center">
                        <span className="w-4 pr-3">{getVerbIcon(datum.key)}</span>
                        <span>
                            {datum.key === '*' ? (
                                '* (All verbs)'
                            ) : (
                                <span className="capitalize">{datum.key}</span>
                            )}
                            :
                        </span>
                    </span>
                </div>
                <div className="w-full p-3 text-sm leading-normal">
                    {datum.values.includes('*') ? '* (All resources)' : datum.values.join(', ')}
                </div>
            </div>
        );
    });
};

export default ScopedPermissions;
