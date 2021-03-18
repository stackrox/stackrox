import React, { ReactElement } from 'react';

type RadioButtonGroupProps = {
    headerText?: string;
    buttons: {
        text: string;
        value: boolean | string;
    }[];
    selected?: boolean | string;
    onClick: (value) => void;
    groupClassName?: string;
    testId?: string;
    useBoolean?: boolean;
    disabled?: boolean;
};

const RadioButtonGroup = ({
    headerText,
    buttons,
    selected,
    onClick,
    groupClassName = '',
    testId = 'radio-button-group',
    useBoolean,
    disabled,
}: RadioButtonGroupProps): ReactElement => {
    function onClickHandler(data) {
        const targetValue = data.target.getAttribute('value');
        if (targetValue) {
            const value = useBoolean ? targetValue === 'true' : targetValue.toString();
            onClick(value);
        }
    }
    const content = buttons.map(({ text, value }, index) => {
        let modifiedValue = text;
        if (value !== undefined) {
            modifiedValue = String(value);
        }
        return (
            <button
                key={text}
                type="button"
                className={`flex flex-1 justify-center items-center px-2 text-sm font-600 font-condensed text-base-600 hover:text-primary-600 uppercase ${
                    index !== 0 ? 'border-l border-base-400' : ''
                } ${
                    selected === modifiedValue
                        ? 'bg-primary-200 text-primary-700 hover:text-primary-700 hover:bg-primary-200'
                        : 'hover:bg-base-200 bg-base-100'
                }`}
                onClick={onClickHandler}
                value={modifiedValue}
                disabled={disabled}
            >
                {text}
            </button>
        );
    });
    return (
        <div
            className={`text-xs flex flex-col uppercase rounded border-2 h-10 border-base-400 text-center font-condensed text-base-600 font-600 ${
                groupClassName || ''
            }`}
            data-testid={testId}
        >
            {headerText && (
                <div className="bg-base-100 border-b-2 border-base-400 px-2 text-base-500">
                    {headerText}
                </div>
            )}
            <div className="flex h-full">{content}</div>
        </div>
    );
};

export default RadioButtonGroup;
