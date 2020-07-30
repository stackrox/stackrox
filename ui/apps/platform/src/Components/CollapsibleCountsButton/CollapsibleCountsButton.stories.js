/* eslint-disable react-hooks/rules-of-hooks */
import React, { useState } from 'react';
import { MessageSquare, Tag } from 'react-feather';

import IconWithCount from 'Components/IconWithCount';
import CollapsibleCountsButton from './CollapsibleCountsButton';

export default {
    title: 'CollapsibleCountsButton',
    component: CollapsibleCountsButton,
};

export const withNoCounts = () => {
    const [isOpen, setOpen] = useState(false);
    function onClickHandler() {
        setOpen(!isOpen);
    }
    return <CollapsibleCountsButton isOpen={isOpen} onClick={onClickHandler} />;
};

export const withCounts = () => {
    const [isOpen, setOpen] = useState(false);
    function onClickHandler() {
        setOpen(!isOpen);
    }
    return (
        <CollapsibleCountsButton isOpen={isOpen} onClick={onClickHandler}>
            <IconWithCount Icon={MessageSquare} count={10} />
            <IconWithCount Icon={Tag} count={25} />
        </CollapsibleCountsButton>
    );
};

export const withLoadingCounts = () => {
    const [isOpen, setOpen] = useState(false);
    function onClickHandler() {
        setOpen(!isOpen);
    }
    return (
        <CollapsibleCountsButton isOpen={isOpen} onClick={onClickHandler}>
            <IconWithCount Icon={MessageSquare} count={120} isLoading />
            <IconWithCount Icon={Tag} count={50} isLoading />
        </CollapsibleCountsButton>
    );
};
