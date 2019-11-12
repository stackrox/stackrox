import React from 'react';
import PropTypes from 'prop-types';
import get from 'lodash/get';

import CollapsibleCard from 'Components/CollapsibleCard';
import formDescriptor from './Form/formDescriptor';

const Field = props => {
    const { label, jsonPath, authProvider } = props;
    const value = get(authProvider, jsonPath);
    if (!value) return null;
    return (
        <div className="mb-4">
            <div className="py-2 text-base-600 font-700">{label}</div>
            <div className="w-1/2">{get(authProvider, jsonPath)}</div>
        </div>
    );
};

const Details = props => {
    const { name, type, active } = props.authProvider;
    const { groups, defaultRole } = props;

    if (!name) return null;
    // Add warning about provider not being editable if active
    let warning = '';
    if (active) {
        warning = (
            <div className="w-full justify-between overflow-auto bg-alert-200 p-4 pl-6 font-700">
                Auth Providers that have been logged into cannot be edited. Please delete and
                recreate.
            </div>
        );
    }

    const title = `1. ${name} Configuration`;
    const propsTitle = `2. Assign StackRox roles to your ${name} users`;
    const fields = formDescriptor[type];
    return (
        <div className="w-full justify-between overflow-auto">
            {warning}
            <div className="w-full justify-between overflow-auto p-4">
                <CollapsibleCard
                    title={title}
                    titleClassName="border-b px-1 border-warning-300 leading-normal cursor-pointer flex justify-between items-center bg-warning-200 hover:border-warning-400"
                >
                    <div className="w-full h-full p-4 pt-2 pb-2">
                        {fields &&
                            fields.map((field, index) => (
                                <Field key={index} {...field} authProvider={props.authProvider} />
                            ))}
                    </div>
                </CollapsibleCard>
                <div className="mt-4">
                    <CollapsibleCard
                        title={propsTitle}
                        titleClassName="border-b px-1 border-warning-300 leading-normal cursor-pointer flex justify-between items-center bg-warning-200 hover:border-warning-400"
                    >
                        <div className="flex flex-col">
                            <div className="p-4 w-full">
                                <div className="text-base-600 font-700 pb-2">
                                    Minimum access role
                                </div>
                                <div className="pb-2">{defaultRole}</div>
                                <div className="pb-2">
                                    <p className="pb-2">
                                        The minimum access role is granted to all users who sign in
                                        with {name}.
                                    </p>
                                    <p className="pb-2">
                                        To give users different roles, add rules. Users are granted
                                        all matching roles.
                                    </p>
                                    <p className="pb-2">
                                        Set the minimum access role to <em>None</em> if you want to
                                        define permissions completely using specific rules.
                                    </p>
                                </div>
                            </div>
                            {groups.map((group, idx) => (
                                <div className="p-4 flex w-full" key={idx}>
                                    <div className="w-full">
                                        <div className="text-base-600 font-700 pb-2">Key</div>
                                        <div>{group.props.key}</div>
                                    </div>
                                    <div className="w-full">
                                        <div className="text-base-600 font-700 pb-2">Value</div>
                                        <div>
                                            {group.props.value || (
                                                <span className="italic">Any value</span>
                                            )}
                                        </div>
                                    </div>
                                    <div className="w-full">
                                        <div className="text-base-600 font-700 pb-2">Role</div>
                                        <div>{group.roleName}</div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </CollapsibleCard>
                </div>
            </div>
        </div>
    );
};

Field.propTypes = {
    label: PropTypes.string,
    jsonPath: PropTypes.string,
    authProvider: PropTypes.shape({
        name: PropTypes.string
    }).isRequired
};

Field.defaultProps = {
    label: '',
    jsonPath: ''
};

Details.propTypes = {
    authProvider: PropTypes.shape({
        name: PropTypes.string,
        type: PropTypes.string,
        active: PropTypes.bool
    }).isRequired,
    groups: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    defaultRole: PropTypes.string
};

Details.defaultProps = {
    defaultRole: 'Admin'
};

export default Details;
