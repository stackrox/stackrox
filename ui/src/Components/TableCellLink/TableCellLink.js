import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';

const TableCellLink = ({ url, text, component, pdf, extraClasses, dataTestId }) => {
    function onClick(e) {
        e.stopPropagation();
    }

    // This field is necessary to exclude rendering the Link during PDF generation. It causes an error where the Link can't be rendered outside a Router
    if (pdf) {
        return text;
    }

    return (
        <Link
            to={url}
            className={`underline h-full text-left items-center flex text-base-700 hover:text-primary-700 ${extraClasses}`}
            onClick={onClick}
            data-testid={dataTestId}
        >
            {component || text}
        </Link>
    );
};

TableCellLink.propTypes = {
    component: PropTypes.element,
    text: PropTypes.string,
    url: PropTypes.string.isRequired,
    pdf: PropTypes.bool,
    extraClasses: PropTypes.string,
    dataTestId: PropTypes.string,
};

TableCellLink.defaultProps = {
    component: null,
    text: null,
    pdf: false,
    extraClasses: '',
    dataTestId: null,
};

export default TableCellLink;
