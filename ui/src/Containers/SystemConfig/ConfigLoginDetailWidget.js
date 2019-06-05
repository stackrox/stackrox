import React from 'react';
import PropTypes from 'prop-types';
import { keyClassName } from './Page';

const ConfigLoginDetailWidget = ({ config }) => {
    const { publicConfig } = config;

    return (
        <div className="bg-base-100 border-base-200 shadow" data-test-id="login-notice-config">
            <div className="py-2 px-4 border-b border-base-300 text-base-600 font-700 text-lg capitalize flex justify-between items-center h-10">
                Login Notice Configuration{' '}
                <div data-test-id="login-notice-state">
                    {publicConfig && publicConfig.loginNotice && publicConfig.loginNotice.enabled
                        ? 'enabled'
                        : 'disabled'}
                </div>
            </div>

            <div className="flex flex-col pt-2 pb-4 px-4 w-full">
                <div className="w-full pr-4 whitespace-pre-wrap">
                    <div className={keyClassName}>Text (2000 character limit):</div>
                    {publicConfig && publicConfig.loginNotice && publicConfig.loginNotice.text
                        ? publicConfig.loginNotice.text
                        : 'None'}
                </div>
            </div>
        </div>
    );
};

ConfigLoginDetailWidget.propTypes = {
    config: PropTypes.shape({
        publicConfig: PropTypes.shape({
            loginNotice: PropTypes.shape({})
        })
    }).isRequired
};

export default ConfigLoginDetailWidget;
