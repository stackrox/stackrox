import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import html2canvas from 'html2canvas';
import jsPDF from 'jspdf';
import 'jspdf-autotable';
import dateFns from 'date-fns';
import computedStyleToInlineStyle from 'computed-style-to-inline-style';
import Button from 'Components/Button';
import { actions } from 'reducers/pdfDownload';
import StackroxLogo from 'images/stackrox-logo.png';

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
    'font-size'
];
const defaultPageLandscapeWidth = 297;
const defaultPagePotraitWidth = 210;

class PDFExportButton extends Component {
    static propTypes = {
        id: PropTypes.string,
        options: PropTypes.shape({}),
        fileName: PropTypes.string,
        setPDFRequestState: PropTypes.func,
        setPDFSuccessState: PropTypes.func,
        onClick: PropTypes.func,
        className: PropTypes.string,
        tableOptions: PropTypes.shape({}),
        pdfTitle: PropTypes.string
    };

    static defaultProps = {
        id: 'capture',
        options: {
            paperSize: 'a4',
            mode: 'p',
            marginType: 'mm'
        },
        tableOptions: null,
        fileName: 'export',
        setPDFRequestState: null,
        setPDFSuccessState: null,
        onClick: null,
        className: '',
        pdfTitle: ''
    };

    beforePDFPrinting = () => {
        const el = document.getElementById(this.props.id);
        const cc = Array.from(el.getElementsByClassName(printClassName));

        const promises = [];
        const div = `<div class="theme-light flex justify-between bg-primary-800 items-center text-primary-100 h-32">
            <img alt="stackrox-logo" src=${StackroxLogo} class="h-24" />
            <div class="pr-4 text-right">
                <div class="text-2xl">${this.props.pdfTitle} Report</div>
                <div class="pt-2 text-xl">${dateFns.format(new Date(), 'MM/DD/YYYY')}</div>
            </div>
        </div>`;
        const header = document.createElement('header');
        header.id = 'pdf-header';
        header.innerHTML = div;
        el.insertBefore(header, el.firstChild);
        promises.push(
            html2canvas(header, {
                scale: 3
            })
        );

        for (let i = 0; i < cc.length; i += 1) {
            const clonedNode = cc[i].cloneNode(true);
            clonedNode.setAttribute('data-class-name', clonedNode.className);
            clonedNode.className = `${clonedNode.className} theme-light border border-base-400`;
            cc[i].parentNode.appendChild(clonedNode);
            computedStyleToInlineStyle(clonedNode, {
                recursive: true,
                properties: printProperties
            });
            cc[i].className = 'pdf-page hidden';

            const promise = html2canvas(clonedNode, {
                scale: 3
            }).then(canvas => {
                Object.assign(canvas, {
                    className: clonedNode.className.replace('pdf-page', 'pdf-page-image')
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
        const tableOptions = Object.assign(
            {
                html: '#pdf-table',
                startY: positionY + 2,
                styles: {
                    fontSize: 6
                },
                margin: { left: 3, right: 3 }
            },
            this.props.tableOptions
        );
        doc.autoTable(tableOptions);
    };

    saveFn = () => {
        const {
            id,
            options,
            fileName,
            setPDFRequestState,
            setPDFSuccessState,
            onClick
        } = this.props;
        setPDFRequestState();
        if (onClick) onClick();
        const { paperSize, mode, marginType } = options;
        const element = document.getElementById(id);
        const imgElements = element.getElementsByClassName(imagesClassName);
        const printElements = Array.from(element.getElementsByClassName(printClassName));

        let imgWidth = options.mode === 'l' ? defaultPageLandscapeWidth : defaultPagePotraitWidth;
        // eslint-disable-next-line new-cap
        const doc = new jsPDF(mode, marginType, paperSize, true);
        let positionX = 0;
        let positionY = 0;
        let count = 1;
        const pageHeight = doc.internal.pageSize.getHeight();
        let remainingHeight = pageHeight;

        setTimeout(() => {
            Promise.all(this.beforePDFPrinting()).then(canvases => {
                const printClonedElements = Array.from(
                    element.getElementsByClassName('clonedNode')
                );
                const header = document.getElementById('pdf-header');
                canvases.forEach((canvas, index) => {
                    const imgData = canvas.toDataURL('image/jpeg');
                    if (id === 'capture-dashboard' && index > 0) {
                        imgWidth = 203;
                    }
                    const imgHeight = (canvas.height * imgWidth) / canvas.width;

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
                        positionY = imgHeight + 2;
                        remainingHeight -= imgHeight;
                    } else {
                        if (id === 'capture-dashboard') {
                            const halfImgWidth = imgWidth / 2;
                            const halfImgHeight = imgHeight / 2;

                            if (index % 2 === 1) {
                                positionX = 2;
                            }

                            if (index % 2 === 0) {
                                positionX += halfImgWidth + 2;
                            }

                            if (remainingHeight < halfImgHeight) {
                                doc.addPage();
                                positionX = 2;
                                positionY = 2;
                                count = 1;
                                remainingHeight = pageHeight;
                            }
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
                            count += 1;
                            if (count % 3 === 0) {
                                //  every  2 widgets
                                const firstCanvas = canvases[index - 1].height;
                                const secondCanvas = canvases[index - 2].height;
                                const prevRowMaxHeight =
                                    firstCanvas > secondCanvas ? firstCanvas : secondCanvas;
                                const prevRowHeight =
                                    (prevRowMaxHeight * imgWidth) / canvases[index - 1].width;
                                const halfPrevRowHeight = prevRowHeight / 2;
                                positionY += halfPrevRowHeight + 2;
                                count = 1;
                                remainingHeight -= halfPrevRowHeight;
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
                    printElements[index].className = printClonedElements[index].getAttribute(
                        'data-class-name'
                    );
                    el.parentNode.removeChild(printClonedElements[index]);
                    el.parentNode.removeChild(el);
                });
                element.removeChild(header);
                doc.save(`${fileName}.pdf`);
                setPDFSuccessState();
            });
        }, 0);
    };

    render() {
        return (
            <Button
                data-test-id="download-pdf-button"
                className={this.props.className}
                text="DOWNLOAD PAGE AS PDF"
                onClick={this.saveFn}
            />
        );
    }
}

const mapDispatchToProps = {
    setPDFRequestState: actions.fetchPdf.request,
    setPDFSuccessState: actions.fetchPdf.success
};

export default connect(
    null,
    mapDispatchToProps
)(PDFExportButton);
