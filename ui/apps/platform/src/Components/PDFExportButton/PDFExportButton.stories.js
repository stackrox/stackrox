import React from 'react';

import PDFExportButton from './PDFExportButton';

export default {
    title: 'PDFExportButton',
    component: PDFExportButton,
};

function doNothing() {}

const DOMElement = () => {
    return (
        <div id="capture-dom-element" className="flex items-center justify-center mx-4 h-48">
            <div className="flex items-center mr-4">Here are some of my favorite shapes: </div>
            <svg width="100" height="100">
                <rect
                    fill="var(--primary-500)"
                    stroke="var(--primary-600)"
                    transform="translate(0,0)"
                    width="100%"
                    height="100%"
                />
            </svg>
            <svg width="100" height="100">
                <polygon
                    points="0,100 50,0 100,100"
                    fill="var(--secondary-500)"
                    stroke="var(--secondary-600)"
                />
            </svg>
            <svg width="100" height="100">
                <circle
                    cx="50%"
                    cy="50%"
                    r="50"
                    stroke="var(--tertiary-600)"
                    fill="var(--tertiary-500)"
                />
            </svg>
        </div>
    );
};

export const withLabel = () => {
    return (
        <>
            <div className="mb-4">
                <DOMElement />
            </div>
            <PDFExportButton
                fileName="DOM Element Report"
                pdfId="capture-dom-element"
                startExportingPDF={doNothing}
                finishExportingPDF={doNothing}
            />
        </>
    );
};
