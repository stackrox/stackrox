import React from 'react';
import PropTypes from 'prop-types';

import Widget from 'Components/Widget';
import { ArrowRight } from 'react-feather';
import NoResultsMessage from 'Components/NoResultsMessage';

const Rules = ({ rules, ...rest }) => {
    let content = <NoResultsMessage message="No rules" className="p-6" />;
    let header = '0 Rules';
    if (rules && rules.length > 0) {
        header = `${rules.length > 0 ? rules.length : ''} Rules`;
        const verbs = rules.map((rule, i) => {
            return (
                // eslint-disable-next-line react/no-array-index-key
                <li className="flex items-center" key={i}>
                    <div className="min-w-48 text-sm bg-base-200 border border-base-400 my-3 p-3 rounded w-full leading-normal">
                        {rule.verbs.includes('*') ? '* (All verbs)' : rule.verbs.join(', ')}
                    </div>
                    <ArrowRight className="h-4 w-4 text-base-500 mx-4" />
                </li>
            );
        });
        const resourcesAndNonResourcesURL = rules.map((rule, i) => {
            const { nonResourceUrls, resources } = rule;
            const urls = [...resources, ...nonResourceUrls];
            return (
                // eslint-disable-next-line react/no-array-index-key
                <li className="flex items-center" key={i}>
                    <div className="text-sm bg-base-200 border border-base-400 my-3 p-3 rounded leading-normal">
                        {urls.includes('*') ? '* (All resources)' : urls.join(', ')}
                    </div>
                </li>
            );
        });
        content = (
            <div className="flex">
                <div>
                    <h1 className="font-700 border-b border-base-300 text-sm justify-left flex p-2 px-3">
                        Verbs
                    </h1>
                    <ul className="p-3">{verbs}</ul>
                </div>
                <div>
                    <h1 className="font-700 border-b border-base-300 text-sm justify-left flex p-2 px-3">
                        Resources and Non-resource URLs
                    </h1>
                    <ul className="p-3">{resourcesAndNonResourcesURL}</ul>
                </div>
            </div>
        );
    }

    return (
        <Widget header={header} {...rest}>
            {content}
        </Widget>
    );
};

Rules.propTypes = {
    rules: PropTypes.arrayOf(PropTypes.shape({})),
};

Rules.defaultProps = {
    rules: null,
};

export default Rules;
