import React from 'react';

import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import ReactDropzone from 'react-dropzone';

export default function DragAndDrop({ message, onDrop }) {
    return (
        <section
            data-test-id="upload-yaml-panel"
            className="bg-base-100 min-h-32 m-3 mt-4 mb-0 flex h-full border border-dashed border-base-300 hover:border-base-500 cursor-pointer"
        >
            <ReactDropzone
                onDrop={onDrop}
                className="flex w-full h-full flex-col self-center uppercase p-5 hover:bg-warning-100 shadow justify-center"
            >
                <div
                    className="h-18 w-18 self-center rounded-full flex items-center justify-center flex-no-shrink"
                    style={{ background: '#faecd2', color: '#b39357' }}
                >
                    <Icon.Upload className="h-8 w-8" strokeWidth="1.5px" />
                </div>

                <div className="text-center pt-5 font-700">{message}</div>
            </ReactDropzone>
        </section>
    );
}

DragAndDrop.propTypes = {
    message: PropTypes.string.isRequired,
    onDrop: PropTypes.func.isRequired
};
