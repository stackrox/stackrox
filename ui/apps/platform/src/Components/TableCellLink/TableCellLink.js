import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';

const TableCellLink = ({ url, children, pdf, extraClasses, testid }) => {
    function onClick(e) {
        e.stopPropagation();
    }

    // This field is necessary to exclude rendering the Link during PDF generation. It causes an error where the Link can't be rendered outside a Router
    if (pdf) {
        return children;
    }

    return (
        <Link
            to={url}
            className={`underline h-full text-left items-center flex text-base-600 hover:text-primary-700 ${extraClasses}`}
            onClick={onClick}
            data-testid={testid}
        >
            {children}
        </Link>
    );
};

TableCellLink.propTypes = {
    children: PropTypes.node.isRequired,
    url: PropTypes.string.isRequired,
    pdf: PropTypes.bool,
    extraClasses: PropTypes.string,
    testid: PropTypes.string,
};

TableCellLink.defaultProps = {
    pdf: false,
    extraClasses: '',
    testid: null,
};

export default TableCellLink;
