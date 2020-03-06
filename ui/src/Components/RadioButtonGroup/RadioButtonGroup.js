import React from 'react';
import PropTypes from 'prop-types';

const RadioButtonGroup = ({ headerText, buttons, selected, onClick }) => {
    function onClickHandler(data) {
        const value = data.target.getAttribute('value');
        onClick(value);
    }

    const content = buttons.map(({ text }, index) => {
        return (
            <button
                key={text}
                type="button"
                className={`flex flex-1 justify-center py-1 px-2 text-sm font-600 font-condensed text-base-600 hover:text-primary-600 uppercase ${
                    index !== 0 ? 'border-l border-base-300' : ''
                } ${
                    selected === text
                        ? 'bg-primary-200 text-primary-700 hover:text-primary-700 hover:bg-primary-200'
                        : 'hover:bg-base-200 bg-base-100'
                }`}
                onClick={onClickHandler}
                value={text}
            >
                {text}
            </button>
        );
    });
    return (
        <div className="text-xs flex flex-col uppercase rounded border-2 h-10 border-base-300 text-center font-condensed text-base-600 font-600">
            <div className="bg-base-100 border-b-2 border-base-300 px-2 text-base-500">
                {headerText}
            </div>
            <div className="flex h-full">{content}</div>
        </div>
    );
};

RadioButtonGroup.propTypes = {
    headerText: PropTypes.string.isRequired,
    buttons: PropTypes.arrayOf(
        PropTypes.shape({
            text: PropTypes.string.isRequired
        })
    ).isRequired,
    selected: PropTypes.string,
    onClick: PropTypes.func.isRequired
};

RadioButtonGroup.defaultProps = {
    selected: null
};

export default RadioButtonGroup;
