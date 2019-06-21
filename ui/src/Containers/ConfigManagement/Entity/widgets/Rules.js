import React from 'react';
import Widget from 'Components/Widget';
import { ArrowRight } from 'react-feather';

const Rules = ({ rules, ...rest }) => {
    const header = `${rules.length} Rules`;
    const verbs = rules.map(rule => {
        return (
            <li className="flex items-center">
                <div className="min-w-48 text-sm bg-base-200 border border-base-400 my-3 p-3 rounded w-full leading-normal">
                    {rule.verbs.join(', ')}
                </div>
                <ArrowRight className="h-4 w-4 text-base-500 mx-4" />
            </li>
        );
    });
    const resourcesAndNonResourcesURL = rules.map(rule => {
        const { nonResourceUrls, resources } = rule;
        return (
            <li className="flex items-center">
                <div className="text-sm bg-base-200 border border-base-400 my-3 p-3 rounded leading-normal">
                    {[...resources, ...nonResourceUrls].join(', ')}
                </div>
            </li>
        );
    });
    return (
        <Widget header={header} {...rest}>
            <div className="flex">
                <div>
                    <h1 className="font-600 border-b border-base-300 text-sm justify-left flex p-2 px-3">
                        Verbs
                    </h1>
                    <ul className="list-reset p-3">{verbs}</ul>
                </div>
                <div>
                    <h1 className="font-600 border-b border-base-300 text-sm justify-left flex p-2 px-3">
                        Resources and Non-resource URLs
                    </h1>
                    <ul className="list-reset p-3">{resourcesAndNonResourcesURL}</ul>
                </div>
            </div>
        </Widget>
    );
};

export default Rules;
