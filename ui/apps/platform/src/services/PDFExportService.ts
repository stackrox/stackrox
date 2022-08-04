import computedStyleToInlineStyle from 'computed-style-to-inline-style';
import JSPDF from 'jspdf';
import logError from 'utils/logError';
import { toast } from 'react-toastify';
import html2canvas from 'html2canvas';

import { getDate, addBrandedTimestampToString } from 'utils/dateUtils';
import { RequestAction, SuccessAction } from 'utils/fetchingReduxRoutines';
import { getProductBranding } from 'constants/productBranding';

/**
 * Creates a container div HTML element that will wrap around all the content to be exported
 * @returns {HTMLElement}
 */
function createPDFContainerElement() {
    const pdfContainer = document.createElement('div');
    pdfContainer.id = 'pdf-container';
    pdfContainer.className = 'flex flex-1 flex-col h-full -z-1 absolute top-0 left-0 theme-light';
    return pdfContainer;
}

/**
 * Creates a header HTML element that will contain the product logo, PDF title, and the current time
 *  @param {string} pdfTitle - The title to display in the top right section of the header
 *  @param {string} timestamp - The timestamp to display in the top right section of the header
 *  @param {string} logoSrc - The full path to the logo image to use as part of the header.
 *  @param {string} logoAlt - Alt text for the provided logo
 *  @returns {HTMLElement}
 */
function createPDFHeaderElement(
    pdfTitle: string,
    timestamp: string,
    logoSrc: string,
    logoAlt: string
) {
    const div = `<div class="theme-light flex justify-between bg-primary-800 items-center text-primary-100 h-32">
            <img alt=${logoAlt} src=${logoSrc} class="h-20 pl-2" />
            <div class="pr-4 text-right">
                <div class="text-2xl">${pdfTitle}</div>
                <div class="pt-2 text-xl">${timestamp}</div>
            </div>
        </div>`;
    const header = document.createElement('header');
    header.id = 'pdf-header';
    header.innerHTML = div;
    return header;
}

/**
 * Creates a div HTML element that will contain the content being exported
 * @returns {HTMLElement}
 */
function createPDFBodyElement() {
    const body = document.createElement('div');
    body.id = 'pdf-body';
    body.className = 'flex flex-1 border-b border-base-300 -z-1';
    return body;
}

/**
 * Converts an HTML element's computed CSS to inline CSS
 * @param {HTMLElement} element
 */
function computeStyles(element) {
    const isThemeDark = document.body.className.includes('theme-dark');

    // if dark mode is enabled, we want to switch to light mode for exporting to PDF
    if (isThemeDark) {
        document.body.classList.remove('theme-dark');
        document.body.classList.add('theme-light');
    }

    computedStyleToInlineStyle(element, {
        recursive: true,
        properties: ['width', 'height', 'fill', 'style', 'class', 'stroke', 'font', 'font-size'],
    });

    // if dark mode was previously enabled, we want to switch back after styles are computed
    if (isThemeDark) {
        document.body.classList.remove('theme-light');
        document.body.classList.add('theme-dark');
    }
}

/**
 * Adds an element to the Root Node
 *  @param {HTMLElement} element
 */
function addElementToRootNode(element) {
    const root = document.getElementById('root');
    if (!root) {
        throw new Error('Expected DOM to contain element with id "root"');
    }
    root.appendChild(element);
}

/**
 * Removes an element from the Root Node
 *  @param {HTMLElement} element
 */
function removeElementFromRootNode(element) {
    if (element?.parentNode) {
        element.parentNode.removeChild(element);
    }
}

/**
 *  Converts a Canvas element -> PNG -> PDF
 *  @param {HTMLElement} canvas
 *  @param {string} pdfFileName - The PDF file name
 */
function savePDF(canvas, pdfFileName) {
    const pdf = new JSPDF();
    const imgData = canvas.toDataURL('image/png');

    // we want the width to be 100% of the PDF page, but the height to scale within the w/h ratio of the Canvas element
    const imgProps = pdf.getImageProperties(imgData);
    const pdfWidth = pdf.internal.pageSize.getWidth();
    const pdfHeight = (imgProps.height * pdfWidth) / imgProps.width;

    pdf.addImage(imgData, 'PNG', 0, 0, pdfWidth, pdfHeight);
    pdf.save(pdfFileName);
}

function exportPDF(
    fileName: string,
    pdfId: string,
    startExportingPDF: RequestAction,
    finishExportingPDF: SuccessAction
) {
    const branding = getProductBranding();
    // This hides all the pdf generation behind an exporting screen
    startExportingPDF();

    const pdfTitle = `${branding.basePageTitle} ${fileName}`;
    const currentTimestamp = getDate(new Date());
    const pdfFileName = addBrandedTimestampToString(fileName);

    // creates a container element that will include everything necessary to convert to a PDF
    const pdfContainerElement = createPDFContainerElement();

    const pdfHeaderElement = createPDFHeaderElement(
        pdfTitle,
        currentTimestamp,
        branding.logoSvg,
        branding.logoAltText
    );
    pdfContainerElement.appendChild(pdfHeaderElement);

    // create a clone of the element to be exported and add it to the body of the container
    const pdfBodyElement = createPDFBodyElement();
    const elementToBeExported = document.getElementById(pdfId);
    if (!elementToBeExported) {
        throw new Error(`Expected to find DOM element with id ${pdfId}`);
    }
    const clonedElementToBeExported = elementToBeExported.cloneNode(true);
    pdfBodyElement.appendChild(clonedElementToBeExported);
    pdfContainerElement.appendChild(pdfBodyElement);

    // we need to add the container element to the DOM in order to compute the styles and eventually convert it from HTML -> Canvas -> PNG -> PDF
    addElementToRootNode(pdfContainerElement);

    // we need to compute styles into inline styles in order for html2canvas to properly work
    computeStyles(pdfBodyElement);

    // convert HTML -> Canvas
    html2canvas(pdfContainerElement, {
        scale: 1,
        allowTaint: true,
    })
        .then((canvas) => {
            // convert Canvas -> PNG -> PDF
            savePDF(canvas, pdfFileName);
            // Remember to clean up after yourself. This makes sure to remove any added elements to the DOM after they're used
            removeElementFromRootNode(pdfContainerElement);
            // remove the exporting screen
            finishExportingPDF();
        })
        .catch((error) => {
            logError(error);
            finishExportingPDF();
            toast('An error occurred while exporting. Please try again.');
        });
}

export default exportPDF;
