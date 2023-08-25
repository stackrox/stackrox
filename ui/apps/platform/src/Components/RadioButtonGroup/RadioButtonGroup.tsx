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
                className={`flex flex-1 justify-center items-center px-2 text-base-600 ${
                    index !== 0 ? 'border-l border-base-400' : ''
                } ${selected === modifiedValue ? 'bg-primary-200 font-700' : ''}`}
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
            className={`text-sm flex flex-col rounded border-2 border-base-400 text-center bg-base-100 text-base-600 ${
                groupClassName || ''
            }`}
            data-testid={testId}
        >
            {headerText && <div className="border-b-2 border-base-400 px-2">{headerText}</div>}
            <div className="flex h-full">{content}</div>
        </div>
    );
};

export default RadioButtonGroup;
