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
                className={`flex flex-1 justify-center p-2 px-4 text-sm font-600 font-condensed text-base-600 hover:text-primary-600 hover:bg-base-200 bg-base-100 uppercase ${
                    index !== 0 ? 'border-l border-primary-300' : ''
                } ${selected === text ? 'bg-base-200 text-primary-600' : ''}`}
                onClick={onClickHandler}
                value={text}
            >
                {text}
            </button>
        );
    });
    return (
        <div className="inline-block text-sm uppercase rounded border border-primary-300 text-center font-condensed text-base-600 font-600">
            <div className="bg-base-100 border-b border-primary-300 px-2 py-1">{headerText}</div>
            <div className="flex">{content}</div>
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
