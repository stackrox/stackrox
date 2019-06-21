import React from 'react';
import Widget from 'Components/Widget';

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
    if (!permissions.length) return null;
    const content = <ScopedPermissions permissions={permissions} />;
    const header = `${permissions.length} Permissions across this cluster`;
    return (
        <Widget header={header} {...rest}>
            <div className="w-full">{content}</div>
        </Widget>
    );
};

export default RulePermissions;
