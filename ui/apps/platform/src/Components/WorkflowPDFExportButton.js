import React, { Component } from 'react';
import PropTypes from 'prop-types';
import html2canvas from 'html2canvas';
import jsPDF from 'jspdf';
import 'jspdf-autotable';
import dateFns from 'date-fns';
import computedStyleToInlineStyle from 'computed-style-to-inline-style';
import Button from 'Components/Button';
import { enhanceWordBreak } from 'utils/pdfUtils';
import { getProductBranding } from 'constants/productBranding';

const printClassName = 'pdf-page';
const imagesClassName = 'pdf-page-image';
const printProperties = [
    'width',
    'height',
    'fill',
    'style',
    'class',
    'stroke',
    'font',
    'font-size',
];
const defaultPageLandscapeWidth = 297;
const defaultPagePortraitWidth = 210;
const WIDGET_WIDTH = 203;

class WorkflowPDFExportButton extends Component {
    static propTypes = {
        id: PropTypes.string,
        options: PropTypes.shape({
            paperSize: PropTypes.string,
            mode: PropTypes.string,
            marginType: PropTypes.string,
        }),
        fileName: PropTypes.string,
        onClick: PropTypes.func,
        className: PropTypes.string,
        tableOptions: PropTypes.shape({}),
        pdfTitle: PropTypes.string,
        isExporting: PropTypes.bool.isRequired,
        setIsExporting: PropTypes.func.isRequired,
    };

    static defaultProps = {
        id: 'capture',
        options: {
            paperSize: 'a4',
            mode: 'p',
            marginType: 'mm',
        },
        tableOptions: null,
        fileName: 'export',
        onClick: null,
        className: '',
        pdfTitle: '',
    };

    beforePDFPrinting = () => {
        const el = document.getElementById(this.props.id);
        const cc = Array.from(el.getElementsByClassName(printClassName));

        const promises = [];
        const div = `<div class="theme-light flex justify-between bg-primary-800 items-center text-primary-100 h-32">
            <img alt="stackrox-logo" src=${getProductBranding().logoSvg} class="h-20 pl-2" />
            <div class="pr-4 text-right">
                <div class="text-2xl">${this.props.pdfTitle}</div>
                <div class="pt-2 text-xl">${dateFns.format(new Date(), 'MM/DD/YYYY')}</div>
            </div>
        </div>`;
        const header = document.createElement('header');
        header.id = 'pdf-header';
        header.innerHTML = div;
        el.insertBefore(header, el.firstChild);
        promises.push(
            html2canvas(header, {
                scale: 3,
            })
        );

        for (let i = 0; i < cc.length; i += 1) {
            const clonedNode = cc[i].cloneNode(true);
            clonedNode.setAttribute('data-class-name', clonedNode.className);
            clonedNode.className = `${clonedNode.className} theme-light pdf-export border border-base-400`;
            cc[i].parentNode.appendChild(clonedNode);
            computedStyleToInlineStyle(clonedNode, {
                recursive: true,
                properties: printProperties,
            });
            cc[i].className = 'pdf-page hidden';

            const promise = html2canvas(clonedNode, {
                scale: 3,
                allowTaint: true,
            }).then((canvas) => {
                Object.assign(canvas, {
                    className: clonedNode.className.replace('pdf-page', 'pdf-page-image'),
                });
                cc[i].parentNode.insertBefore(canvas, clonedNode);
                clonedNode.className = 'clonedNode hidden';
                return canvas;
            });
            promises.push(promise);
        }
        return promises;
    };

    drawTable = (positionY, doc) => {
        const tableOptions = {
            html: '#pdf-table',
            startY: positionY + 2,
            styles: {
                fontSize: 6,
            },
            margin: { left: 3, right: 3 },
            didParseCell: enhanceWordBreak,
            ...this.props.tableOptions,
        };
        doc.autoTable(tableOptions);
    };

    saveFn = () => {
        const { id, options, fileName, setIsExporting, onClick } = this.props;
        setIsExporting(true);
        if (onClick) {
            onClick();
        }
        const { paperSize, mode, marginType } = options;
        const element = document.getElementById(id);
        const imgElements = element.getElementsByClassName(imagesClassName);
        const printElements = Array.from(element.getElementsByClassName(printClassName));

        let imgWidth = options.mode === 'l' ? defaultPageLandscapeWidth : defaultPagePortraitWidth;
        // eslint-disable-next-line new-cap
        const doc = new jsPDF(mode, marginType, paperSize, true);
        let positionX = 0;
        let positionY = 0;
        const pageHeight = doc.internal.pageSize.getHeight();
        let remainingHeight = pageHeight;

        const paddingX = 2;
        const paddingY = 2;
        const pageWidgetWidth = (WIDGET_WIDTH + paddingX) * 2;
        let remainingWidgetWidth = pageWidgetWidth;

        function drawPDF(imgHeight, imgWidgetWidth, index, imgData, canvases) {
            const halfImgWidth = imgWidth / 2;
            const halfImgHeight = imgHeight / 2;

            // beginning of a new row
            if (remainingWidgetWidth < imgWidgetWidth || remainingWidgetWidth === pageWidgetWidth) {
                positionX = paddingX;

                // reset remaining width in widget row
                remainingWidgetWidth = pageWidgetWidth;

                // calculating new row height position
                const prevCanvasHeight = index === 1 ? 0 : canvases[index - 1].height;
                const prevRowHeight = (prevCanvasHeight * imgWidth) / canvases[index].width;
                const halfPrevRowHeight = prevRowHeight / 2;
                positionY += halfPrevRowHeight + paddingY;
                remainingHeight -= halfPrevRowHeight;
            } else {
                // still in same row of widgets, just move x position
                positionX += halfImgWidth + paddingX;
            }

            // beginning of new page
            if (remainingHeight < halfImgHeight || canvases[index].classList.contains('pdf-new')) {
                doc.addPage();
                positionX = paddingX;
                positionY = paddingY;
                remainingHeight = pageHeight;
            }

            // calculating remaining width
            remainingWidgetWidth -= imgWidgetWidth;

            doc.addImage(
                imgData,
                'jpg',
                positionX,
                positionY,
                halfImgWidth,
                halfImgHeight,
                `Image${index}`,
                'FAST'
            );
        }

        function drawStretchPDF(imgHeight, index, imgData) {
            positionX = paddingX;
            if (remainingHeight < imgHeight) {
                doc.addPage();
                positionX = paddingX;
                positionY = paddingY;
                remainingHeight = pageHeight;
            }
            doc.addImage(
                imgData,
                'jpg',
                positionX,
                positionY,
                imgWidth,
                imgHeight,
                `Image${index}`,
                'FAST'
            );
            positionY += imgHeight + paddingY;
            remainingHeight -= imgHeight;
        }

        setTimeout(() => {
            Promise.all(this.beforePDFPrinting()).then((canvases) => {
                const printClonedElements = Array.from(
                    element.getElementsByClassName('clonedNode')
                );
                const header = document.getElementById('pdf-header');
                canvases.forEach((canvas, index) => {
                    const isWidgetsView =
                        id.includes('capture-dashboard') || id.includes('capture-widgets');
                    const imgData = canvas.toDataURL('image/jpeg');
                    if (isWidgetsView && index > 0) {
                        imgWidth = canvas.classList.contains('pdf-stretch')
                            ? (WIDGET_WIDTH + paddingX) * 2
                            : WIDGET_WIDTH;
                    }
                    const imgHeight = (canvas.height * imgWidth) / canvas.width;

                    // for PDF page header
                    if (index === 0) {
                        doc.addImage(
                            imgData,
                            'jpg',
                            positionX,
                            positionY,
                            imgWidth,
                            imgHeight,
                            `Image${index}`,
                            'FAST'
                        );
                        positionY = imgHeight + paddingY;
                        remainingHeight -= imgHeight;
                    } else {
                        if (isWidgetsView) {
                            if (id === 'capture-dashboard' || id === 'capture-widgets') {
                                drawPDF(imgHeight, imgWidth, index, imgData, canvases);
                            } else {
                                drawStretchPDF(imgHeight, index, imgData);
                            }
                        }
                        if (id === 'capture-list') {
                            doc.addImage(
                                imgData,
                                'JPEG',
                                positionX,
                                positionY,
                                imgWidth,
                                imgHeight
                            );
                            positionY += imgHeight;
                            this.drawTable(positionY, doc);
                        }
                    }
                });

                // only header and table
                if (canvases.length === 1) {
                    if (id === 'capture-list') {
                        this.drawTable(positionY, doc);
                    }
                }
                Array.from(imgElements).forEach((el, index) => {
                    printElements[index].className =
                        printClonedElements[index].getAttribute('data-class-name');
                    el.parentNode.removeChild(printClonedElements[index]);
                    el.parentNode.removeChild(el);
                });
                element.removeChild(header);
                doc.save(`${fileName}.pdf`);
                setIsExporting(false);
            });
        }, 0);
    };

    render() {
        return (
            <Button
                isLoading={this.props.isExporting}
                disabled={this.props.isExporting}
                dataTestId="download-pdf-button"
                className={this.props.className}
                text="Download Page as PDF"
                onClick={this.saveFn}
            />
        );
    }
}

export default WorkflowPDFExportButton;
