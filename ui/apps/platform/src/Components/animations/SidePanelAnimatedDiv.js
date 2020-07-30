import React from 'react';
import PropTypes from 'prop-types';
import { motion, AnimatePresence } from 'framer-motion';

const variants = {
    open: { x: 0 },
    closed: { x: '100%' },
};

const transition = {
    ease: 'easeInOut',
    transition: 2,
};

const SidePanelAnimatedDiv = ({ defaultOpen, isOpen, children }) => {
    return (
        <AnimatePresence initial={defaultOpen}>
            {isOpen && (
                <motion.div
                    className="absolute z-10 h-full top-0 right-0 w-full lg:w-9/10"
                    initial="closed"
                    animate="open"
                    exit="closed"
                    variants={variants}
                    transition={transition}
                >
                    {children}
                </motion.div>
            )}
        </AnimatePresence>
    );
};

SidePanelAnimatedDiv.propTypes = {
    defaultOpen: PropTypes.bool,
    isOpen: PropTypes.bool.isRequired,
    children: PropTypes.element.isRequired,
};

SidePanelAnimatedDiv.defaultProps = {
    defaultOpen: false,
};

export default SidePanelAnimatedDiv;
