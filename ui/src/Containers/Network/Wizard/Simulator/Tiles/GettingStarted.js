import React from 'react';

import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import gettingStarted from 'images/getting-started.svg';

export default function GettingStarted({ onClick }) {
    return (
        <section className="bg-base-100 shadow text-base-600 border border-base-200 m-3 mt-4 mb-0 flex flex-col flex-no-shrink">
            <div className="p-3 pb-2 border-b border-base-300 text-primary-600 flex justify-between space-between">
                <h1 className="text-base text-base-600 text-lg font-700">Getting Started</h1>
                <Icon.X
                    className="h-4 w-4 text-base-500 cursor-pointer hover:text-base-600"
                    onClick={onClick}
                />
            </div>
            <div className="pt-3 pr-3 pl-3 self-center">
                <img alt="" src={gettingStarted} />
            </div>
            <div className="m-3 border-t border-dashed border-base-300 pt-3 leading-loose font-600">
                The network simulator allows you to quickly preview your environment under different
                policy configurations. After proper configuration, notify and share the YAML file
                with your team. To get started, upload a YAML file below.
            </div>
        </section>
    );
}

GettingStarted.propTypes = {
    onClick: PropTypes.func.isRequired
};
