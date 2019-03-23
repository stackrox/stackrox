import React, { useRef } from 'react';
import PropTypes from 'prop-types';
import { toast } from 'react-toastify';
import Button from 'Components/Button';

const UploadButton = ({ onChange, validExtensions, ...props }) => {
    const inputRef = useRef(null);
    const onClickHandler = () => () => {
        inputRef.current.click();
    };
    const onChangeHandler = () => () => {
        const url = inputRef.current.value;
        const ext = url.substring(url.lastIndexOf('.') + 1).toLowerCase();
        const { files } = inputRef.current;
        const extensionIsValid = validExtensions ? validExtensions.includes(ext) : true;
        if (files && files[0] && extensionIsValid) {
            const reader = new FileReader();
            reader.onload = e => {
                onChange(e.target.result);
                inputRef.current.value = '';
            };
            reader.onerror = error => {
                toast(error);
            };
            reader.readAsText(files[0]);
        } else {
            toast('Invalid file format');
        }
    };
    return (
        <>
            <input ref={inputRef} type="file" className="hidden" onChange={onChangeHandler()} />
            <Button {...props} onClick={onClickHandler()} />
        </>
    );
};

UploadButton.propTypes = {
    className: PropTypes.string.isRequired,
    icon: PropTypes.element,
    text: PropTypes.string,
    textCondensed: PropTypes.string,
    textClass: PropTypes.string,
    disabled: PropTypes.bool,
    onChange: PropTypes.func.isRequired,
    validExtensions: PropTypes.arrayOf(PropTypes.string)
};

UploadButton.defaultProps = {
    icon: null,
    text: null,
    textCondensed: null,
    textClass: null,
    disabled: false,
    validExtensions: null
};

export default UploadButton;
