import React from 'react';
import PropTypes from 'prop-types';

import Widget from 'Components/Widget';
import NoResultsMessage from 'Components/NoResultsMessage';
import ScopedPermissions from './ScopedPermissions';

const RulePermissions = ({ rules, ...rest }) => {
    let content = <NoResultsMessage message="No Permissions" className="p-6" />;
    let header = 'Permissions across this cluster';
    if (rules && rules.length) {
        const permissionsMap = rules.reduce((acc, curr) => {
            curr.verbs.forEach((verb) => {
                acc[verb] = [...(acc[verb] || []), ...curr.resources, ...curr.nonResourceUrls];
            });
            return acc;
        }, {});
        const permissions = Object.keys(permissionsMap).map((key) => {
            const values = permissionsMap[key];
            return { key, values };
        });

        if (permissions.length > 0) {
            header = `${permissions.length} Permissions across this cluster`;
            content = <ScopedPermissions permissions={permissions} />;
        }
    }

    return (
        <Widget header={header} {...rest}>
            <div className="w-full">{content}</div>
        </Widget>
    );
};

RulePermissions.propTypes = {
    rules: PropTypes.arrayOf(PropTypes.shape({})),
};

RulePermissions.defaultProps = {
    rules: null,
};

export default RulePermissions;
