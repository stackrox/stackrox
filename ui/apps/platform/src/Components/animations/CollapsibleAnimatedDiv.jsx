import React from 'react';
import PropTypes from 'prop-types';
import { motion, AnimatePresence } from 'framer-motion';

const variants = {
    open: { opacity: 1, height: 'auto' },
    collapsed: { opacity: 0, height: 0 },
};

const transition = {
    ease: 'easeInOut',
};

function CollapsibleAnimatedDiv({ dataTestId, defaultOpen, isOpen, children }) {
    return (
        <AnimatePresence initial={defaultOpen}>
            {isOpen && (
                <motion.div
                    data-testid={dataTestId}
                    key="content"
                    initial="collapsed"
                    animate="open"
                    exit="collapsed"
                    variants={variants}
                    transition={transition}
                >
                    {children}
                </motion.div>
            )}
        </AnimatePresence>
    );
}

CollapsibleAnimatedDiv.propTypes = {
    dataTestId: PropTypes.string,
    defaultOpen: PropTypes.bool,
    isOpen: PropTypes.bool.isRequired,
    children: PropTypes.node.isRequired,
};

CollapsibleAnimatedDiv.defaultProps = {
    dataTestId: null,
    defaultOpen: true,
};

export default CollapsibleAnimatedDiv;
