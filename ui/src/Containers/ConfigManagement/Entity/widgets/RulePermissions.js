import React from 'react';
import Widget from 'Components/Widget';

import NoResultsMessage from 'Components/NoResultsMessage';
import ScopedPermissions from './ScopedPermissions';

const RulePermissions = ({ rules, ...rest }) => {
    const permissionsMap = rules.reduce((acc, curr) => {
        curr.verbs.forEach(verb => {
            acc[verb] = [...(acc[verb] || []), ...curr.resources, ...curr.nonResourceUrls];
        });
        return acc;
    }, {});
    const permissions = Object.keys(permissionsMap).map(key => {
        const values = permissionsMap[key];
        return { key, values };
    });

    let content = <ScopedPermissions permissions={permissions} />;
    const header = `${
        permissions.length > 0 ? permissions.length : ''
    } Permissions across this cluster`;
    if (!permissions.length)
        content = <NoResultsMessage message="No Permissions" className="p-6" />;
    return (
        <Widget header={header} {...rest}>
            <div className="w-full">{content}</div>
        </Widget>
    );
};

export default RulePermissions;
