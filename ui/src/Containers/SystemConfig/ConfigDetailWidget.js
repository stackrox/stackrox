import React from 'react';
import PropTypes from 'prop-types';

import capitalize from 'lodash/capitalize';
import ColorPicker from 'Components/ColorPicker';
import { keyClassName } from './Page';

const ConfigDetailWidget = ({ type, config }) => {
    const { publicConfig } = config;
    function getValue(key) {
        return !publicConfig || !publicConfig[type] || !publicConfig[type][key]
            ? 'None'
            : publicConfig[type][key];
    }

    return (
        <div className="px-3 pt-5 w-full">
            <div className="bg-base-100 border-base-200 shadow">
                <div className="py-2 px-4 border-b border-base-300 text-base-600 font-700 text-lg capitalize flex justify-between items-center h-10">
                    {type} Configuration{' '}
                    <div>
                        {publicConfig && publicConfig[type] && publicConfig[type].enabled
                            ? 'enabled'
                            : 'disabled'}
                    </div>
                </div>

                <div className="flex flex-col pt-2 pb-4 px-4 w-full">
                    <div className="flex w-full justify-between">
                        <div className="w-full pr-4 whitespace-pre">
                            <div className={keyClassName}>Text (2000 character limit):</div>
                            {getValue('text')}
                        </div>
                        <div className="w-1/6">
                            <div className={keyClassName}>Text Color:</div>
                            <ColorPicker
                                color={
                                    publicConfig && publicConfig[type] && publicConfig[type].color
                                }
                                disabled
                            />
                            {getValue('color')}
                        </div>
                    </div>
                    <div className="border-base-300 border-t flex justify-between mt-6 pt-4 w-full">
                        <div className="w-full pr-4">
                            <div className={keyClassName}>Text Size</div>
                            <div>{capitalize(getValue('size'))}</div>
                        </div>
                        <div className="w-1/6">
                            <div className={keyClassName}>Background Color:</div>
                            <ColorPicker
                                color={
                                    publicConfig &&
                                    publicConfig[type] &&
                                    publicConfig[type].backgroundColor
                                }
                                disabled
                            />
                            {getValue('backgroundColor')}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};

ConfigDetailWidget.propTypes = {
    type: PropTypes.string.isRequired,
    config: PropTypes.shape({
        publicConfig: PropTypes.shape({
            header: PropTypes.shape({}),
            footer: PropTypes.shape({}),
            loginNotice: PropTypes.shape({})
        })
    }).isRequired
};

export default ConfigDetailWidget;
