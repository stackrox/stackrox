import React from 'react';
import PropTypes from 'prop-types';

import ReduxTextAreaField from 'Components/forms/ReduxTextAreaField';
import ReduxSelectField from 'Components/forms/ReduxSelectField';
import ReduxToggleField from 'Components/forms/ReduxToggleField';
import ReduxColorPickerField from 'Components/forms/ReduxColorPickerField';

import { keyClassName } from './Page';

const backgroundSizeOptions = [
    {
        label: 'Small',
        value: 'SMALL'
    },
    {
        label: 'Medium',
        value: 'MEDIUM'
    },
    {
        label: 'Large',
        value: 'LARGE'
    }
];

const ConfigBannerFormWidget = ({ type }) => (
    <div className="px-3 w-full" data-test-id={`${type}-config`}>
        <div className="bg-base-100 border-base-200 shadow">
            <div className="py-2 px-4 border-b border-base-300 text-base-600 font-700 text-lg capitalize flex justify-between items-center">
                {`${type} configuration`}
                <ReduxToggleField name={`publicConfig.${type}.enabled`} />
            </div>

            <div className="flex flex-col pt-2 pb-4 px-4 w-full">
                <div className="flex w-full justify-between">
                    <div className="w-full pr-4">
                        <div className={keyClassName}>Text (2000 character limit):</div>
                        <ReduxTextAreaField
                            name={`publicConfig.${type}.text`}
                            placeholder={`Place ${type} text here...`}
                            maxLength="2000"
                        />
                    </div>
                    <div className="w-1/6">
                        <div className={keyClassName}>Text Color:</div>
                        <ReduxColorPickerField name={`publicConfig.${type}.color`} />
                    </div>
                </div>
                <div className="border-base-300 border-t flex justify-between mt-6 pt-4 w-full">
                    <div className="w-full pr-4">
                        <div className={keyClassName}>{`${type} Size:`}</div>
                        <ReduxSelectField
                            name={`publicConfig.${type}.size`}
                            options={backgroundSizeOptions}
                        />
                    </div>
                    <div className="w-1/6">
                        <div className={keyClassName}>Background Color:</div>
                        <ReduxColorPickerField name={`publicConfig.${type}.backgroundColor`} />
                    </div>
                </div>
            </div>
        </div>
    </div>
);

ConfigBannerFormWidget.propTypes = {
    type: PropTypes.string.isRequired
};

export default ConfigBannerFormWidget;
